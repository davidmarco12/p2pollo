// Package player proporciona un controlador de mpv usando libmpv (embedding)
package player

import (
	"fmt"
	"os"
	"strings"
	"sync"

	mpv "github.com/gen2brain/go-mpv"
	"github.com/davidmarco12/p2pollo/internal/config"
	"github.com/sirupsen/logrus"
)

// LibMPV controlador de mpv usando libmpv (embedded player)
type LibMPV struct {
	cfg         *config.Config
	log         *logrus.Logger
	mpv         *mpv.Mpv
	hwnd        uintptr // HWND sugerido (ventana principal de Wails)
	childHWND   uintptr // HWND del child window creado para mpv
	mu          sync.RWMutex
}

// NewLibMPV crea un nuevo controlador usando libmpv
func NewLibMPV(cfg *config.Config) (*LibMPV, error) {
	log := logrus.New()
	log.SetLevel(logrus.InfoLevel)
	if cfg.Logging.Level == "debug" {
		log.SetLevel(logrus.DebugLevel)
	}

	// Crear instancia de mpv
	m := mpv.New()
	if m == nil {
		return nil, fmt.Errorf("mpv.New() retornó nil")
	}

	return &LibMPV{
		cfg: cfg,
		log: log,
		mpv: m,
	}, nil
}

// SetHWND establece el HWND donde renderizar el video
func (l *LibMPV) SetHWND(hwnd uintptr) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.hwnd = hwnd
}

// Start inicializa mpv con la configuración necesaria
func (l *LibMPV) Start() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Si mpv fue destruido previamente (por Stop()), recrearlo
	if l.mpv == nil {
		l.log.Info("Recreando instancia de mpv")
		l.mpv = mpv.New()
		if l.mpv == nil {
			return fmt.Errorf("error recreando mpv: mpv.New() retornó nil")
		}
	}

	// Habilitar logging verbose de mpv para diagnóstico
	l.mpv.SetOptionString("terminal", "yes")
	l.mpv.SetOptionString("msg-level", "all=v")
	l.log.Info("Logging verbose de mpv habilitado")

	// Configurar opciones de mpv
	l.mpv.SetOptionString("keep-open", "yes")
	l.mpv.SetOptionString("cache", "yes")
	l.mpv.SetOptionString("cache-secs", "120")

	// Configurar ventana de mpv (separada)
	// Nota: El embedding en Wails/WebView2 es complejo debido a la arquitectura de renderizado.
	// Usamos ventana separada que es el método estándar y más confiable para mpv.
	l.mpv.SetOptionString("force-window", "immediate")
	l.mpv.SetOptionString("ontop", "yes")           // Mantener encima de otras ventanas
	l.mpv.SetOptionString("border", "yes")           // Mostrar bordes de ventana
	l.mpv.SetOptionString("title", "p2pollo - Video Player") // Título personalizado
	l.mpv.SetOptionString("geometry", "960x540")     // Tamaño inicial
	l.log.Info("Creando ventana de mpv (optimizada para mejor UX)")

	// Inicializar mpv
	if err := l.mpv.Initialize(); err != nil {
		return fmt.Errorf("error inicializando libmpv: %w", err)
	}

	l.log.Info("libmpv inicializado correctamente")

	// Verificar que el sistema de comandos funciona
	l.log.Info("Probando sistema de comandos con get_property version")
	if version, err := l.mpv.GetProperty("mpv-version", mpv.FormatString); err != nil {
		l.log.Warnf("No se pudo obtener mpv-version: %v", err)
	} else {
		l.log.Infof("mpv version: %v", version)
	}

	return nil
}

// LoadFile carga un archivo para reproducción
func (l *LibMPV) LoadFile(path string) error {
	l.mu.RLock()
	m := l.mpv
	l.mu.RUnlock()

	if m == nil {
		return fmt.Errorf("mpv no inicializado")
	}

	// Verificar que el archivo existe
	fileInfo, err := os.Stat(path)
	if err != nil {
		l.log.Errorf("Archivo no existe o no es accesible: %s, error: %v", path, err)
		return fmt.Errorf("archivo no accesible: %w", err)
	}
	l.log.Infof("Archivo encontrado: %s (tamaño: %d bytes)", path, fileInfo.Size())

	// Convertir path de Windows a formato Unix (mpv prefiere forward slashes)
	unixPath := strings.ReplaceAll(path, "\\", "/")
	l.log.Infof("Path convertido a formato Unix: %s", unixPath)

	// DIAGNÓSTICO: Intentar varios métodos de carga

	// Método 1: Command con path Unix + flag "replace"
	l.log.Info("Método 1: Intentando loadfile con path Unix + flag 'replace'")
	if err := m.Command([]string{"loadfile", unixPath, "replace"}); err != nil {
		l.log.Warnf("Método 1 falló: %v", err)

		// Intentar obtener mensaje de error de mpv
		if errStr, propErr := m.GetProperty("error-string", mpv.FormatString); propErr == nil {
			l.log.Errorf("mpv error-string: %v", errStr)
		}
	} else {
		l.log.Info("✓ Método 1 exitoso - archivo cargado correctamente")
		return nil
	}

	// Método 2: Command con path Windows + flag "replace"
	l.log.Info("Método 2: Intentando loadfile con path Windows + flag 'replace'")
	if err := m.Command([]string{"loadfile", path, "replace"}); err != nil {
		l.log.Warnf("Método 2 falló: %v", err)
	} else {
		l.log.Info("✓ Método 2 exitoso - archivo cargado correctamente")
		return nil
	}

	// Todos los métodos fallaron
	return fmt.Errorf("todos los métodos de carga fallaron para: %s", path)
}

// Command envía un comando genérico a mpv
func (l *LibMPV) Command(name string, args ...interface{}) error {
	l.mu.RLock()
	m := l.mpv
	l.mu.RUnlock()

	if m == nil {
		return fmt.Errorf("mpv no inicializado")
	}

	// Construir array de strings para el comando
	cmdArgs := make([]string, 0, len(args)+1)
	cmdArgs = append(cmdArgs, name)
	for _, arg := range args {
		cmdArgs = append(cmdArgs, fmt.Sprintf("%v", arg))
	}

	return m.Command(cmdArgs)
}

// GetProperty obtiene el valor de una propiedad de mpv
func (l *LibMPV) GetProperty(name string) (interface{}, error) {
	l.mu.RLock()
	m := l.mpv
	l.mu.RUnlock()

	if m == nil {
		return nil, fmt.Errorf("mpv no inicializado")
	}

	// go-mpv usa GetProperty con formato especificado
	val, err := m.GetProperty(name, mpv.FormatString)
	if err != nil {
		return nil, err
	}

	return val, nil
}

// SetProperty establece el valor de una propiedad de mpv
func (l *LibMPV) SetProperty(name string, value interface{}) error {
	l.mu.RLock()
	m := l.mpv
	l.mu.RUnlock()

	if m == nil {
		return fmt.Errorf("mpv no inicializado")
	}

	return m.SetPropertyString(name, fmt.Sprintf("%v", value))
}

// Stop detiene mpv y libera recursos
func (l *LibMPV) Stop() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.mpv == nil {
		return nil
	}

	l.log.Info("Deteniendo libmpv")
	l.mpv.TerminateDestroy()
	l.mpv = nil

	return nil
}

// IsRunning indica si mpv está actualmente ejecutándose
func (l *LibMPV) IsRunning() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.mpv != nil
}

// --- Métodos helper para compatibilidad con API existente ---

// Play resume la reproducción
func (l *LibMPV) Play() error {
	return l.SetProperty("pause", "no")
}

// Pause pausa la reproducción
func (l *LibMPV) Pause() error {
	return l.SetProperty("pause", "yes")
}

// TogglePause alterna entre play y pausa
func (l *LibMPV) TogglePause() error {
	return l.Command("cycle", "pause")
}

// Seek salta a una posición específica (en segundos)
func (l *LibMPV) Seek(seconds float64, absolute bool) error {
	mode := "relative"
	if absolute {
		mode = "absolute"
	}
	return l.Command("seek", seconds, mode)
}

// SetVolume establece el volumen (0-100)
func (l *LibMPV) SetVolume(volume int) error {
	if volume < 0 {
		volume = 0
	}
	if volume > 100 {
		volume = 100
	}
	return l.SetProperty("volume", volume)
}

// ToggleFullscreen alterna pantalla completa
func (l *LibMPV) ToggleFullscreen() error {
	return l.Command("cycle", "fullscreen")
}

// GetTimePos obtiene la posición actual de reproducción (en segundos)
func (l *LibMPV) GetTimePos() (float64, error) {
	val, err := l.GetProperty("time-pos")
	if err != nil {
		return 0, err
	}

	switch v := val.(type) {
	case float64:
		return v, nil
	case string:
		// Intentar parsear
		var f float64
		fmt.Sscanf(v, "%f", &f)
		return f, nil
	default:
		return 0, fmt.Errorf("time-pos tipo inesperado: %T", val)
	}
}

// GetDuration obtiene la duración total del video (en segundos)
func (l *LibMPV) GetDuration() (float64, error) {
	val, err := l.GetProperty("duration")
	if err != nil {
		return 0, err
	}

	switch v := val.(type) {
	case float64:
		return v, nil
	case string:
		var f float64
		fmt.Sscanf(v, "%f", &f)
		return f, nil
	default:
		return 0, fmt.Errorf("duration tipo inesperado: %T", val)
	}
}

// IsPaused indica si la reproducción está pausada
func (l *LibMPV) IsPaused() (bool, error) {
	val, err := l.GetProperty("pause")
	if err != nil {
		return false, err
	}

	switch v := val.(type) {
	case bool:
		return v, nil
	case string:
		return v == "yes", nil
	default:
		return false, fmt.Errorf("pause tipo inesperado: %T", val)
	}
}

// GetVolume obtiene el volumen actual (0-100)
func (l *LibMPV) GetVolume() (int, error) {
	val, err := l.GetProperty("volume")
	if err != nil {
		return 0, err
	}

	switch v := val.(type) {
	case float64:
		return int(v), nil
	case int:
		return v, nil
	case string:
		var i int
		fmt.Sscanf(v, "%d", &i)
		return i, nil
	default:
		return 0, fmt.Errorf("volume tipo inesperado: %T", val)
	}
}

// GetTracks obtiene la lista de todos los tracks (usa track-list de mpv)
func (l *LibMPV) GetTracks() ([]TrackInfo, error) {
	// TODO: Implementar parsing de track-list de mpv
	// Por ahora retornar vacío
	return []TrackInfo{}, nil
}

// GetSubtitleTracks obtiene solo los tracks de subtítulos
func (l *LibMPV) GetSubtitleTracks() ([]TrackInfo, error) {
	allTracks, err := l.GetTracks()
	if err != nil {
		return nil, err
	}

	var subs []TrackInfo
	for _, track := range allTracks {
		if track.Type == "sub" {
			subs = append(subs, track)
		}
	}

	return subs, nil
}
