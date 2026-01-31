package player

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/davidmarco12/p2pollo/internal/config"
	"github.com/sirupsen/logrus"
)

// MPV controlador de mpv
type MPV struct {
	config    *config.Config
	log       *logrus.Logger
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    io.ReadCloser
	running   bool
	paused    bool
	position  float64
	duration  float64
	mu        sync.RWMutex
	eventChan chan Event
	stopChan  chan struct{}
}

// Event evento del reproductor
type Event struct {
	Type string
	Data interface{}
}

// PlaybackState estado de reproducción
type PlaybackState struct {
	Playing  bool
	Paused   bool
	Position float64 // segundos
	Duration float64 // segundos
	Volume   int     // 0-100
}

// New crea un nuevo controlador de MPV
func New(cfg *config.Config) (*MPV, error) {
	log := logrus.New()
	log.SetLevel(logrus.InfoLevel)
	if cfg.Logging.Level == "debug" {
		log.SetLevel(logrus.DebugLevel)
	}

	// Verificar que mpv existe
	mpvPath := cfg.Player.MPVPath
	if mpvPath == "" {
		mpvPath = "mpv"
	}

	_, err := exec.LookPath(mpvPath)
	if err != nil {
		return nil, fmt.Errorf("mpv no encontrado: %w (instala desde https://mpv.io)", err)
	}

	return &MPV{
		config:    cfg,
		log:       log,
		eventChan: make(chan Event, 100),
		stopChan:  make(chan struct{}),
	}, nil
}

// Play inicia reproducción desde un reader
func (m *MPV) Play(reader io.Reader) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return fmt.Errorf("mpv ya está ejecutándose")
	}

	m.log.Info("Iniciando reproducción con mpv (streaming mode)")

	// Crear un archivo temporal para el streaming
	tmpFile, err := os.CreateTemp("", "mpv-stream-*.mp4")
	if err != nil {
		return fmt.Errorf("error creando archivo temporal: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Copiar datos del reader al archivo temporal en un goroutine
	go func() {
		defer tmpFile.Close()
		_, err := io.Copy(tmpFile, reader)
		if err != nil {
			m.log.Errorf("Error copiando a archivo temporal: %v", err)
		}
	}()

	// Esperar a que se escriban suficientes datos (5MB aprox)
	m.log.Debug("Esperando datos iniciales...")
	time.Sleep(2 * time.Second)

	// Verificar que el archivo tiene datos
	stat, err := os.Stat(tmpPath)
	if err != nil || stat.Size() < 1024*1024 {
		m.log.Warnf("Archivo temporal muy pequeño (%.1f MB), continuando...", float64(stat.Size())/(1024*1024))
	}

	// Construir argumentos de mpv optimizados
	args := []string{
		"--force-window=immediate", // Mostrar ventana inmediatamente
		"--cache=yes",              // Activar cache
		"--cache-secs=60",          // Cache de 60 segundos
		tmpPath,                    // Reproducir archivo temporal
	}

	// Agregar opciones configuradas
	args = append(args, m.config.Player.MPVOptions...)

	// Crear comando
	m.cmd = exec.Command(m.config.Player.MPVPath, args...)

	// Mostrar stderr de MPV para debugging
	m.cmd.Stderr = os.Stderr
	m.log.Debugf("MPV args: %v", args)

	// Iniciar mpv
	err = m.cmd.Start()
	if err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("error iniciando mpv: %w", err)
	}

	m.running = true
	m.paused = false
	m.log.Info("mpv iniciado correctamente para reproducir temporal")

	// Limpiar archivo temporal cuando mpv termina
	go func() {
		m.cmd.Wait()
		time.Sleep(1 * time.Second)
		os.Remove(tmpPath)
		m.log.Debugf("Archivo temporal eliminado: %s", tmpPath)
	}()

	return nil
}

// PlayFile reproduce un archivo directamente
func (m *MPV) PlayFile(filepath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return fmt.Errorf("mpv ya está ejecutándose")
	}

	m.log.Infof("Reproduciendo archivo: %s", filepath)

	// Argumentos optimizados para reproducir archivos en descarga
	args := []string{
		"--force-window=yes",                    // Forzar ventana visible
		"--cache=yes",                           // Activar cache
		"--cache-secs=120",                      // Cache de 120 segundos para tolerar pausas
		"--force-media-title=P2Pollo Streaming", // Título custom (más visible)
		"--ytdl=no",                             // No intentar descargar
		"--keepaspect=yes",                      // Mantener relación de aspecto
		"--pause=no",                            // No pausar al abrir
		filepath,                                // Ruta del archivo
	}
	args = append(args, m.config.Player.MPVOptions...)

	m.log.Debugf("MPV args: %v", args)
	m.log.Infof("Ejecutando: %s %v", m.config.Player.MPVPath, args)

	// Obtener la ruta completa de MPV
	mpvPath := m.config.Player.MPVPath
	if mpvPath == "" {
		mpvPath = "mpv"
	}

	// En Windows, buscar la ruta completa
	if runtime.GOOS == "windows" {
		fullPath, err := exec.LookPath(mpvPath)
		if err != nil {
			return fmt.Errorf("mpv no encontrado en PATH: %w", err)
		}
		mpvPath = fullPath

		m.log.Infof("Ruta completa de MPV: %s", mpvPath)

		// Usar StartProcess directamente para mejor control
		procAttr := &os.ProcAttr{
			Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
		}

		proc, err := os.StartProcess(mpvPath, append([]string{mpvPath}, args...), procAttr)
		if err != nil {
			return fmt.Errorf("error iniciando mpv: %w", err)
		}

		// Guardar el proceso para poder esperar luego
		m.cmd = &exec.Cmd{
			Path:    mpvPath,
			Args:    append([]string{mpvPath}, args...),
			Stdout:  os.Stdout,
			Stderr:  os.Stderr,
			Stdin:   os.Stdin,
			Process: proc,
		}
	} else {
		m.cmd = exec.Command(mpvPath, args...)

		// Capturar para logging
		stdoutPipe, _ := m.cmd.StdoutPipe()
		stderrPipe, _ := m.cmd.StderrPipe()

		go func() {
			if stdoutPipe != nil {
				scanner := bufio.NewScanner(stdoutPipe)
				for scanner.Scan() {
					m.log.Debugf("[MPV stdout] %s", scanner.Text())
				}
			}
		}()

		go func() {
			if stderrPipe != nil {
				scanner := bufio.NewScanner(stderrPipe)
				for scanner.Scan() {
					m.log.Warnf("[MPV stderr] %s", scanner.Text())
				}
			}
		}()

		err := m.cmd.Start()
		if err != nil {
			return fmt.Errorf("error iniciando mpv: %w", err)
		}
	}

	m.running = true
	m.paused = false

	return nil
}

// Pause pausa la reproducción
func (m *MPV) Pause() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return fmt.Errorf("mpv no está ejecutándose")
	}

	m.paused = !m.paused
	m.log.Infof("Pausa: %v", m.paused)

	return nil
}

// Stop detiene la reproducción
func (m *MPV) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return nil
	}

	m.log.Info("Deteniendo mpv")

	close(m.stopChan)

	if m.cmd != nil && m.cmd.Process != nil {
		err := m.cmd.Process.Kill()
		if err != nil {
			m.log.Warnf("Error deteniendo mpv: %v", err)
		}
	}

	m.running = false
	m.paused = false

	return nil
}

// Seek salta a una posición específica (en segundos)
func (m *MPV) Seek(seconds float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return fmt.Errorf("mpv no está ejecutándose")
	}

	m.log.Infof("Seek a: %.2f segundos", seconds)
	m.position = seconds

	return nil
}

// SetVolume establece el volumen (0-100)
func (m *MPV) SetVolume(volume int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return fmt.Errorf("mpv no está ejecutándose")
	}

	if volume < 0 || volume > 100 {
		return fmt.Errorf("volumen debe estar entre 0-100")
	}

	m.log.Infof("Volumen: %d", volume)

	return nil
}

// State retorna el estado actual de reproducción
func (m *MPV) State() PlaybackState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return PlaybackState{
		Playing:  m.running && !m.paused,
		Paused:   m.paused,
		Position: m.position,
		Duration: m.duration,
		Volume:   100,
	}
}

// IsRunning verifica si mpv está ejecutándose
func (m *MPV) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.running
}

// IsPaused verifica si está pausado
func (m *MPV) IsPaused() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.paused
}

// Position retorna la posición actual
func (m *MPV) Position() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.position
}

// Duration retorna la duración total
func (m *MPV) Duration() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.duration
}

// Events retorna el canal de eventos
func (m *MPV) Events() <-chan Event {
	return m.eventChan
}

// Wait espera a que mpv termine
func (m *MPV) Wait() error {
	if m.cmd == nil {
		return nil
	}

	return m.cmd.Wait()
}

// monitorOutput monitorea la salida de mpv
func (m *MPV) monitorOutput() {
	if m.stdout == nil {
		// Si no hay stdout, simplemente retornar
		return
	}

	scanner := bufio.NewScanner(m.stdout)

	for scanner.Scan() {
		select {
		case <-m.stopChan:
			return
		default:
			line := scanner.Text()
			m.parseOutput(line)
		}
	}

	if err := scanner.Err(); err != nil {
		m.log.Debugf("Error leyendo stdout: %v", err)
	}
}

// parseOutput parsea la salida de mp  v
func (m *MPV) parseOutput(line string) {
	m.log.Debugf("mpv output: %s", line)

	// Intentar parsear como JSON (si mpv está en modo JSON)
	var jsonData map[string]interface{}
	if err := json.Unmarshal([]byte(line), &jsonData); err == nil {
		m.handleJSONEvent(jsonData)
	}
}

// handleJSONEvent maneja eventos JSON de mpv
func (m *MPV) handleJSONEvent(data map[string]interface{}) {
	eventType, ok := data["event"].(string)
	if !ok {
		return
	}

	event := Event{
		Type: eventType,
		Data: data,
	}

	select {
	case m.eventChan <- event:
	case <-time.After(100 * time.Millisecond):
		// No bloquear si el canal está lleno
	}
}

// Close cierra el reproductor
func (m *MPV) Close() error {
	return m.Stop()
}

// CheckMPV verifica si mpv está instalado y retorna su versión
func CheckMPV(mpvPath string) (string, error) {
	if mpvPath == "" {
		mpvPath = "mpv"
	}

	cmd := exec.Command(mpvPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("mpv no encontrado o no ejecutable: %w", err)
	}

	return string(output), nil
}
