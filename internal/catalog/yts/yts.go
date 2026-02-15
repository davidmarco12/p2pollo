package yts

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/davidmarco12/p2pollo/internal/catalog"
)

// YTS implementa catalog.CatalogProvider scrapeando en.yts-official.org
type YTS struct {
	baseURL  string
	client   *http.Client
	trackers []string
}

// New crea una nueva instancia del proveedor YTS
func New(extraTrackers []string) *YTS {
	trackers := append([]string{}, catalog.DefaultTrackers...)
	trackers = append(trackers, extraTrackers...)

	return &YTS{
		baseURL: "https://en.yts-official.org",
		client: &http.Client{
			Timeout: 20 * time.Second,
		},
		trackers: trackers,
	}
}

// onclickRegex parsea el atributo onclick="openModal(id, "imdbId", "title", "year")"
var onclickRegex = regexp.MustCompile(`openModal\((\d+),\s*"([^"]*)",\s*"([^"]*)",\s*"(\d+)"\)`)

// ratingRegex extrae el numero de rating del texto (ej: "8.4" de " 8.4")
var ratingRegex = regexp.MustCompile(`([\d.]+)`)

// Popular retorna las peliculas mas recientes de la pagina principal
func (y *YTS) Popular(page int) ([]catalog.Movie, error) {
	u := y.baseURL + "/"
	if page > 1 {
		u = fmt.Sprintf("%s/?page=%d", y.baseURL, page)
	}
	return y.fetchMovies(u)
}

// Search busca peliculas por titulo
func (y *YTS) Search(query string, page int) ([]catalog.Movie, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("consulta vacia")
	}

	u := fmt.Sprintf("%s/?search=%s", y.baseURL, url.QueryEscape(query))
	if page > 1 {
		u = fmt.Sprintf("%s&page=%d", u, page)
	}
	return y.fetchMovies(u)
}

// fetchMovies obtiene y parsea las movie cards de una pagina HTML
func (y *YTS) fetchMovies(pageURL string) ([]catalog.Movie, error) {
	doc, err := y.fetchDocument(pageURL)
	if err != nil {
		return nil, err
	}

	var movies []catalog.Movie

	doc.Find("div.movie-card").Each(func(i int, card *goquery.Selection) {
		onclick, exists := card.Attr("onclick")
		if !exists {
			return
		}

		matches := onclickRegex.FindStringSubmatch(onclick)
		if len(matches) < 5 {
			return
		}

		movieID := matches[1]
		imdbID := matches[2]
		title := matches[3]
		year, _ := strconv.Atoi(matches[4])

		// Poster URL
		posterURL, _ := card.Find("img.movie-poster").Attr("src")

		// Rating
		ratingText := card.Find("span.movie-rating").Text()
		var rating float64
		if rm := ratingRegex.FindStringSubmatch(ratingText); len(rm) > 1 {
			rating, _ = strconv.ParseFloat(rm[1], 64)
		}

		// Generos
		genres := strings.TrimSpace(card.Find("span.movie-genres").Text())

		movies = append(movies, catalog.Movie{
			ID:        movieID,
			ImdbID:    imdbID,
			Title:     title,
			Year:      year,
			Rating:    rating,
			PosterURL: posterURL,
			Genres:    genres,
		})
	})

	return movies, nil
}

// --- AJAX para detalle de pelicula ---

// ajaxResponse es la respuesta del endpoint AJAX de YTS
type ajaxResponse struct {
	Success bool `json:"success"`
	YTS     struct {
		Data struct {
			Movie ajaxMovie `json:"movie"`
		} `json:"data"`
	} `json:"yts"`
	TMDB *tmdbData `json:"tmdb"`
}

type ajaxMovie struct {
	ID              int           `json:"id"`
	Title           string        `json:"title"`
	Year            int           `json:"year"`
	Rating          float64       `json:"rating"`
	Runtime         int           `json:"runtime"`
	Genres          []string      `json:"genres"`
	DescriptionFull string        `json:"description_full"`
	Torrents        []ajaxTorrent `json:"torrents"`
}

type ajaxTorrent struct {
	Hash    string `json:"hash"`
	Quality string `json:"quality"`
	Type    string `json:"type"`
	Size    string `json:"size"`
	Seeds   int    `json:"seeds"`
	Peers   int    `json:"peers"`
}

type tmdbData struct {
	Title        string  `json:"title"`
	Overview     string  `json:"overview"`
	Runtime      int     `json:"runtime"`
	VoteAverage  float64 `json:"vote_average"`
	PosterPath   string  `json:"poster_path"`
	BackdropPath string  `json:"backdrop_path"`
}

// Details obtiene el detalle completo de una pelicula incluyendo torrents
func (y *YTS) Details(movie catalog.Movie) (*catalog.MovieDetail, error) {
	u := fmt.Sprintf("%s/?ajax=movie_details&movie_id=%s&imdb_id=%s&title=%s&year=%d",
		y.baseURL,
		movie.ID,
		url.QueryEscape(movie.ImdbID),
		url.QueryEscape(movie.Title),
		movie.Year,
	)

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("error creando request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json")

	resp, err := y.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error en request HTTP: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status HTTP %d para detalle de %s", resp.StatusCode, movie.Title)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error leyendo respuesta: %w", err)
	}

	var data ajaxResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("error parseando JSON: %w", err)
	}

	m := data.YTS.Data.Movie

	// Usar TMDB para mejor metadata cuando este disponible
	description := m.DescriptionFull
	runtime := m.Runtime
	if data.TMDB != nil {
		if data.TMDB.Overview != "" {
			description = data.TMDB.Overview
		}
		if data.TMDB.Runtime > 0 {
			runtime = data.TMDB.Runtime
		}
	}

	// Construir torrents con magnet links
	torrents := make([]catalog.Torrent, 0, len(m.Torrents))
	// Filtrar torrents duplicados (mismo hash + quality)
	seen := make(map[string]bool)
	for _, t := range m.Torrents {
		key := t.Hash + "_" + t.Quality + "_" + t.Type
		if seen[key] {
			continue
		}
		seen[key] = true

		magnetLink := catalog.BuildMagnetLink(t.Hash, movie.Title, y.trackers)
		torrents = append(torrents, catalog.Torrent{
			Hash:       t.Hash,
			Quality:    t.Quality,
			Type:       t.Type,
			Size:       t.Size,
			Seeds:      t.Seeds,
			Peers:      t.Peers,
			MagnetLink: magnetLink,
		})
	}

	detail := &catalog.MovieDetail{
		Movie:       movie,
		Description: description,
		Runtime:     runtime,
		Torrents:    torrents,
	}

	return detail, nil
}

// fetchDocument hace un GET y retorna el documento HTML parseado
func (y *YTS) fetchDocument(u string) (*goquery.Document, error) {
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("error creando request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := y.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error en request HTTP: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status HTTP %d para %s", resp.StatusCode, u)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error parseando HTML: %w", err)
	}

	return doc, nil
}
