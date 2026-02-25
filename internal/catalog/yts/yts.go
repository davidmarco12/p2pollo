package yts

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"p2pollo/internal/catalog"
	"github.com/PuerkitoBio/goquery"
	"github.com/sirupsen/logrus"
)

// YTS implementa catalog.CatalogProvider
// Popular: scraping de en.yts-official.org
// Search: OMDB API (omdbapi.com)
// Details: OMDB API para descripcion + scraper para torrents
type YTS struct {
	baseURL  string
	omdbKey  string
	client   *http.Client
	trackers []string
	log      *logrus.Logger
}

// onclickRegex parsea el atributo onclick="openModal(id, "imdbId", "title", "year")"
var onclickRegex = regexp.MustCompile(`openModal\((\d+),\s*"([^"]*)",\s*"([^"]*)",\s*"(\d+)"\)`)

// ratingRegex extrae el numero de rating del texto
var ratingRegex = regexp.MustCompile(`([\d.]+)`)

// New crea una nueva instancia del proveedor YTS
func New(extraTrackers []string) *YTS {
	trackers := append([]string{}, catalog.DefaultTrackers...)
	trackers = append(trackers, extraTrackers...)

	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	return &YTS{
		baseURL: "https://en.yts-official.org",
		omdbKey: "trilogy",
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
		trackers: trackers,
		log:      log,
	}
}

// --- Popular: scraping HTML de YTS ---

// Popular retorna las peliculas mas populares scrapeando la pagina principal
func (y *YTS) Popular(page int) ([]catalog.Movie, error) {
	u := y.baseURL + "/"
	if page > 1 {
		u = fmt.Sprintf("%s/?page=%d", y.baseURL, page)
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

		posterURL, _ := card.Find("img.movie-poster").Attr("src")

		ratingText := card.Find("span.movie-rating").Text()
		var rating float64
		if rm := ratingRegex.FindStringSubmatch(ratingText); len(rm) > 1 {
			rating, _ = strconv.ParseFloat(rm[1], 64)
		}

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

// --- Search: OMDB API ---

type omdbSearchResponse struct {
	Response string       `json:"Response"`
	Search   []omdbResult `json:"Search"`
	Error    string       `json:"Error"`
}

type omdbResult struct {
	Title  string `json:"Title"`
	Year   string `json:"Year"`
	ImdbID string `json:"imdbID"`
	Type   string `json:"Type"`
	Poster string `json:"Poster"`
}

// Search busca peliculas por titulo usando OMDB API
func (y *YTS) Search(query string, page int) ([]catalog.Movie, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("consulta vacia")
	}

	u := fmt.Sprintf("https://www.omdbapi.com/?s=%s&type=movie&page=%d&apikey=%s",
		url.QueryEscape(query), page, y.omdbKey)

	body, err := y.doGet(u)
	if err != nil {
		return nil, err
	}

	var resp omdbSearchResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("error parseando respuesta OMDB: %w", err)
	}

	if resp.Response != "True" {
		if resp.Error != "" {
			return nil, fmt.Errorf("OMDB: %s", resp.Error)
		}
		return []catalog.Movie{}, nil
	}

	movies := make([]catalog.Movie, 0, len(resp.Search))
	for _, r := range resp.Search {
		if r.Type != "movie" {
			continue
		}
		year := 0
		fmt.Sscanf(r.Year, "%d", &year)

		poster := r.Poster
		if poster == "N/A" {
			poster = ""
		}

		movies = append(movies, catalog.Movie{
			ID:        r.ImdbID,
			ImdbID:    r.ImdbID,
			Title:     r.Title,
			Year:      year,
			Rating:    0,
			PosterURL: poster,
			Genres:    "",
		})
	}

	return movies, nil
}

// --- Details: OMDB API para descripcion ---

type omdbDetailResponse struct {
	Title    string `json:"Title"`
	Year     string `json:"Year"`
	Genre    string `json:"Genre"`
	Plot     string `json:"Plot"`
	Runtime  string `json:"Runtime"`
	ImdbID   string `json:"imdbID"`
	ImdbRating string `json:"imdbRating"`
	Response string `json:"Response"`
}

// Details obtiene la descripcion de una pelicula via OMDB y su runtime
func (y *YTS) Details(movie catalog.Movie) (*catalog.MovieDetail, error) {
	imdbID := movie.ImdbID
	if imdbID == "" {
		imdbID = movie.ID
	}
	if imdbID == "" {
		return nil, fmt.Errorf("no imdbID disponible")
	}

	u := fmt.Sprintf("https://www.omdbapi.com/?i=%s&plot=full&apikey=%s",
		url.QueryEscape(imdbID), y.omdbKey)

	body, err := y.doGet(u)
	if err != nil {
		return nil, err
	}

	var resp omdbDetailResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("error parseando respuesta OMDB: %w", err)
	}

	if resp.Response != "True" {
		return nil, fmt.Errorf("OMDB: pelicula no encontrada")
	}

	// Parsear runtime (ej: "127 min" → 127)
	runtime := 0
	fmt.Sscanf(resp.Runtime, "%d min", &runtime)

	// Actualizar rating si esta disponible
	if resp.ImdbRating != "" && resp.ImdbRating != "N/A" {
		rating, _ := strconv.ParseFloat(resp.ImdbRating, 64)
		movie.Rating = rating
	}

	// Actualizar generos si estan disponibles
	if resp.Genre != "" && resp.Genre != "N/A" && movie.Genres == "" {
		movie.Genres = resp.Genre
	}

	plot := resp.Plot
	if plot == "N/A" {
		plot = ""
	}

	// Construir torrents YTS via magnet hash si tenemos los datos
	torrents := make([]catalog.Torrent, 0)
	sort.Slice(torrents, func(i, j int) bool {
		return torrents[i].Seeds > torrents[j].Seeds
	})

	return &catalog.MovieDetail{
		Movie:       movie,
		Description: plot,
		Runtime:     runtime,
		Torrents:    torrents,
	}, nil
}

// doGet hace un GET y retorna el body
func (y *YTS) doGet(u string) ([]byte, error) {
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
		return nil, fmt.Errorf("status HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error leyendo respuesta: %w", err)
	}
	return body, nil
}
