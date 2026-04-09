package eztv

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const baseURL = "https://eztvx.to"

// Client es el cliente para la API de EZTV
type Client struct {
	http *http.Client
}

// New crea un nuevo cliente EZTV
func New() *Client {
	return &Client{
		http: &http.Client{Timeout: 15 * time.Second},
	}
}

// Torrent representa un torrent de EZTV
type Torrent struct {
	ID        int    `json:"id"`
	Hash      string `json:"hash"`
	Filename  string `json:"filename"`
	MagnetURL string `json:"magnet_url"`
	Title     string `json:"title"`
	ImdbID    string `json:"imdb_id"`
	Season    string `json:"season"`
	Episode   string `json:"episode"`
	Seeds     int    `json:"seeds"`
	Peers     int    `json:"peers"`
	Size      string `json:"size_bytes"` // tamaño en bytes como string
}

type eztvResponse struct {
	ImdbID        string    `json:"imdb_id"`
	TorrentsCount int       `json:"torrents_count"`
	Torrents      []Torrent `json:"torrents"`
}

// sxxexxRegex extrae temporada y episodio del filename cuando los campos de la API vienen en 0.
// Ej: "The.Terror.S01E01.720p..." → season=1, episode=1
var sxxexxRegex = regexp.MustCompile(`(?i)[._\s-]S(\d{1,2})E(\d{1,2})[._\s-]`)

// parseSE extrae temporada y episodio de un torrent.
// Usa los campos de la API si son válidos; si no, parsea el filename.
func parseSE(t Torrent) (season, episode int) {
	s, _ := strconv.Atoi(t.Season)
	e, _ := strconv.Atoi(t.Episode)
	if s > 0 && e > 0 {
		return s, e
	}
	// Fallback: parsear desde el filename
	if m := sxxexxRegex.FindStringSubmatch(t.Filename); len(m) == 3 {
		s, _ = strconv.Atoi(m[1])
		e, _ = strconv.Atoi(m[2])
	}
	return s, e
}

// GetEpisodeTorrents retorna torrents de EZTV para un episodio específico
// imdbID puede tener o no el prefijo "tt" (ej: "tt0903747" o "0903747")
func (c *Client) GetEpisodeTorrents(imdbID string, season, episode int) ([]Torrent, error) {
	// EZTV API espera el ID sin el prefijo "tt"
	numericID := strings.TrimPrefix(imdbID, "tt")
	if numericID == "" {
		return nil, fmt.Errorf("imdbID invalido: %s", imdbID)
	}

	// Obtener todos los torrents del show paginando si es necesario
	var all []Torrent
	page := 1
	for {
		u := fmt.Sprintf("%s/api/get-torrents?imdb_id=%s&limit=100&page=%d", baseURL, numericID, page)
		body, err := c.get(u)
		if err != nil {
			break
		}

		var resp eztvResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			break
		}

		if len(resp.Torrents) == 0 {
			break
		}

		all = append(all, resp.Torrents...)

		// Si ya encontramos episodios del season/episode buscado, parar
		found := false
		for _, t := range resp.Torrents {
			s, e := parseSE(t)
			if s == season && e == episode {
				found = true
				break
			}
		}
		if found || len(resp.Torrents) < 100 {
			break
		}
		page++
	}

	// Filtrar por temporada y episodio
	var result []Torrent
	for _, t := range all {
		s, e := parseSE(t)
		if s == season && e == episode {
			result = append(result, t)
		}
	}

	return result, nil
}

// get realiza una peticion GET y retorna el body
func (c *Client) get(u string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("error creando request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error en request HTTP: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status HTTP %d para EZTV", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error leyendo respuesta: %w", err)
	}

	return body, nil
}
