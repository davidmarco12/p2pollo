package tvmaze

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

const baseURL = "https://api.tvmaze.com"

// Client es el cliente para la API de TVmaze (gratuita, sin API key)
type Client struct {
	http *http.Client
}

// New crea un nuevo cliente TVmaze
func New() *Client {
	return &Client{
		http: &http.Client{Timeout: 15 * time.Second},
	}
}

// TVShow representa una serie de TV con toda su metadata
type TVShow struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Summary   string `json:"summary"`
	Premiered string `json:"premiered"` // "2008-01-20"
	Status    string `json:"status"`
	Rating    struct {
		Average float64 `json:"average"`
	} `json:"rating"`
	Image *struct {
		Medium   string `json:"medium"`
		Original string `json:"original"`
	} `json:"image"`
	Genres    []string `json:"genres"`
	Externals struct {
		IMDB    string `json:"imdb"`
		TheTVDB int    `json:"thetvdb"`
	} `json:"externals"`
}

// TVEpisode representa un episodio de una serie en TVmaze
type TVEpisode struct {
	Season  int    `json:"season"`
	Number  int    `json:"number"`
	Name    string `json:"name"`
	AirDate string `json:"airdate"`
}

// SeasonGroup agrupa episodios por temporada
type SeasonGroup struct {
	Number   int         `json:"number"`
	Episodes []TVEpisode `json:"episodes"`
}

// searchResult es la estructura de respuesta del endpoint de búsqueda
type searchResult struct {
	Score float64 `json:"score"`
	Show  TVShow  `json:"show"`
}

// PopularShows retorna shows populares de TVmaze (paginado)
func (c *Client) PopularShows(page int) ([]TVShow, error) {
	u := fmt.Sprintf("%s/shows?page=%d", baseURL, page)
	body, err := c.get(u)
	if err != nil {
		return nil, err
	}

	var shows []TVShow
	if err := json.Unmarshal(body, &shows); err != nil {
		return nil, fmt.Errorf("error parseando shows: %w", err)
	}

	return shows, nil
}

// SearchShows busca series de TV por titulo
func (c *Client) SearchShows(query string) ([]TVShow, error) {
	u := fmt.Sprintf("%s/search/shows?q=%s", baseURL, url.QueryEscape(query))
	body, err := c.get(u)
	if err != nil {
		return nil, err
	}

	var results []searchResult
	if err := json.Unmarshal(body, &results); err != nil {
		return nil, fmt.Errorf("error parseando resultados de busqueda: %w", err)
	}

	shows := make([]TVShow, 0, len(results))
	for _, r := range results {
		shows = append(shows, r.Show)
	}

	return shows, nil
}

// GetShowByIMDB busca una serie en TVmaze por su IMDB ID
func (c *Client) GetShowByIMDB(imdbID string) (*TVShow, error) {
	if imdbID == "" {
		return nil, fmt.Errorf("imdbID vacio")
	}

	u := fmt.Sprintf("%s/lookup/shows?imdb=%s", baseURL, imdbID)
	body, err := c.get(u)
	if err != nil {
		return nil, err
	}

	var show TVShow
	if err := json.Unmarshal(body, &show); err != nil {
		return nil, fmt.Errorf("error parseando show: %w", err)
	}

	return &show, nil
}

// GetSeasons obtiene todos los episodios de una serie agrupados por temporada
func (c *Client) GetSeasons(tvmazeID int) ([]SeasonGroup, error) {
	u := fmt.Sprintf("%s/shows/%d/episodes", baseURL, tvmazeID)
	body, err := c.get(u)
	if err != nil {
		return nil, err
	}

	var episodes []TVEpisode
	if err := json.Unmarshal(body, &episodes); err != nil {
		return nil, fmt.Errorf("error parseando episodios: %w", err)
	}

	seasonMap := make(map[int][]TVEpisode)
	for _, ep := range episodes {
		if ep.Number > 0 {
			seasonMap[ep.Season] = append(seasonMap[ep.Season], ep)
		}
	}

	var seasons []SeasonGroup
	for num, eps := range seasonMap {
		seasons = append(seasons, SeasonGroup{Number: num, Episodes: eps})
	}

	sort.Slice(seasons, func(i, j int) bool {
		return seasons[i].Number < seasons[j].Number
	})

	return seasons, nil
}

// ShowYear extrae el año de premiered (ej: "2008-01-20" → 2008)
func ShowYear(show TVShow) int {
	if len(show.Premiered) >= 4 {
		var y int
		fmt.Sscanf(show.Premiered[:4], "%d", &y)
		return y
	}
	return 0
}

// ShowGenres convierte slice de géneros a string separado por coma
func ShowGenres(show TVShow) string {
	return strings.Join(show.Genres, ", ")
}

// ShowPoster retorna la URL del poster (medium, o vacío si no hay imagen)
func ShowPoster(show TVShow) string {
	if show.Image != nil {
		return show.Image.Medium
	}
	return ""
}

// get realiza una peticion GET y retorna el body
func (c *Client) get(u string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("error creando request: %w", err)
	}
	req.Header.Set("User-Agent", "p2pollo/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error en request HTTP: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("no encontrado en TVmaze")
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status HTTP %d para %s", resp.StatusCode, u)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error leyendo respuesta: %w", err)
	}

	return body, nil
}
