package justwatch

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"p2pollo/internal/catalog"
)

const (
	apiURL       = "https://apis.justwatch.com/graphql"
	imageBaseURL = "https://images.justwatch.com"
	pageSize     = 20
)

// Client es el cliente para la API GraphQL de JustWatch
type Client struct {
	http *http.Client
}

// New crea un nuevo cliente JustWatch
func New() *Client {
	return &Client{
		http: &http.Client{Timeout: 15 * time.Second},
	}
}

// --- Tipos internos para el API GraphQL ---

type gqlRequest struct {
	Query string `json:"query"`
}

type gqlResponse struct {
	Data   *gqlData `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

type gqlData struct {
	PopularTitles gqlConnection `json:"popularTitles"`
}

type gqlConnection struct {
	Edges []gqlEdge `json:"edges"`
}

type gqlEdge struct {
	Node gqlNode `json:"node"`
}

type gqlNode struct {
	ID         string     `json:"id"`
	ObjectType string     `json:"objectType"`
	Content    gqlContent `json:"content"`
}

type gqlContent struct {
	Title               string `json:"title"`
	OriginalReleaseYear int    `json:"originalReleaseYear"`
	PosterURL           string `json:"posterUrl"`
	ExternalIDs         struct {
		ImdbID string `json:"imdbId"`
	} `json:"externalIds"`
	Genres []struct {
		Translation string `json:"translation"`
	} `json:"genres"`
	Scoring struct {
		ImdbScore *float64 `json:"imdbScore"`
	} `json:"scoring"`
}

// PopularMovies retorna películas populares de JustWatch (página 1-indexed)
func (c *Client) PopularMovies(page int) ([]catalog.Movie, error) {
	nodes, err := c.fetchPopular(page, "MOVIE", "Movie")
	if err != nil {
		return nil, err
	}

	movies := make([]catalog.Movie, 0, len(nodes))
	for _, node := range nodes {
		content := node.Content
		if content.PosterURL == "" {
			continue
		}

		genreList := make([]string, len(content.Genres))
		for i, g := range content.Genres {
			genreList[i] = g.Translation
		}

		rating := 0.0
		if content.Scoring.ImdbScore != nil {
			rating = *content.Scoring.ImdbScore
		}

		movies = append(movies, catalog.Movie{
			ID:        node.ID,
			ImdbID:    content.ExternalIDs.ImdbID,
			Title:     content.Title,
			Year:      content.OriginalReleaseYear,
			Rating:    rating,
			PosterURL: imageBaseURL + content.PosterURL,
			Genres:    strings.Join(genreList, ", "),
		})
	}

	return movies, nil
}

// PopularSeries retorna series populares de JustWatch (página 1-indexed)
func (c *Client) PopularSeries(page int) ([]catalog.Series, error) {
	nodes, err := c.fetchPopular(page, "SHOW", "Show")
	if err != nil {
		return nil, err
	}

	series := make([]catalog.Series, 0, len(nodes))
	for _, node := range nodes {
		content := node.Content
		if content.PosterURL == "" {
			continue
		}

		genreList := make([]string, len(content.Genres))
		for i, g := range content.Genres {
			genreList[i] = g.Translation
		}

		rating := 0.0
		if content.Scoring.ImdbScore != nil {
			rating = *content.Scoring.ImdbScore
		}

		series = append(series, catalog.Series{
			ID:        node.ID,
			ImdbID:    content.ExternalIDs.ImdbID,
			Title:     content.Title,
			Year:      content.OriginalReleaseYear,
			Rating:    rating,
			PosterURL: imageBaseURL + content.PosterURL,
			Genres:    strings.Join(genreList, ", "),
		})
	}

	return series, nil
}

// fetchPopular obtiene títulos populares del API GraphQL de JustWatch
// objectType: "MOVIE" o "SHOW", fragmentType: "Movie" o "Show"
func (c *Client) fetchPopular(page int, objectType, fragmentType string) ([]gqlNode, error) {
	afterClause := ""
	if page > 1 {
		// El cursor de JustWatch es base64 del índice del ítem (1-indexed)
		cursor := base64.StdEncoding.EncodeToString([]byte(strconv.Itoa((page-1) * pageSize)))
		afterClause = fmt.Sprintf(`, after: "%s"`, cursor)
	}

	query := fmt.Sprintf(`{
		popularTitles(country: US, first: %d, filter: {objectTypes: [%s]}%s) {
			edges {
				node {
					objectType
					... on %s {
						id
						content(country: US, language: "en") {
							title
							originalReleaseYear
							posterUrl(profile: S332, format: JPG)
							externalIds { imdbId }
							genres { translation(language: "en") }
							scoring { imdbScore }
						}
					}
				}
			}
		}
	}`, pageSize, objectType, afterClause, fragmentType)

	body, err := c.doPost(query)
	if err != nil {
		return nil, err
	}

	var resp gqlResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("error parseando respuesta JustWatch: %w", err)
	}

	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("JustWatch GraphQL error: %s", resp.Errors[0].Message)
	}

	if resp.Data == nil {
		return nil, fmt.Errorf("respuesta vacía de JustWatch")
	}

	nodes := make([]gqlNode, 0, len(resp.Data.PopularTitles.Edges))
	for _, edge := range resp.Data.PopularTitles.Edges {
		nodes = append(nodes, edge.Node)
	}

	return nodes, nil
}

// doPost realiza una petición POST al endpoint GraphQL de JustWatch
func (c *Client) doPost(query string) ([]byte, error) {
	reqBody, err := json.Marshal(gqlRequest{Query: query})
	if err != nil {
		return nil, fmt.Errorf("error serializando request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("error creando request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error en request HTTP: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status HTTP %d de JustWatch API", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error leyendo respuesta: %w", err)
	}

	return body, nil
}
