package providers

import (
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"

	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"

	"github.com/PuerkitoBio/goquery"
	"p2pollo/internal/scraper"
)

// mirrors son dominios alternativos de 1337x.
// Se prueban en orden hasta que uno responda.
// 1337x.to tiene Cloudflare agresivo — si tls-client logra pasar, se usa.
// 1337xx.to (doble x) es fallback sin Cloudflare pero con contenido diferente.
var mirrors = []string{
	"https://1337x.to",
	"https://www.1337x.to",
	"https://1337x.st",
	"https://1337x.gd",
	"https://1337x.so",
	"https://x1337x.ws",
	"https://www.1337xx.to",
}

// Leet implementa el proveedor de búsqueda para el sitio 1337x.
// Realiza scraping del HTML para extraer los torrents disponibles.
type Leet struct {
	baseURL string
	client  tls_client.HttpClient
}

// NewLeet crea una nueva instancia del proveedor 1337x.
// Usa tls-client con perfil Chrome_131 para pasar Cloudflare:
// - TLS fingerprint idéntico a Chrome (JA3)
// - Header ordering correcto
// - HTTP/2 con pseudo-headers en orden correcto
func NewLeet() *Leet {
	client, _ := tls_client.NewHttpClient(
		tls_client.NewNoopLogger(),
		tls_client.WithTimeoutSeconds(15),
		tls_client.WithClientProfile(profiles.Chrome_131),
	)

	return &Leet{client: client}
}

// searchResult almacena los datos parciales de la página de resultados
// (sin magnet link, que se obtiene en la página de detalle)
type searchResult struct {
	name      string
	detailURL string // ruta relativa, ej: /torrent/123456/nombre/
	seeds     int
	leechers  int
	size      string
}

// Search busca torrents en 1337x para la consulta dada.
// Prueba múltiples mirrors hasta encontrar uno que responda.
func (l *Leet) Search(query string) ([]scraper.Result, error) {
	query = strings.ReplaceAll(strings.TrimSpace(query), " ", "+")
	if query == "" {
		return nil, fmt.Errorf("consulta vacía")
	}

	// Si ya tenemos un mirror funcional, usarlo primero
	if l.baseURL != "" {
		results, err := l.searchWithBase(l.baseURL, query)
		if err == nil {
			return results, nil
		}
		// Si falló, limpiar y probar los demás
		l.baseURL = ""
	}

	// Probar cada mirror hasta que uno funcione
	var lastErr error
	for _, mirror := range mirrors {
		results, err := l.searchWithBase(mirror, query)
		if err == nil {
			l.baseURL = mirror // Recordar el mirror funcional
			return results, nil
		}
		lastErr = err
	}

	return nil, fmt.Errorf("ningún mirror disponible, último error: %w", lastErr)
}

// searchWithBase realiza la búsqueda usando un dominio base específico
func (l *Leet) searchWithBase(base, query string) ([]scraper.Result, error) {
	searchURL := fmt.Sprintf("%s/sort-search/%s/seeders/desc/1/", base, query)
	partial, err := l.fetchSearchResults(searchURL)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo resultados: %w", err)
	}

	if len(partial) == 0 {
		return []scraper.Result{}, nil
	}

	// Obtener magnet links usando el mismo base domain
	results := l.fetchMagnetLinks(base, partial)
	return results, nil
}

// fetchSearchResults parsea la tabla de resultados de 1337x
func (l *Leet) fetchSearchResults(url string) ([]searchResult, error) {
	doc, err := l.fetchDocument(url)
	if err != nil {
		return nil, err
	}

	// Si no hay tabla de resultados, la página es un challenge de Cloudflare
	// u otra página de error — saltar a siguiente mirror
	if doc.Find("table.table-list").Length() == 0 {
		return nil, fmt.Errorf("tabla de resultados no encontrada (posible bloqueo de Cloudflare)")
	}

	var results []searchResult

	// Cada fila de la tabla de resultados
	doc.Find("table.table-list tbody tr").Each(func(i int, row *goquery.Selection) {
		// Nombre y link al detalle — usar coll-1 con filtro de href para obtener el link correcto
		nameLink := row.Find("td.coll-1 a[href*='/torrent/']").First()
		name := strings.TrimSpace(nameLink.Text())
		detailURL, _ := nameLink.Attr("href")

		// Seeds (coll-2)
		seedsText := strings.TrimSpace(row.Find("td.coll-2").Text())
		seeds := parseIntSafe(seedsText)

		// Leechers (coll-3)
		leechText := strings.TrimSpace(row.Find("td.coll-3").Text())
		leechers := parseIntSafe(leechText)

		// Tamaño (coll-4) — tiene el valor + un <span> con la unidad duplicada
		sizeNode := row.Find("td.coll-4")
		// Remover el span oculto para quedarnos con el texto limpio
		sizeNode.Find("span").Remove()
		size := strings.TrimSpace(sizeNode.Text())

		if name != "" && detailURL != "" {
			results = append(results, searchResult{
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
// Limita la concurrencia para no saturar el servidor.
func (l *Leet) fetchMagnetLinks(base string, partial []searchResult) []scraper.Result {
	var (
		mu      sync.Mutex
		wg      sync.WaitGroup
		results []scraper.Result
		sem     = make(chan struct{}, 5) // máximo 5 requests concurrentes
	)

	for _, p := range partial {
		wg.Add(1)
		go func(sr searchResult) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			magnet := l.fetchMagnetFromDetail(base, sr.detailURL)

			// Solo incluir si obtuvimos el magnet link
			if magnet != "" {
				mu.Lock()
				results = append(results, scraper.Result{
					Name:       sr.name,
					MagnetLink: magnet,
					Size:       sr.size,
					Seeds:      sr.seeds,
					Leechers:   sr.leechers,
					Source:     "1337x",
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
func (l *Leet) fetchMagnetFromDetail(base, detailPath string) string {
	url := base + detailPath
	doc, err := l.fetchDocument(url)
	if err != nil {
		return ""
	}

	// El magnet link está en un <a> con href que empieza con "magnet:"
	magnet, _ := doc.Find("a[href^='magnet:']").First().Attr("href")
	return magnet
}

// fetchDocument hace un GET con perfil Chrome y retorna el documento parseado.
// tls-client se encarga del TLS fingerprint, header ordering y HTTP/2.
func (l *Leet) fetchDocument(url string) (*goquery.Document, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creando request: %w", err)
	}

	// Headers de Chrome con orden correcto (Cloudflare verifica el orden)
	req.Header = http.Header{
		"upgrade-insecure-requests": {"1"},
		"user-agent":                {"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"},
		"accept":                    {"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8"},
		"sec-fetch-site":            {"none"},
		"sec-fetch-mode":            {"navigate"},
		"sec-fetch-user":            {"?1"},
		"sec-fetch-dest":            {"document"},
		"accept-encoding":           {"gzip, deflate, br"},
		"accept-language":           {"en-US,en;q=0.9"},
		http.HeaderOrderKey: {
			"upgrade-insecure-requests",
			"user-agent",
			"accept",
			"sec-fetch-site",
			"sec-fetch-mode",
			"sec-fetch-user",
			"sec-fetch-dest",
			"accept-encoding",
			"accept-language",
		},
		http.PHeaderOrderKey: {
			":method",
			":authority",
			":scheme",
			":path",
		},
	}

	resp, err := l.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error en request HTTP: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		// Loguear snippet del body para diagnosticar el tipo de bloqueo
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 500))
		snippet := strings.TrimSpace(string(bodyBytes))
		log.Printf("[1337x] HTTP %d para %s — snippet: %s", resp.StatusCode, url, snippet)
		return nil, fmt.Errorf("status HTTP %d para %s", resp.StatusCode, url)
	}

	// Decodificar gzip si el servidor responde comprimido
	var body io.Reader = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gr, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("error decodificando gzip: %w", err)
		}
		defer gr.Close()
		body = gr
	}

	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return nil, fmt.Errorf("error parseando HTML: %w", err)
	}

	return doc, nil
}

// parseIntSafe convierte un string a int, retornando 0 si falla
func parseIntSafe(s string) int {
	n := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}
