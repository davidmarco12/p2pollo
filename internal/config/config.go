package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// Config representa la configuración completa de la aplicación
type Config struct {
	Trackers  []string        `yaml:"trackers" mapstructure:"trackers"`
	Client    ClientConfig    `yaml:"client" mapstructure:"client"`
	Player    PlayerConfig    `yaml:"player" mapstructure:"player"`
	Streaming StreamingConfig `yaml:"streaming" mapstructure:"streaming"`
	Search    SearchConfig    `yaml:"search" mapstructure:"search"`
	Paths     PathsConfig     `yaml:"paths" mapstructure:"paths"`
	Logging   LoggingConfig   `yaml:"logging" mapstructure:"logging"`
}

// ClientConfig configuración del cliente torrent
type ClientConfig struct {
	Port              int `yaml:"port" mapstructure:"port"`
	DownloadRateLimit int `yaml:"download_rate_limit" mapstructure:"download_rate_limit"` // KB/s, 0 = sin límite
	UploadRateLimit   int `yaml:"upload_rate_limit" mapstructure:"upload_rate_limit"`     // KB/s, 0 = sin límite
	MaxConnections    int `yaml:"max_connections" mapstructure:"max_connections"`
	ConnectionTimeout int `yaml:"connection_timeout" mapstructure:"connection_timeout"` // segundos
}

// PlayerConfig configuración del reproductor
type PlayerConfig struct {
	MPVPath    string   `yaml:"mpv_path" mapstructure:"mpv_path"`
	MPVOptions []string `yaml:"mpv_options" mapstructure:"mpv_options"`
}

// StreamingConfig configuración del streaming
type StreamingConfig struct {
	InitialBufferSize  int  `yaml:"initial_buffer_size" mapstructure:"initial_buffer_size"` // MB
	MinBufferSize      int  `yaml:"min_buffer_size" mapstructure:"min_buffer_size"`         // MB
	SequentialDownload bool `yaml:"sequential_download" mapstructure:"sequential_download"`
	ReadaheadPieces    int  `yaml:"readahead_pieces" mapstructure:"readahead_pieces"`
}

// SearchConfig configuración de búsqueda
type SearchConfig struct {
	MaxResultsPerTracker int `yaml:"max_results_per_tracker" mapstructure:"max_results_per_tracker"`
	SearchTimeout        int `yaml:"search_timeout" mapstructure:"search_timeout"` // segundos
}

// PathsConfig rutas de archivos y directorios
type PathsConfig struct {
	CacheDir   string `yaml:"cache_dir" mapstructure:"cache_dir"`
	LogDir     string `yaml:"log_dir" mapstructure:"log_dir"`
	ConfigFile string `yaml:"config_file" mapstructure:"config_file"`
}

// LoggingConfig configuración de logging
type LoggingConfig struct {
	Level       string `yaml:"level" mapstructure:"level"`   // debug, info, warn, error
	Format      string `yaml:"format" mapstructure:"format"` // text, json
	FileLogging bool   `yaml:"file_logging" mapstructure:"file_logging"`
}

// DefaultConfig retorna una configuración con valores por defecto
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()

	return &Config{
		Trackers: []string{
			"udp://tracker.opentrackr.org:1337/announce",
			"udp://open.stealth.si:80/announce",
			"udp://tracker.torrent.eu.org:451/announce",
			"udp://tracker.moeking.me:6969/announce",
		},
		Client: ClientConfig{
			Port:              0, // 0 = puerto aleatorio
			DownloadRateLimit: 0,
			UploadRateLimit:   0,
			MaxConnections:    200,
			ConnectionTimeout: 30,
		},
		Player: PlayerConfig{
			MPVPath: "mpv",
			MPVOptions: []string{
				"--cache=yes",
				"--demuxer-max-bytes=150M",
				"--demuxer-max-back-bytes=75M",
			},
		},
		Streaming: StreamingConfig{
			InitialBufferSize:  10,
			MinBufferSize:      5,
			SequentialDownload: true,
			ReadaheadPieces:    50,
		},
		Search: SearchConfig{
			MaxResultsPerTracker: 20,
			SearchTimeout:        30,
		},
		Paths: PathsConfig{
			CacheDir:   filepath.Join(homeDir, ".cache", "p2pollo"),
			LogDir:     filepath.Join(homeDir, ".local", "share", "p2pollo", "logs"),
			ConfigFile: filepath.Join(homeDir, ".config", "p2pollo", "config.yaml"),
		},
		Logging: LoggingConfig{
			Level:       "info",
			Format:      "text",
			FileLogging: true,
		},
	}
}

// Validate valida la configuración
func (c *Config) Validate() error {
	// Validar trackers
	if len(c.Trackers) == 0 {
		return fmt.Errorf("debe haber al menos un tracker configurado")
	}

	// Validar puerto
	if c.Client.Port < 0 || c.Client.Port > 65535 {
		return fmt.Errorf("puerto inválido: %d (debe estar entre 0-65535)", c.Client.Port)
	}

	// Validar límites de velocidad
	if c.Client.DownloadRateLimit < 0 {
		return fmt.Errorf("límite de descarga inválido: %d", c.Client.DownloadRateLimit)
	}
	if c.Client.UploadRateLimit < 0 {
		return fmt.Errorf("límite de subida inválido: %d", c.Client.UploadRateLimit)
	}

	// Validar conexiones
	if c.Client.MaxConnections <= 0 {
		return fmt.Errorf("max_connections debe ser mayor a 0")
	}
	if c.Client.ConnectionTimeout <= 0 {
		return fmt.Errorf("connection_timeout debe ser mayor a 0")
	}

	// Validar buffer
	if c.Streaming.InitialBufferSize <= 0 {
		return fmt.Errorf("initial_buffer_size debe ser mayor a 0")
	}
	if c.Streaming.MinBufferSize <= 0 {
		return fmt.Errorf("min_buffer_size debe ser mayor a 0")
	}
	if c.Streaming.MinBufferSize > c.Streaming.InitialBufferSize {
		return fmt.Errorf("min_buffer_size no puede ser mayor que initial_buffer_size")
	}

	// Validar readahead
	if c.Streaming.ReadaheadPieces <= 0 {
		return fmt.Errorf("readahead_pieces debe ser mayor a 0")
	}

	// Validar búsqueda
	if c.Search.MaxResultsPerTracker <= 0 {
		return fmt.Errorf("max_results_per_tracker debe ser mayor a 0")
	}
	if c.Search.SearchTimeout <= 0 {
		return fmt.Errorf("search_timeout debe ser mayor a 0")
	}

	// Validar nivel de log
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLevels[c.Logging.Level] {
		return fmt.Errorf("nivel de log inválido: %s (debe ser: debug, info, warn, error)", c.Logging.Level)
	}

	// Validar formato de log
	validFormats := map[string]bool{
		"text": true,
		"json": true,
	}
	if !validFormats[c.Logging.Format] {
		return fmt.Errorf("formato de log inválido: %s (debe ser: text, json)", c.Logging.Format)
	}

	// Validar que mpv existe (si está especificado)
	if c.Player.MPVPath != "" && c.Player.MPVPath != "mpv" {
		if _, err := os.Stat(c.Player.MPVPath); os.IsNotExist(err) {
			return fmt.Errorf("mpv no encontrado en: %s", c.Player.MPVPath)
		}
	}

	return nil
}

// AddTracker agrega un tracker a la lista
func (c *Config) AddTracker(tracker string) {
	// Verificar que no exista ya
	for _, t := range c.Trackers {
		if t == tracker {
			return
		}
	}
	c.Trackers = append(c.Trackers, tracker)
}

// RemoveTracker elimina un tracker de la lista
func (c *Config) RemoveTracker(tracker string) bool {
	for i, t := range c.Trackers {
		if t == tracker {
			c.Trackers = append(c.Trackers[:i], c.Trackers[i+1:]...)
			return true
		}
	}
	return false
}

// Load carga la configuración desde un archivo
func Load(configPath string) (*Config, error) {
	// Si no existe el archivo, retornar configuración por defecto
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return DefaultConfig(), nil
	}

	// Configurar viper
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	// Leer archivo
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error leyendo config: %w", err)
	}

	// Unmarshal a estructura
	config := &Config{}
	if err := viper.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("error parseando config: %w", err)
	}

	// Validar
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuración inválida: %w", err)
	}

	return config, nil
}

// Save guarda la configuración en un archivo
func (c *Config) Save(configPath string) error {
	// Validar antes de guardar
	if err := c.Validate(); err != nil {
		return fmt.Errorf("configuración inválida: %w", err)
	}

	// Crear directorio si no existe
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("error creando directorio: %w", err)
	}

	// Marshal a YAML
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("error serializando config: %w", err)
	}

	// Escribir archivo
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("error escribiendo archivo: %w", err)
	}

	return nil
}

// EnsureDirectories crea los directorios necesarios
func (c *Config) EnsureDirectories() error {
	dirs := []string{
		c.Paths.CacheDir,
		c.Paths.LogDir,
		filepath.Dir(c.Paths.ConfigFile),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("error creando directorio %s: %w", dir, err)
		}
	}

	return nil
}
