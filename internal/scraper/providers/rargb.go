package providers

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/davidmarco12/p2pollo/internal/scraper"
)

// Rargb implementa el proveedor de búsqueda para rargb.to.
// Realiza scraping del HTML para extraer los torrents disponibles.
type Rargb struct {
	baseURL string
	client  *http.Client
}

// NewRargb crea una nueva instancia del proveedor rargb.to.
func NewRargb() *Rargb {
	return &Rargb{
		baseURL: "https://rargb.to",
		client: &http.Client{
			Timeout: 15 * 1000000000, // 15 segundos
		},
	}
}

// rargbResult almacena los datos parciales de la página de resultados
// (sin magnet link, que se obtiene en la página de detalle)
type rargbResult struct {
	name      string
	detailURL string // ruta relativa, ej: /torrent/nombre-123456.html
	seeds     int
	leechers  int
	size      string
}

// detailInfo contiene los datos extraídos de la página de detalle
type detailInfo struct {
	magnet    string
	subtitles []string
}

// Search busca torrents en rargb.to para la consulta dada.
func (r *Rargb) Search(query string) ([]scraper.Result, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("consulta vacía")
	}

	// Buscar ordenado por seeders descendente
	searchURL := fmt.Sprintf("%s/search/?search=%s&order=seeders&by=DESC",
		r.baseURL, strings.ReplaceAll(query, " ", "+"))

	partial, err := r.fetchSearchResults(searchURL)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo resultados: %w", err)
	}

	if len(partial) == 0 {
		return []scraper.Result{}, nil
	}

	// Obtener magnet links y subtítulos de las páginas de detalle
	results := r.fetchDetails(partial)
	return results, nil
}

// fetchSearchResults parsea la tabla de resultados de rargb.to
func (r *Rargb) fetchSearchResults(url string) ([]rargbResult, error) {
	doc, err := r.fetchDocument(url)
	if err != nil {
		return nil, err
	}

	var results []rargbResult

	// Cada fila de la tabla de resultados (clase lista2)
	doc.Find("table.lista2t tr.lista2").Each(func(i int, row *goquery.Selection) {
		cells := row.Find("td.lista")

		// Necesitamos al menos 7 celdas (cat, file, category, date, size, seeds, leechers)
		if cells.Length() < 7 {
			return
		}

		// Nombre y link al detalle (segunda celda, índice 1)
		nameCell := cells.Eq(1)
		nameLink := nameCell.Find("a").First()
		name := strings.TrimSpace(nameLink.Text())
		detailURL, _ := nameLink.Attr("href")

		// Tamaño (quinta celda, índice 4)
		size := strings.TrimSpace(cells.Eq(4).Text())

		// Seeds (sexta celda, índice 5) — puede estar dentro de <font>
		seedsText := strings.TrimSpace(cells.Eq(5).Text())
		seeds := parseIntSafe(seedsText)

		// Leechers (séptima celda, índice 6)
		leechText := strings.TrimSpace(cells.Eq(6).Text())
		leechers := parseIntSafe(leechText)

		if name != "" && detailURL != "" {
			results = append(results, rargbResult{
				name:      name,
				detailURL: detailURL,
				seeds:     seeds,
				leechers:  leechers,
				size:      size,
			})
		}
	})

	return results, nil
}

// fetchDetails obtiene el magnet link y subtítulos de las páginas de detalle de forma concurrente.
func (r *Rargb) fetchDetails(partial []rargbResult) []scraper.Result {
	var (
		mu      sync.Mutex
		wg      sync.WaitGroup
		results []scraper.Result
		sem     = make(chan struct{}, 5) // máximo 5 requests concurrentes
	)

	for _, p := range partial {
		wg.Add(1)
		go func(sr rargbResult) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			info := r.fetchDetailInfo(sr.detailURL)

			// Solo incluir si obtuvimos el magnet link
			if info.magnet != "" {
				mu.Lock()
				results = append(results, scraper.Result{
					Name:       sr.name,
					MagnetLink: info.magnet,
					Size:       sr.size,
					Seeds:      sr.seeds,
					Leechers:   sr.leechers,
					Source:     "rargb",
					Subtitles:  info.subtitles,
				})
				mu.Unlock()
			}
		}(p)
	}

	wg.Wait()
	return results
}

// fetchDetailInfo extrae el magnet link y los idiomas de subtítulos de la página de detalle.
func (r *Rargb) fetchDetailInfo(detailPath string) detailInfo {
	doc, err := r.fetchDocument(r.baseURL + detailPath)
	if err != nil {
		return detailInfo{}
	}

	magnet, _ := doc.Find("a[href^='magnet:']").First().Attr("href")
	subtitles := scrapeSubtitlesFromDetail(doc)

	return detailInfo{magnet: magnet, subtitles: subtitles}
}

// subtitleSectionRe detecta el inicio de una sección de subtítulos en MediaInfo.
// Cubre los patrones: "-- Subtitle --", "---Subtitle----", "Text #1", "Text #2", etc.
var subtitleSectionRe = regexp.MustCompile(`(?i)(text\s*#\d+|\-+\s*subtitle\s*\-+|^subtitle\s*$)`)

// languageLineRe detecta una línea con información de idioma.
var languageLineRe = regexp.MustCompile(`(?i)\blanguage\b`)

// subtitleLangPatterns mapea nombres completos de idiomas a sus códigos.
var subtitleLangPatterns = []struct {
	re   *regexp.Regexp
	code string
}{
	{regexp.MustCompile(`(?i)\benglish\b`), "ENG"},
	{regexp.MustCompile(`(?i)\bspanish\b`), "SPA"},
	{regexp.MustCompile(`(?i)\blatin\b`), "LAT"},
	{regexp.MustCompile(`(?i)\bportuguese\b`), "POR"},
	{regexp.MustCompile(`(?i)\bfrench\b`), "FRE"},
	{regexp.MustCompile(`(?i)\bgerman\b`), "GER"},
	{regexp.MustCompile(`(?i)\bitalian\b`), "ITA"},
	{regexp.MustCompile(`(?i)\bjapanese\b`), "JPN"},
	{regexp.MustCompile(`(?i)\bchinese\b|(?i)\bmandarin\b`), "CHI"},
	{regexp.MustCompile(`(?i)\bkorean\b`), "KOR"},
	{regexp.MustCompile(`(?i)\brussian\b`), "RUS"},
	{regexp.MustCompile(`(?i)\barabic\b`), "ARA"},
	{regexp.MustCompile(`(?i)\bdutch\b`), "NLD"},
	{regexp.MustCompile(`(?i)\bswedish\b`), "SWE"},
	{regexp.MustCompile(`(?i)\bnorwegian\b`), "NOR"},
	{regexp.MustCompile(`(?i)\bpolish\b`), "POL"},
	{regexp.MustCompile(`(?i)\bturkish\b`), "TUR"},
	{regexp.MustCompile(`(?i)\bhindi\b`), "HIN"},
	{regexp.MustCompile(`(?i)\bgreek\b`), "GRE"},
	{regexp.MustCompile(`(?i)\bhebrew\b`), "HEB"},
	{regexp.MustCompile(`(?i)\bthai\b`), "THA"},
}

// scrapeSubtitlesFromDetail extrae los idiomas de subtítulos del HTML de la página de detalle.
// Busca en secciones MediaInfo del tipo "Text #N" y "-- Subtitle --".
func scrapeSubtitlesFromDetail(doc *goquery.Document) []string {
	seen := make(map[string]bool)
	var result []string

	addLangs := func(text string) {
		for _, p := range subtitleLangPatterns {
			if p.re.MatchString(text) && !seen[p.code] {
				seen[p.code] = true
				result = append(result, p.code)
			}
		}
	}

	// Estrategia 1: celdas de tabla adyacentes al header "Subtitle"
	doc.Find("td, th").Each(func(_ int, s *goquery.Selection) {
		if subtitleSectionRe.MatchString(strings.TrimSpace(s.Text())) {
			s.Next().Each(func(_ int, next *goquery.Selection) {
				addLangs(next.Text())
			})
			addLangs(s.Text())
		}
	})

	// Estrategia 2: parsear el texto completo línea por línea buscando secciones subtitle
	// (cubre MediaInfo embebido en divs o pre/textarea)
	fullText := doc.Find("body").Text()
	lines := strings.Split(fullText, "\n")

	inSubtitleBlock := false
	blankLines := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			blankLines++
			// Después de 3 líneas en blanco salimos del bloque subtitle
			if blankLines >= 3 {
				inSubtitleBlock = false
			}
			continue
		}
		blankLines = 0

		if subtitleSectionRe.MatchString(trimmed) {
			inSubtitleBlock = true
		}

		if inSubtitleBlock {
			// Solo procesar líneas que hablan de "Language" o tienen idiomas directamente
			if languageLineRe.MatchString(trimmed) || inSubtitleBlock {
				addLangs(trimmed)
			}
		}
	}

	return result
}

// fetchDocument hace un GET y retorna el documento parseado.
func (r *Rargb) fetchDocument(url string) (*goquery.Document, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creando request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error en request HTTP: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status HTTP %d para %s", resp.StatusCode, url)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error parseando HTML: %w", err)
	}

	return doc, nil
}
