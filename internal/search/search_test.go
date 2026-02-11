package search

import (
	"testing"
	"time"

	"github.com/davidmarco12/p2pollo/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNew prueba la creación del motor de búsqueda
func TestNew(t *testing.T) {
	cfg := config.DefaultConfig()
	engine := New(cfg)

	assert.NotNil(t, engine)
	assert.NotNil(t, engine.cache)
}

// TestSearch prueba la búsqueda básica
func TestSearch(t *testing.T) {
	cfg := config.DefaultConfig()
	engine := New(cfg)

	results, err := engine.Search("example")
	require.NoError(t, err)
	assert.Greater(t, len(results), 0, "Debería retornar resultados")

	// Verificar estructura de resultados
	for _, r := range results {
		assert.NotEmpty(t, r.Name)
		assert.NotEmpty(t, r.MagnetLink)
		assert.Greater(t, r.Size, int64(0))
	}
}

// TestSearch_EmptyQuery prueba búsqueda con query vacía
func TestSearch_EmptyQuery(t *testing.T) {
	cfg := config.DefaultConfig()
	engine := New(cfg)

	_, err := engine.Search("")
	assert.Error(t, err, "Debería fallar con query vacía")
}

// TestSearch_Cache prueba el caché de búsquedas
func TestSearch_Cache(t *testing.T) {
	cfg := config.DefaultConfig()
	engine := New(cfg)

	query := "test movie"

	// Primera búsqueda
	results1, err := engine.Search(query)
	require.NoError(t, err)

	// Segunda búsqueda (debería usar caché)
	results2, err := engine.Search(query)
	require.NoError(t, err)

	// Deberían ser iguales
	assert.Equal(t, len(results1), len(results2))

	// Verificar que está en caché
	stats := engine.Stats()
	assert.Equal(t, 1, stats.CachedQueries)
}

// TestSearchWithFilters prueba búsqueda con filtros
func TestSearchWithFilters(t *testing.T) {
	cfg := config.DefaultConfig()
	engine := New(cfg)

	filters := Filters{
		MinSeeds: 10,
		SortBy:   SortBySeeds,
	}

	results, err := engine.SearchWithFilters("example", filters)
	require.NoError(t, err)

	// Verificar que todos tienen al menos 10 seeds
	for _, r := range results {
		assert.GreaterOrEqual(t, r.Seeds, 10)
	}
}

// TestFilters_MinSeeds prueba filtro de seeds mínimos
func TestFilters_MinSeeds(t *testing.T) {
	cfg := config.DefaultConfig()
	engine := New(cfg)

	filters := Filters{
		MinSeeds: 100,
	}

	results, err := engine.SearchWithFilters("example", filters)
	require.NoError(t, err)

	for _, r := range results {
		assert.GreaterOrEqual(t, r.Seeds, 100)
	}
}

// TestFilters_MaxSize prueba filtro de tamaño máximo
func TestFilters_MaxSize(t *testing.T) {
	cfg := config.DefaultConfig()
	engine := New(cfg)

	maxSize := int64(1024 * 1024 * 1024) // 1 GB
	filters := Filters{
		MaxSize: maxSize,
	}

	results, err := engine.SearchWithFilters("example", filters)
	require.NoError(t, err)

	for _, r := range results {
		assert.LessOrEqual(t, r.Size, maxSize)
	}
}

// TestSortResults prueba ordenamiento de resultados
func TestSortResults(t *testing.T) {
	cfg := config.DefaultConfig()
	engine := New(cfg)

	tests := []struct {
		name    string
		sortBy  SortOrder
		checkFn func([]Result) bool
	}{
		{
			name:   "ordenar por seeds",
			sortBy: SortBySeeds,
			checkFn: func(results []Result) bool {
				for i := 0; i < len(results)-1; i++ {
					if results[i].Seeds < results[i+1].Seeds {
						return false
					}
				}
				return true
			},
		},
		{
			name:   "ordenar por tamaño",
			sortBy: SortBySize,
			checkFn: func(results []Result) bool {
				for i := 0; i < len(results)-1; i++ {
					if results[i].Size < results[i+1].Size {
						return false
					}
				}
				return true
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filters := Filters{SortBy: tt.sortBy}
			results, err := engine.SearchWithFilters("example", filters)
			require.NoError(t, err)

			if len(results) > 1 {
				assert.True(t, tt.checkFn(results), "Resultados mal ordenados")
			}
		})
	}
}

// TestRankResults prueba ranking de resultados
func TestRankResults(t *testing.T) {
	cfg := config.DefaultConfig()
	engine := New(cfg)

	results := []Result{
		{Name: "Low", Seeds: 10, Leechers: 5},
		{Name: "High", Seeds: 100, Leechers: 50},
		{Name: "Medium", Seeds: 50, Leechers: 25},
	}

	ranked := engine.RankResults(results)

	// El primero debería ser el de más seeds
	assert.Equal(t, "High", ranked[0].Name)
	assert.Equal(t, "Low", ranked[2].Name)
}

// TestClearCache prueba limpiar el caché
func TestClearCache(t *testing.T) {
	cfg := config.DefaultConfig()
	engine := New(cfg)

	// Agregar algo al caché
	engine.Search("test")

	stats := engine.Stats()
	assert.Equal(t, 1, stats.CachedQueries)

	// Limpiar caché
	engine.ClearCache()

	stats = engine.Stats()
	assert.Equal(t, 0, stats.CachedQueries)
}

// TestFormatSize prueba formato de tamaños
func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1024 * 1024, "1.0 MB"},
		{1024 * 1024 * 1024, "1.0 GB"},
		{1536 * 1024 * 1024, "1.5 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := FormatSize(tt.bytes)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestHealthScore prueba cálculo de score de salud
func TestHealthScore(t *testing.T) {
	tests := []struct {
		name     string
		result   Result
		minScore int
	}{
		{
			name:     "muchos seeds",
			result:   Result{Seeds: 100, Leechers: 10},
			minScore: 80,
		},
		{
			name:     "pocos seeds",
			result:   Result{Seeds: 5, Leechers: 50},
			minScore: 0,
		},
		{
			name:     "sin seeds",
			result:   Result{Seeds: 0, Leechers: 100},
			minScore: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := tt.result.HealthScore()
			assert.GreaterOrEqual(t, score, tt.minScore)
			assert.LessOrEqual(t, score, 100)
		})
	}
}

// TestCacheExpiration prueba expiración del caché
func TestCacheExpiration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test que requiere tiempo")
	}

	cfg := config.DefaultConfig()
	engine := New(cfg)

	// Agregar al caché
	engine.Search("test")

	// Modificar timestamp manualmente para simular expiración
	engine.cache.mu.Lock()
	for _, entry := range engine.cache.entries {
		entry.timestamp = time.Now().Add(-20 * time.Minute)
	}
	engine.cache.mu.Unlock()

	// Buscar de nuevo (debería estar expirado)
	engine.Search("test")

	// Verificar que se renovó
	stats := engine.Stats()
	assert.Equal(t, 1, stats.CachedQueries)
}
