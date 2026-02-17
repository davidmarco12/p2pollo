package providers

import (
	"fmt"
	"net/http"
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

	// Obtener magnet links de las páginas de detalle
	results := r.fetchMagnetLinks(partial)
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

// fetchMagnetLinks obtiene los magnet links de las páginas de detalle de forma concurrente.
func (r *Rargb) fetchMagnetLinks(partial []rargbResult) []scraper.Result {
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

			magnet := r.fetchMagnetFromDetail(sr.detailURL)

			// Solo incluir si obtuvimos el magnet link
			if magnet != "" {
				mu.Lock()
				results = append(results, scraper.Result{
					Name:       sr.name,
					MagnetLink: magnet,
					Size:       sr.size,
					Seeds:      sr.seeds,
					Leechers:   sr.leechers,
					Source:     "rargb",
				})
				mu.Unlock()
			}
		}(p)
	}

	wg.Wait()
	return results
}

// fetchMagnetFromDetail entra a la página de detalle de un torrent
// y extrae el magnet link.
func (r *Rargb) fetchMagnetFromDetail(detailPath string) string {
	url := r.baseURL + detailPath
	doc, err := r.fetchDocument(url)
	if err != nil {
		return ""
	}

	// El magnet link está en un <a> con href que empieza con "magnet:"
	magnet, _ := doc.Find("a[href^='magnet:']").First().Attr("href")
	return magnet
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
