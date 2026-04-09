package providers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"p2pollo/internal/scraper"
)

// ytsDefaultTrackers son los trackers que YTS recomienda en sus magnet links.
var ytsDefaultTrackers = []string{
	"udp://open.tracker.cl:1337/announce",
	"udp://tracker.opentrackr.org:1337/announce",
	"udp://tracker.openbittorrent.com:6969/announce",
	"udp://p4p.arenabg.com:1337/announce",
	"udp://tracker.torrent.eu.org:451/announce",
}

// YTS implementa búsqueda usando la API JSON pública de YTS (yts.mx).
// No requiere scraping HTML — la API devuelve JSON directamente con hashes e info de torrents.
type YTSScraper struct {
	apiURL   string
	client   *http.Client
	trackers []string
}

// NewYTS crea una nueva instancia del proveedor YTS.
// trackers: lista adicional del config (se fusiona con los trackers por defecto).
func NewYTS(trackers []string) *YTSScraper {
	seen := make(map[string]bool)
	merged := make([]string, 0, len(ytsDefaultTrackers)+len(trackers))
	for _, tr := range ytsDefaultTrackers {
		if !seen[tr] {
			seen[tr] = true
			merged = append(merged, tr)
		}
	}
	for _, tr := range trackers {
		if !seen[tr] {
			seen[tr] = true
			merged = append(merged, tr)
		}
	}

	return &YTSScraper{
		apiURL:   "https://yts.mx/api/v2",
		client:   &http.Client{Timeout: 15 * time.Second},
		trackers: merged,
	}
}

// ytsAPIResponse representa la respuesta de la API de YTS.
type ytsAPIResponse struct {
	Status string      `json:"status"`
	Data   ytsAPIData  `json:"data"`
}

type ytsAPIData struct {
	MovieCount int        `json:"movie_count"`
	Movies     []ytsMovie `json:"movies"`
}

type ytsMovie struct {
	Title    string       `json:"title"`
	Year     int          `json:"year"`
	Rating   float64      `json:"rating"`
	Genres   []string     `json:"genres"`
	Torrents []ytsTorrent `json:"torrents"`
}

type ytsTorrent struct {
	Hash       string `json:"hash"`
	Quality    string `json:"quality"`
	Type       string `json:"type"`
	SizeBytes  int64  `json:"size_bytes"`
	Seeds      int    `json:"seeds"`
	Peers      int    `json:"peers"`
	DateUpload string `json:"date_uploaded"`
}

// Search busca películas en YTS via la API pública y devuelve un resultado por torrent.
func (y *YTSScraper) Search(query string) ([]scraper.Result, error) {
	apiURL := fmt.Sprintf("%s/list_movies.json?query_term=%s&sort_by=seeds&limit=50",
		y.apiURL, url.QueryEscape(query))

	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creando request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json")

	resp, err := y.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error en request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status HTTP %d para %s", resp.StatusCode, apiURL)
	}

	var apiResp ytsAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("error parseando respuesta JSON: %w", err)
	}

	if apiResp.Status != "ok" {
		return []scraper.Result{}, nil
	}

	results := make([]scraper.Result, 0)
	for _, movie := range apiResp.Data.Movies {
		for _, torrent := range movie.Torrents {
			if torrent.Hash == "" {
				continue
			}

			// Nombre del release: "Titulo (Año) [Calidad] [Tipo] [YTS.MX]"
			name := fmt.Sprintf("%s (%d) [%s] [%s] [YTS.MX]",
				movie.Title, movie.Year, torrent.Quality, torrent.Type)

			sizeStr := ""
			if torrent.SizeBytes > 0 {
				sizeStr = scraper.FormatSize(torrent.SizeBytes)
			}

			results = append(results, scraper.Result{
				Name:       name,
				MagnetLink: y.buildMagnet(torrent.Hash, name),
				Size:       sizeStr,
				Seeds:      torrent.Seeds,
				Leechers:   torrent.Peers,
				Source:     "yts.mx",
				UploadDate: torrent.DateUpload,
				Subtitles:  scraper.ParseSubtitleLanguages(name),
			})
		}
	}

	return results, nil
}

// buildMagnet construye el magnet link desde el hash e infohash del torrent.
func (y *YTSScraper) buildMagnet(hash, name string) string {
	magnet := fmt.Sprintf("magnet:?xt=urn:btih:%s&dn=%s", hash, url.QueryEscape(name))
	for _, tr := range y.trackers {
		magnet += "&tr=" + url.QueryEscape(tr)
	}
	return magnet
}
