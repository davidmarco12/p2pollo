package providers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"p2pollo/internal/scraper"
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
	ID       string `json:"id"`
	Name     string `json:"name"`
	InfoHash string `json:"info_hash"`
	Leechers string `json:"leechers"`
	Seeders  string `json:"seeders"`
	Size     string `json:"size"`
	Category string `json:"category"`
	Status   string `json:"status"`
}

// tpbDetailResult representa la respuesta de /t.php?id=<id>.
type tpbDetailResult struct {
	Descr string `json:"descr"`
}

var (
	subtitleLineRe = regexp.MustCompile(`(?i)Subtitles?:\s*([^\r\n]+)`)
	bracketRe      = regexp.MustCompile(`\[[^\]]*\]`)
)

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
	if len(apiResults) == 1 && strings.EqualFold(strings.TrimRight(apiResults[0].Name, "."), "No results returned") {
		return []scraper.Result{}, nil
	}

	// Filtrar resultados válidos preservando el orden.
	valid := make([]tpbAPIResult, 0, len(apiResults))
	for _, r := range apiResults {
		if r.InfoHash != "" {
			valid = append(valid, r)
		}
	}

	// Enriquecer con subtítulos del endpoint de detalle en paralelo.
	subtitles := make([][]string, len(valid))
	sem := make(chan struct{}, 5)
	var wg sync.WaitGroup
	for i, r := range valid {
		wg.Add(1)
		go func(i int, r tpbAPIResult) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			subtitles[i] = t.fetchSubtitles(r.ID, r.Name)
		}(i, r)
	}
	wg.Wait()

	results := make([]scraper.Result, 0, len(valid))
	for i, r := range valid {
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
			Subtitles:  subtitles[i],
		})
	}

	return results, nil
}

// fetchSubtitles obtiene los subtítulos del endpoint de detalle de apibay.
// Si falla, cae al parseo del nombre del release.
func (t *ThePirateBay) fetchSubtitles(id, name string) []string {
	if id == "" {
		return scraper.ParseSubtitleLanguages(name)
	}
	detailURL := fmt.Sprintf("%s/t.php?id=%s", t.apiURL, id)
	resp, err := t.client.Get(detailURL)
	if err != nil {
		return scraper.ParseSubtitleLanguages(name)
	}
	defer resp.Body.Close()

	var detail tpbDetailResult
	if err := json.NewDecoder(resp.Body).Decode(&detail); err != nil {
		return scraper.ParseSubtitleLanguages(name)
	}

	subs := parseDescrSubtitles(detail.Descr)
	if len(subs) == 0 {
		return scraper.ParseSubtitleLanguages(name)
	}
	return subs
}

// parseDescrSubtitles extrae idiomas de subtítulos del campo descr de apibay.
// Ejemplo: "Subtitles: English [Selectable]" → ["ENG"]
func parseDescrSubtitles(descr string) []string {
	m := subtitleLineRe.FindStringSubmatch(descr)
	if m == nil {
		return nil
	}
	// Eliminar notas entre corchetes: [Selectable], [Forced], etc.
	line := bracketRe.ReplaceAllString(m[1], "")
	// Reemplazar comas por espacios para que ParseSubtitleLanguages tokenice bien
	line = strings.ReplaceAll(line, ",", " ")
	return scraper.ParseSubtitleLanguages(line)
}

// buildMagnet construye el magnet link desde el info hash y el nombre del torrent.
func (t *ThePirateBay) buildMagnet(infoHash, name string) string {
	magnet := fmt.Sprintf("magnet:?xt=urn:btih:%s&dn=%s", infoHash, url.QueryEscape(name))
	for _, tr := range t.trackers {
		magnet += "&tr=" + url.QueryEscape(tr)
	}
	return magnet
}
