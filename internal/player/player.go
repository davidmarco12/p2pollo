package player

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"sync"

	"github.com/davidmarco12/p2pollo/internal/config"
	"github.com/sirupsen/logrus"
)

// MPV controlador de mpv
type MPV struct {
	config  *config.Config
	log     *logrus.Logger
	cmd     *exec.Cmd
	running bool
	mu      sync.RWMutex
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
		config: cfg,
		log:    log,
	}, nil
}

// Play inicia reproducción desde un reader (modo stdin/pipe)
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

	// Copiar datos del reader al archivo temporal
	go func() {
		defer tmpFile.Close()
		if _, err := io.Copy(tmpFile, reader); err != nil {
			m.log.Errorf("Error copiando a archivo temporal: %v", err)
		}
	}()

	// Construir argumentos de mpv
	args := []string{
		"--force-window=immediate",
		"--cache=yes",
		"--cache-secs=60",
		tmpPath,
	}
	args = append(args, m.config.Player.MPVOptions...)

	m.cmd = exec.Command(m.config.Player.MPVPath, args...)
	m.cmd.Stderr = os.Stderr

	if err := m.cmd.Start(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("error iniciando mpv: %w", err)
	}

	m.running = true

	// Limpiar archivo temporal cuando mpv termina
	go func() {
		m.cmd.Wait()
		os.Remove(tmpPath)
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

	args := []string{
		"--force-window=yes",
		"--cache=yes",
		"--cache-secs=120",
		"--force-media-title=P2Pollo Streaming",
		"--ytdl=no",
		"--keepaspect=yes",
		"--pause=no",
		filepath,
	}
	args = append(args, m.config.Player.MPVOptions...)

	mpvPath := m.config.Player.MPVPath
	if mpvPath == "" {
		mpvPath = "mpv"
	}

	// En Windows, usar StartProcess directamente para mejor control
	if runtime.GOOS == "windows" {
		fullPath, err := exec.LookPath(mpvPath)
		if err != nil {
			return fmt.Errorf("mpv no encontrado en PATH: %w", err)
		}

		procAttr := &os.ProcAttr{
			Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
		}

		proc, err := os.StartProcess(fullPath, append([]string{fullPath}, args...), procAttr)
		if err != nil {
			return fmt.Errorf("error iniciando mpv: %w", err)
		}

		m.cmd = &exec.Cmd{
			Path:    fullPath,
			Args:    append([]string{fullPath}, args...),
			Stdout:  os.Stdout,
			Stderr:  os.Stderr,
			Stdin:   os.Stdin,
			Process: proc,
		}
	} else {
		m.cmd = exec.Command(mpvPath, args...)

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

		if err := m.cmd.Start(); err != nil {
			return fmt.Errorf("error iniciando mpv: %w", err)
		}
	}

	m.running = true
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

	if m.cmd != nil && m.cmd.Process != nil {
		if err := m.cmd.Process.Kill(); err != nil {
			m.log.Warnf("Error deteniendo mpv: %v", err)
		}
	}

	m.running = false
	return nil
}

// Wait espera a que mpv termine
func (m *MPV) Wait() error {
	if m.cmd == nil {
		return nil
	}
	return m.cmd.Wait()
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
