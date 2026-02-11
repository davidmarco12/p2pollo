package scraper

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Result representa un resultado de scraping de un sitio de torrents
type Result struct {
	Name       string // Nombre del torrent
	MagnetLink string // Enlace magnet
	Size       string // Tamano formateado (ej: "1.5 GB")
	Seeds      int    // Cantidad de seeders
	Leechers   int    // Cantidad de leechers
	Source     string // Sitio de origen (ej: "1337x", "Nyaa.si")
	Category   string // Categoria del contenido
	UploadDate string // Fecha de subida como texto
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

// SortOrder define el criterio de ordenamiento de resultados
type SortOrder string

const (
	SortBySeeds     SortOrder = "seeds"     // Ordenar por seeds (descendente)
	SortBySize      SortOrder = "size"      // Ordenar por tamano
	SortByDate      SortOrder = "date"      // Ordenar por fecha de subida
	SortByRelevance SortOrder = "relevance" // Ordenar por relevancia (seeds + leechers)
)

// Filters contiene los filtros aplicables a una busqueda
type Filters struct {
	MinSeeds int       // Minimo de seeds requerido
	MaxSize  string    // Tamano maximo (ej: "5 GB") — reservado para filtrado futuro
	SortBy   SortOrder // Criterio de ordenamiento
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

// SearchWithFilters busca en todos los proveedores y aplica los filtros indicados
func (s *Scraper) SearchWithFilters(query string, filters Filters) ([]Result, error) {
	results, err := s.Search(query)
	if err != nil {
		return nil, err
	}

	// Filtrar por minimo de seeds
	if filters.MinSeeds > 0 {
		var filtrados []Result
		for _, r := range results {
			if r.Seeds >= filters.MinSeeds {
				filtrados = append(filtrados, r)
			}
		}
		results = filtrados
	}

	// Ordenar segun el criterio seleccionado
	sortResults(results, filters.SortBy)

	return results, nil
}

// sortResults ordena los resultados segun el criterio indicado
func sortResults(results []Result, order SortOrder) {
	switch order {
	case SortBySeeds:
		sort.Slice(results, func(i, j int) bool {
			return results[i].Seeds > results[j].Seeds
		})
	case SortByDate:
		sort.Slice(results, func(i, j int) bool {
			// Comparacion lexicografica — asume formato consistente de fecha
			return results[i].UploadDate > results[j].UploadDate
		})
	case SortByRelevance:
		sort.Slice(results, func(i, j int) bool {
			scoreI := results[i].Seeds + results[i].Leechers
			scoreJ := results[j].Seeds + results[j].Leechers
			return scoreI > scoreJ
		})
	case SortBySize:
		// Ordenar por tamano como texto — para ordenamiento preciso
		// se necesitaria parsear el tamano a bytes
		sort.Slice(results, func(i, j int) bool {
			return results[i].Size > results[j].Size
		})
	}
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
