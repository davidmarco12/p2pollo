package search

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/davidmarco12/p2pollo/internal/config"
	"github.com/sirupsen/logrus"
)

// Result representa un resultado de búsqueda
type Result struct {
	Name       string    // Nombre del torrent
	MagnetLink string    // Magnet link
	Size       int64     // Tamaño en bytes
	Seeds      int       // Seeders
	Leechers   int       // Leechers
	UploadDate time.Time // Fecha de subida
	Category   string    // Categoría (video, audio, etc)
	Source     string    // Tracker de origen
}

// Engine motor de búsqueda en trackers
type Engine struct {
	config *config.Config
	log    *logrus.Logger
	cache  *searchCache
	mu     sync.RWMutex
}

// searchCache caché de búsquedas recientes
type searchCache struct {
	entries map[string]*cacheEntry
	mu      sync.RWMutex
}

type cacheEntry struct {
	results   []Result
	timestamp time.Time
	query     string
}

// New crea un nuevo motor de búsqueda
func New(cfg *config.Config) *Engine {
	log := logrus.New()
	log.SetLevel(logrus.InfoLevel)
	if cfg.Logging.Level == "debug" {
		log.SetLevel(logrus.DebugLevel)
	}

	return &Engine{
		config: cfg,
		log:    log,
		cache: &searchCache{
			entries: make(map[string]*cacheEntry),
		},
	}
}

// Search busca torrents en los trackers configurados
func (e *Engine) Search(query string) ([]Result, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if query == "" {
		return nil, fmt.Errorf("query vacía")
	}

	// Normalizar query
	query = strings.ToLower(strings.TrimSpace(query))

	// Buscar en caché
	if cached := e.cache.get(query); cached != nil {
		e.log.Infof("Resultados obtenidos del caché para: %s", query)
		return cached, nil
	}

	e.log.Infof("Buscando: %s", query)

	// Buscar en trackers (simulado por ahora)
	// En una implementación real, aquí se conectaría a APIs de trackers públicos
	// o se haría scraping de sitios web
	results := e.searchInTrackers(query)

	// Guardar en caché
	e.cache.set(query, results)

	e.log.Infof("Encontrados %d resultados para: %s", len(results), query)

	return results, nil
}

// searchInTrackers realiza la búsqueda en los trackers
// NOTA: Esta es una implementación simulada
// En producción, aquí se usarían APIs reales o scraping
func (e *Engine) searchInTrackers(query string) []Result {
	// Por ahora retornamos resultados de ejemplo
	// En una implementación real, esto haría requests HTTP a trackers públicos

	results := []Result{
		{
			Name:       "Example Movie 2024 1080p",
			MagnetLink: "magnet:?xt=urn:btih:1234567890abcdef1234567890abcdef12345678",
			Size:       2 * 1024 * 1024 * 1024, // 2 GB
			Seeds:      150,
			Leechers:   25,
			UploadDate: time.Now().Add(-24 * time.Hour),
			Category:   "Video",
			Source:     "Example Tracker",
		},
		{
			Name:       "Example Series S01E01 720p",
			MagnetLink: "magnet:?xt=urn:btih:abcdef1234567890abcdef1234567890abcdef12",
			Size:       500 * 1024 * 1024, // 500 MB
			Seeds:      80,
			Leechers:   15,
			UploadDate: time.Now().Add(-48 * time.Hour),
			Category:   "Video",
			Source:     "Example Tracker",
		},
	}

	// Filtrar por query (simulado)
	var filtered []Result
	for _, r := range results {
		if strings.Contains(strings.ToLower(r.Name), query) {
			filtered = append(filtered, r)
		}
	}

	return filtered
}

// SearchWithFilters busca con filtros adicionales
func (e *Engine) SearchWithFilters(query string, filters Filters) ([]Result, error) {
	// Obtener resultados base
	results, err := e.Search(query)
	if err != nil {
		return nil, err
	}

	// Aplicar filtros
	filtered := e.applyFilters(results, filters)

	return filtered, nil
}

// Filters filtros de búsqueda
type Filters struct {
	MinSeeds int       // Mínimo de seeds
	MaxSize  int64     // Tamaño máximo en bytes (0 = sin límite)
	MinSize  int64     // Tamaño mínimo en bytes
	Category string    // Categoría específica
	SortBy   SortOrder // Ordenamiento
}

// SortOrder orden de resultados
type SortOrder string

const (
	SortBySeeds     SortOrder = "seeds"     // Por número de seeds (descendente)
	SortByLeechers  SortOrder = "leechers"  // Por número de leechers
	SortBySize      SortOrder = "size"      // Por tamaño
	SortByDate      SortOrder = "date"      // Por fecha (más reciente primero)
	SortByRelevance SortOrder = "relevance" // Por relevancia (seeds + leechers)
)

// applyFilters aplica filtros a los resultados
func (e *Engine) applyFilters(results []Result, filters Filters) []Result {
	var filtered []Result

	for _, r := range results {
		// Filtrar por seeds
		if filters.MinSeeds > 0 && r.Seeds < filters.MinSeeds {
			continue
		}

		// Filtrar por tamaño máximo
		if filters.MaxSize > 0 && r.Size > filters.MaxSize {
			continue
		}

		// Filtrar por tamaño mínimo
		if filters.MinSize > 0 && r.Size < filters.MinSize {
			continue
		}

		// Filtrar por categoría
		if filters.Category != "" && !strings.EqualFold(r.Category, filters.Category) {
			continue
		}

		filtered = append(filtered, r)
	}

	// Ordenar resultados
	e.sortResults(filtered, filters.SortBy)

	return filtered
}

// sortResults ordena los resultados según el criterio
func (e *Engine) sortResults(results []Result, order SortOrder) {
	switch order {
	case SortBySeeds:
		sort.Slice(results, func(i, j int) bool {
			return results[i].Seeds > results[j].Seeds
		})
	case SortByLeechers:
		sort.Slice(results, func(i, j int) bool {
			return results[i].Leechers > results[j].Leechers
		})
	case SortBySize:
		sort.Slice(results, func(i, j int) bool {
			return results[i].Size > results[j].Size
		})
	case SortByDate:
		sort.Slice(results, func(i, j int) bool {
			return results[i].UploadDate.After(results[j].UploadDate)
		})
	case SortByRelevance:
		sort.Slice(results, func(i, j int) bool {
			scoreI := results[i].Seeds + results[i].Leechers
			scoreJ := results[j].Seeds + results[j].Leechers
			return scoreI > scoreJ
		})
	}
}

// RankResults rankea resultados por relevancia
func (e *Engine) RankResults(results []Result) []Result {
	// Copiar para no modificar el original
	ranked := make([]Result, len(results))
	copy(ranked, results)

	// Ordenar por relevancia (seeds + leechers)
	sort.Slice(ranked, func(i, j int) bool {
		scoreI := ranked[i].Seeds*2 + ranked[i].Leechers // Seeds valen el doble
		scoreJ := ranked[j].Seeds*2 + ranked[j].Leechers
		return scoreI > scoreJ
	})

	return ranked
}

// get obtiene resultados del caché
func (c *searchCache) get(query string) []Result {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[query]
	if !exists {
		return nil
	}

	// Verificar si el caché expiró (15 minutos)
	if time.Since(entry.timestamp) > 15*time.Minute {
		delete(c.entries, query)
		return nil
	}

	return entry.results
}

// set guarda resultados en el caché
func (c *searchCache) set(query string, results []Result) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[query] = &cacheEntry{
		results:   results,
		timestamp: time.Now(),
		query:     query,
	}

	// Limpiar caché viejo (mantener máximo 100 entradas)
	if len(c.entries) > 100 {
		c.cleanup()
	}
}

// cleanup limpia entradas viejas del caché
func (c *searchCache) cleanup() {
	now := time.Now()
	for key, entry := range c.entries {
		if now.Sub(entry.timestamp) > 15*time.Minute {
			delete(c.entries, key)
		}
	}
}

// ClearCache limpia el caché de búsquedas
func (e *Engine) ClearCache() {
	e.cache.mu.Lock()
	defer e.cache.mu.Unlock()

	e.cache.entries = make(map[string]*cacheEntry)
	e.log.Info("Caché de búsquedas limpiado")
}

// Stats estadísticas del motor de búsqueda
func (e *Engine) Stats() SearchStats {
	e.cache.mu.RLock()
	defer e.cache.mu.RUnlock()

	return SearchStats{
		CachedQueries: len(e.cache.entries),
	}
}

// SearchStats estadísticas de búsqueda
type SearchStats struct {
	CachedQueries int // Queries en caché
}

// FormatSize formatea el tamaño en formato legible
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

// HealthScore calcula un score de salud del torrent (0-100)
func (r *Result) HealthScore() int {
	if r.Seeds == 0 {
		return 0
	}

	total := r.Seeds + r.Leechers
	if total == 0 {
		return 0
	}

	// Ratio de seeds vs total
	ratio := float64(r.Seeds) / float64(total)

	// Score base
	score := int(ratio * 100)

	// Bonus por cantidad absoluta de seeds
	if r.Seeds >= 100 {
		score = min(100, score+10)
	} else if r.Seeds >= 50 {
		score = min(100, score+5)
	}

	return score
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
