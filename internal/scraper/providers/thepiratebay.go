package providers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/davidmarco12/p2pollo/internal/scraper"
)

// tpbDefaultTrackers son los trackers que The Pirate Bay embebe en sus magnet links.
var tpbDefaultTrackers = []string{
	"udp://tracker.opentrackr.org:1337/announce",
	"udp://open.tracker.cl:1337/announce",
	"udp://tracker.openbittorrent.com:6969/announce",
	"http://p4p.arenabg.com:1337/announce",
	"udp://tracker.torrent.eu.org:451/announce",
	"udp://tracker.bittor.pw:1337/announce",
}

// ThePirateBay implementa búsqueda usando la API JSON pública de apibay.org.
// No requiere scraping HTML — la API devuelve JSON directamente.
type ThePirateBay struct {
	apiURL   string
	client   *http.Client
	trackers []string
}

// NewThePirateBay crea una nueva instancia del proveedor.
// trackers: lista adicional del config (se fusiona con los trackers por defecto de TPB).
func NewThePirateBay(trackers []string) *ThePirateBay {
	seen := make(map[string]bool)
	merged := make([]string, 0, len(tpbDefaultTrackers)+len(trackers))
	for _, tr := range tpbDefaultTrackers {
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

	return &ThePirateBay{
		apiURL:   "https://apibay.org",
		client:   &http.Client{Timeout: 15 * time.Second},
		trackers: merged,
	}
}

// tpbAPIResult representa un elemento de la respuesta JSON de apibay.org.
type tpbAPIResult struct {
	Name     string `json:"name"`
	InfoHash string `json:"info_hash"`
	Leechers string `json:"leechers"`
	Seeders  string `json:"seeders"`
	Size     string `json:"size"`
	Category string `json:"category"`
	Status   string `json:"status"`
}

// Search busca torrents en The Pirate Bay via la API de apibay.org.
func (t *ThePirateBay) Search(query string) ([]scraper.Result, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("consulta vacía")
	}

	apiURL := fmt.Sprintf("%s/q.php?q=%s&cat=0", t.apiURL, url.QueryEscape(query))

	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creando request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error en request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status HTTP %d para %s", resp.StatusCode, apiURL)
	}

	var apiResults []tpbAPIResult
	if err := json.NewDecoder(resp.Body).Decode(&apiResults); err != nil {
		return nil, fmt.Errorf("error parseando respuesta JSON: %w", err)
	}

	// La API devuelve un único elemento con este nombre cuando no hay resultados.
	if len(apiResults) == 1 && apiResults[0].Name == "No results returned." {
		return []scraper.Result{}, nil
	}

	results := make([]scraper.Result, 0, len(apiResults))
	for _, r := range apiResults {
		if r.InfoHash == "" {
			continue
		}

		seeds, _ := strconv.Atoi(r.Seeders)
		leechers, _ := strconv.Atoi(r.Leechers)

		sizeBytes, _ := strconv.ParseInt(r.Size, 10, 64)
		sizeStr := ""
		if sizeBytes > 0 {
			sizeStr = scraper.FormatSize(sizeBytes)
		}

		results = append(results, scraper.Result{
			Name:       r.Name,
			MagnetLink: t.buildMagnet(r.InfoHash, r.Name),
			Size:       sizeStr,
			Seeds:      seeds,
			Leechers:   leechers,
			Source:     "thepiratebay",
			Subtitles:  scraper.ParseSubtitleLanguages(r.Name),
		})
	}

	return results, nil
}

// buildMagnet construye el magnet link desde el info hash y el nombre del torrent.
func (t *ThePirateBay) buildMagnet(infoHash, name string) string {
	magnet := fmt.Sprintf("magnet:?xt=urn:btih:%s&dn=%s", infoHash, url.QueryEscape(name))
	for _, tr := range t.trackers {
		magnet += "&tr=" + url.QueryEscape(tr)
	}
	return magnet
}
