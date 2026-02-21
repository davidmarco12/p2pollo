// Package player proporciona un controlador de mpv via IPC (JSON) para
// reproducción de video con control programático desde Wails.
package player

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/davidmarco12/p2pollo/internal/config"
	"github.com/sirupsen/logrus"
)

// MPV controlador de mpv con comunicación via IPC
type MPV struct {
	cfg    *config.Config
	log    *logrus.Logger
	cmd    *exec.Cmd
	ipc    *IPCClient
	mu     sync.RWMutex
	cancel context.CancelFunc
}

// New crea un nuevo controlador de mpv con IPC
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

	if _, err := exec.LookPath(mpvPath); err != nil {
		return nil, fmt.Errorf("mpv no encontrado en PATH: %w\nDescarga desde https://mpv.io", err)
	}

	return &MPV{
		cfg: cfg,
		log: log,
	}, nil
}

// Start inicia mpv en modo idle con IPC habilitado.
// No carga ningún archivo inicialmente, solo espera comandos via IPC.
func (m *MPV) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cmd != nil {
		return fmt.Errorf("mpv ya está ejecutándose")
	}

	// Generar path para IPC socket/pipe
	ipcPath := m.getIPCPath()

	// Limpiar socket previo si existe (Unix)
	if runtime.GOOS != "windows" {
		os.Remove(ipcPath)
	}

	mpvPath := m.cfg.Player.MPVPath
	if mpvPath == "" {
		mpvPath = "mpv"
	}

	// Argumentos de mpv:
	// --idle: no cerrar cuando no hay archivo cargado
	// --force-window: crear ventana inmediatamente
	// --input-ipc-server: habilitar IPC
	// --no-terminal: no usar terminal para input
	args := []string{
		"--idle",
		"--force-window=immediate",
		"--input-ipc-server=" + ipcPath,
		"--no-terminal",
		"--keep-open=yes",
		"--cache=yes",
		"--cache-secs=120",
	}

	// Agregar opciones custom del config
	args = append(args, m.cfg.Player.MPVOptions...)

	m.log.Infof("Iniciando mpv con IPC: %s", ipcPath)

	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel

	m.cmd = exec.CommandContext(ctx, mpvPath, args...)

	// En desarrollo, mostrar stderr de mpv
	if m.cfg.Logging.Level == "debug" {
		m.cmd.Stderr = os.Stderr
		m.cmd.Stdout = os.Stdout
	}

	if err := m.cmd.Start(); err != nil {
		m.cancel()
		m.cancel = nil
		m.cmd = nil
		return fmt.Errorf("error iniciando mpv: %w", err)
	}

	// Crear cliente IPC
	ipc, err := NewIPCClient(ipcPath, m.log)
	if err != nil {
		m.cmd.Process.Kill()
		m.cancel()
		m.cancel = nil
		m.cmd = nil
		return fmt.Errorf("error conectando IPC: %w", err)
	}

	m.ipc = ipc

	// Monitor de proceso mpv (limpiar cuando termina)
	go func() {
		m.cmd.Wait()
		m.mu.Lock()
		m.log.Info("mpv terminó")
		if m.ipc != nil {
			m.ipc.Close()
			m.ipc = nil
		}
		m.cmd = nil
		if m.cancel != nil {
			m.cancel()
			m.cancel = nil
		}
		m.mu.Unlock()
	}()

	m.log.Info("mpv iniciado correctamente")
	return nil
}

// LoadFile carga un archivo o URL en mpv
func (m *MPV) LoadFile(path string) error {
	m.mu.RLock()
	ipc := m.ipc
	m.mu.RUnlock()

	if ipc == nil {
		return fmt.Errorf("mpv no está ejecutándose")
	}

	m.log.Infof("Cargando archivo: %s", path)
	return ipc.Command("loadfile", path)
}

// Command envía un comando genérico a mpv via IPC
func (m *MPV) Command(name string, args ...interface{}) error {
	m.mu.RLock()
	ipc := m.ipc
	m.mu.RUnlock()

	if ipc == nil {
		return fmt.Errorf("mpv no está ejecutándose")
	}

	return ipc.Command(name, args...)
}

// GetProperty obtiene el valor de una propiedad de mpv
func (m *MPV) GetProperty(name string) (interface{}, error) {
	m.mu.RLock()
	ipc := m.ipc
	m.mu.RUnlock()

	if ipc == nil {
		return nil, fmt.Errorf("mpv no está ejecutándose")
	}

	return ipc.GetProperty(name)
}

// SetProperty establece el valor de una propiedad de mpv
func (m *MPV) SetProperty(name string, value interface{}) error {
	m.mu.RLock()
	ipc := m.ipc
	m.mu.RUnlock()

	if ipc == nil {
		return fmt.Errorf("mpv no está ejecutándose")
	}

	return ipc.SetProperty(name, value)
}

// Stop detiene mpv completamente
func (m *MPV) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cmd == nil {
		return nil
	}

	m.log.Info("Deteniendo mpv")

	// Cerrar IPC primero
	if m.ipc != nil {
		m.ipc.Close()
		m.ipc = nil
	}

	// Cancelar contexto (mata el proceso)
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}

	// Esperar a que termine (timeout de 2s)
	done := make(chan struct{})
	go func() {
		if m.cmd != nil {
			m.cmd.Wait()
		}
		close(done)
	}()

	timeout := time.After(2 * time.Second)
	select {
	case <-done:
		m.log.Info("mpv detenido correctamente")
	case <-timeout:
		m.log.Warn("Timeout esperando que mpv termine, forzando kill")
		if m.cmd != nil && m.cmd.Process != nil {
			m.cmd.Process.Kill()
		}
	}

	m.cmd = nil
	return nil
}

// IsRunning indica si mpv está actualmente ejecutándose
func (m *MPV) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cmd != nil && m.ipc != nil
}

// getIPCPath retorna el path del socket/pipe para IPC según la plataforma
func (m *MPV) getIPCPath() string {
	pid := os.Getpid()

	if runtime.GOOS == "windows" {
		// Named pipe en Windows
		return fmt.Sprintf(`\\.\pipe\mpv-p2pollo-%d`, pid)
	}

	// Unix socket en Linux/macOS
	tmpDir := os.TempDir()
	return filepath.Join(tmpDir, fmt.Sprintf("mpv-p2pollo-%d.sock", pid))
}
