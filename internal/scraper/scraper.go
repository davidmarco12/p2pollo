package scraper

import (
	"fmt"
	"strings"
	"sync"
)

type Result struct {
	Name       string   // Nombre del torrent
	MagnetLink string   // Enlace magnet
	Size       string   // Tamano formateado (ej: "1.5 GB")
	Seeds      int      // Cantidad de seeders
	Leechers   int      // Cantidad de leechers
	Source     string   // Sitio de origen (ej: "1337x", "Nyaa.si")
	Category   string   // Categoria del contenido
	UploadDate string   // Fecha de subida como texto
	Subtitles  []string // Idiomas de subtitulos detectados en el nombre del release
}

// HealthScore calcula un puntaje de salud del torrent (0-100)
// basado en la proporcion de seeds sobre el total de pares
func (r *Result) HealthScore() int {
	total := r.Seeds + r.Leechers
	if total == 0 {
		return 0
	}
	return int(float64(r.Seeds) / float64(total) * 100)
}

// Provider es la interfaz que debe implementar cada sitio de torrents
type Provider interface {
	// Search realiza una busqueda en el sitio y devuelve los resultados
	Search(query string) ([]Result, error)
}

// Scraper es el orquestador principal que coordina busquedas
// a traves de multiples proveedores de torrents
type Scraper struct {
	providers map[string]Provider
	mu        sync.RWMutex
}

// New crea una nueva instancia de Scraper sin proveedores registrados
func New() *Scraper {
	return &Scraper{
		providers: make(map[string]Provider),
	}
}

// RegisterProvider registra un proveedor de busqueda con un nombre identificador
func (s *Scraper) RegisterProvider(name string, p Provider) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.providers[name] = p
}

// Search busca en todos los proveedores registrados de forma concurrente
// y combina los resultados en una sola lista
func (s *Scraper) Search(query string) ([]Result, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.providers) == 0 {
		return nil, fmt.Errorf("no hay proveedores registrados")
	}

	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("la consulta no puede estar vacia")
	}

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		results []Result
		errs    []error
	)

	// Lanzar una goroutine por cada proveedor
	for name, provider := range s.providers {
		wg.Add(1)
		go func(name string, p Provider) {
			defer wg.Done()

			res, err := p.Search(query)
			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				errs = append(errs, fmt.Errorf("error en proveedor %s: %w", name, err))
				return
			}
			results = append(results, res...)
		}(name, provider)
	}

	wg.Wait()

	// Si todos los proveedores fallaron, devolver error combinado
	if len(results) == 0 && len(errs) > 0 {
		mensajes := make([]string, len(errs))
		for i, e := range errs {
			mensajes[i] = e.Error()
		}
		return nil, fmt.Errorf("todos los proveedores fallaron: %s", strings.Join(mensajes, "; "))
	}

	return results, nil
}

// subtitleTokens mapea tokens del nombre del release a su etiqueta normalizada.
// Solo incluye tokens que son claramente indicadores de subtítulos/idioma.
var subtitleTokens = map[string]string{
	"MULTI":    "MULTI",
	"MULTISUB": "MULTI",
	"DUAL":     "DUAL",
	"SUB":      "SUB",
	"SUBS":     "SUB",
	"CC":       "CC",
	"ENG":      "ENG",
	"ENGLISH":  "ENG",
	"SPA":      "SPA",
	"SPANISH":  "SPA",
	"ESP":      "SPA",
	"LAT":      "LAT",
	"LATINO":   "LAT",
	"POR":      "POR",
	"PTBR":     "POR",
	"FRE":      "FRE",
	"FRENCH":   "FRE",
	"FRA":      "FRE",
	"GER":      "GER",
	"GERMAN":   "GER",
	"DEU":      "GER",
	"ITA":      "ITA",
	"ITALIAN":  "ITA",
	"JPN":      "JPN",
	"JAPANESE": "JPN",
	"CHI":      "CHI",
	"CHINESE":  "CHI",
	"KOR":      "KOR",
	"KOREAN":   "KOR",
	"RUS":      "RUS",
	"RUSSIAN":  "RUS",
	"ARA":      "ARA",
	"ARABIC":   "ARA",
}

// ParseSubtitleLanguages extrae etiquetas de idioma/subtítulo del nombre de un release.
// Devuelve una lista deduplicada y ordenada de etiquetas (ej: ["ENG", "MULTI", "SPA"]).
// Si no se detecta nada, devuelve nil.
func ParseSubtitleLanguages(name string) []string {
	// Separar por puntos, guiones, espacios y guiones bajos
	replacer := strings.NewReplacer(".", " ", "-", " ", "_", " ")
	tokens := strings.Fields(strings.ToUpper(replacer.Replace(name)))

	seen := make(map[string]bool)
	var result []string

	for _, tok := range tokens {
		if label, ok := subtitleTokens[tok]; ok {
			if !seen[label] {
				seen[label] = true
				result = append(result, label)
			}
		}
	}

	// Solo devolver si hay algo significativo (MULTI, idiomas concretos, etc.)
	// Filtrar el caso donde solo hay "SUB" genérico sin idioma concreto
	if len(result) == 1 && (result[0] == "SUB" || result[0] == "CC") {
		return nil
	}

	return result
}

// FormatSize convierte una cantidad de bytes a formato legible (ej: "1.5 GB")
func FormatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
