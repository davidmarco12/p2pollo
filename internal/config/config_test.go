package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDefaultConfig prueba la creación de una configuración con valores por defecto
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.NotNil(t, cfg)
	assert.Equal(t, 0, cfg.Client.Port, "Puerto por defecto debería ser 0 (aleatorio)")
	assert.Greater(t, len(cfg.Trackers), 0, "Debería tener al menos un tracker")
	assert.Equal(t, "info", cfg.Logging.Level)
	assert.Equal(t, 10, cfg.Streaming.InitialBufferSize)
}

// TestLoadConfig prueba la carga de configuración desde archivo
func TestLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Crear archivo de configuración de prueba
	configData := `trackers:
  - "udp://test.tracker.com:1337"
client:
  port: 6881
  max_connections: 100        
  connection_timeout: 30     
  download_rate_limit: 0      
  upload_rate_limit: 0        
player:
  mpv_path: "mpv"
streaming:
  initial_buffer_size: 10
  min_buffer_size: 5
  sequential_download: true
  readahead_pieces: 50
search:
  max_results_per_tracker: 20
  search_timeout: 30
paths:
  cache_dir: "/tmp/test"
  log_dir: "/tmp/test/logs"
  config_file: "/tmp/test/config.yaml"
logging:
  level: "info"
  format: "text"
  file_logging: true
`
	err := os.WriteFile(configPath, []byte(configData), 0644)
	require.NoError(t, err)

	// Cargar configuración
	cfg, err := Load(configPath)
	require.NoError(t, err)
	assert.Equal(t, 6881, cfg.Client.Port)
	assert.Contains(t, cfg.Trackers, "udp://test.tracker.com:1337")
}

// TestLoadConfig_FileNotExists prueba carga cuando el archivo no existe
func TestLoadConfig_FileNotExists(t *testing.T) {
	cfg, err := Load("/path/que/no/existe/config.yaml")
	require.NoError(t, err, "Debería retornar config por defecto si no existe")
	assert.NotNil(t, cfg)
	assert.Greater(t, len(cfg.Trackers), 0)
}

// TestValidateConfig prueba la validación de configuración
func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name:    "configuración válida",
			config:  DefaultConfig(),
			wantErr: false,
		},
		{
			name: "puerto inválido negativo",
			config: &Config{
				Trackers: []string{"udp://test:1337"},
				Client:   ClientConfig{Port: -1, MaxConnections: 100, ConnectionTimeout: 30},
				Streaming: StreamingConfig{
					InitialBufferSize: 10,
					MinBufferSize:     5,
					ReadaheadPieces:   50,
				},
				Search:  SearchConfig{MaxResultsPerTracker: 20, SearchTimeout: 30},
				Logging: LoggingConfig{Level: "info", Format: "text"},
			},
			wantErr: true,
			errMsg:  "puerto inválido",
		},
		{
			name: "puerto muy alto",
			config: &Config{
				Trackers: []string{"udp://test:1337"},
				Client:   ClientConfig{Port: 99999, MaxConnections: 100, ConnectionTimeout: 30},
				Streaming: StreamingConfig{
					InitialBufferSize: 10,
					MinBufferSize:     5,
					ReadaheadPieces:   50,
				},
				Search:  SearchConfig{MaxResultsPerTracker: 20, SearchTimeout: 30},
				Logging: LoggingConfig{Level: "info", Format: "text"},
			},
			wantErr: true,
			errMsg:  "puerto inválido",
		},
		{
			name: "sin trackers",
			config: &Config{
				Trackers: []string{},
				Client:   ClientConfig{Port: 6881, MaxConnections: 100, ConnectionTimeout: 30},
				Streaming: StreamingConfig{
					InitialBufferSize: 10,
					MinBufferSize:     5,
					ReadaheadPieces:   50,
				},
				Search:  SearchConfig{MaxResultsPerTracker: 20, SearchTimeout: 30},
				Logging: LoggingConfig{Level: "info", Format: "text"},
			},
			wantErr: true,
			errMsg:  "debe haber al menos un tracker",
		},
		{
			name: "buffer mínimo mayor que inicial",
			config: &Config{
				Trackers: []string{"udp://test:1337"},
				Client:   ClientConfig{Port: 6881, MaxConnections: 100, ConnectionTimeout: 30},
				Streaming: StreamingConfig{
					InitialBufferSize: 5,
					MinBufferSize:     10,
					ReadaheadPieces:   50,
				},
				Search:  SearchConfig{MaxResultsPerTracker: 20, SearchTimeout: 30},
				Logging: LoggingConfig{Level: "info", Format: "text"},
			},
			wantErr: true,
			errMsg:  "min_buffer_size no puede ser mayor que initial_buffer_size",
		},
		{
			name: "nivel de log inválido",
			config: &Config{
				Trackers: []string{"udp://test:1337"},
				Client:   ClientConfig{Port: 6881, MaxConnections: 100, ConnectionTimeout: 30},
				Streaming: StreamingConfig{
					InitialBufferSize: 10,
					MinBufferSize:     5,
					ReadaheadPieces:   50,
				},
				Search:  SearchConfig{MaxResultsPerTracker: 20, SearchTimeout: 30},
				Logging: LoggingConfig{Level: "invalid", Format: "text"},
			},
			wantErr: true,
			errMsg:  "nivel de log inválido",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestSaveConfig prueba el guardado de configuración
func TestSaveConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg := DefaultConfig()
	cfg.Client.Port = 6881
	cfg.Trackers = []string{"udp://tracker.com:1337"}

	err := cfg.Save(configPath)
	require.NoError(t, err)

	// Verificar que el archivo existe
	_, err = os.Stat(configPath)
	assert.NoError(t, err)

	// Cargar y verificar
	loadedCfg, err := Load(configPath)
	require.NoError(t, err)
	assert.Equal(t, cfg.Client.Port, loadedCfg.Client.Port)
	assert.Equal(t, cfg.Trackers, loadedCfg.Trackers)
}

// TestAddTracker prueba agregar trackers
func TestAddTracker(t *testing.T) {
	cfg := DefaultConfig()
	initialCount := len(cfg.Trackers)

	// Agregar nuevo tracker
	cfg.AddTracker("udp://new.tracker:1337")
	assert.Equal(t, initialCount+1, len(cfg.Trackers))
	assert.Contains(t, cfg.Trackers, "udp://new.tracker:1337")

	// Agregar duplicado (no debería agregarse)
	cfg.AddTracker("udp://new.tracker:1337")
	assert.Equal(t, initialCount+1, len(cfg.Trackers), "No debería agregar duplicados")
}

// TestRemoveTracker prueba eliminar trackers
func TestRemoveTracker(t *testing.T) {
	cfg := DefaultConfig()
	testTracker := "udp://test.tracker:1337"

	cfg.AddTracker(testTracker)
	assert.Contains(t, cfg.Trackers, testTracker)

	// Remover tracker existente
	removed := cfg.RemoveTracker(testTracker)
	assert.True(t, removed, "Debería retornar true al remover tracker existente")
	assert.NotContains(t, cfg.Trackers, testTracker)

	// Intentar remover tracker que no existe
	removed = cfg.RemoveTracker("udp://noexiste:1337")
	assert.False(t, removed, "Debería retornar false al intentar remover tracker inexistente")
}

// TestEnsureDirectories prueba la creación de directorios
func TestEnsureDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := DefaultConfig()
	cfg.Paths.CacheDir = filepath.Join(tmpDir, "cache")
	cfg.Paths.LogDir = filepath.Join(tmpDir, "logs")
	cfg.Paths.ConfigFile = filepath.Join(tmpDir, "config", "config.yaml")

	err := cfg.EnsureDirectories()
	require.NoError(t, err)

	// Verificar que los directorios existen
	_, err = os.Stat(cfg.Paths.CacheDir)
	assert.NoError(t, err, "CacheDir debería existir")

	_, err = os.Stat(cfg.Paths.LogDir)
	assert.NoError(t, err, "LogDir debería existir")

	_, err = os.Stat(filepath.Dir(cfg.Paths.ConfigFile))
	assert.NoError(t, err, "Config directory debería existir")
}
