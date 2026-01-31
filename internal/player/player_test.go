package player

import (
	"strings"
	"testing"

	"github.com/davidmarco12/p2pollo/internal/config"
	"github.com/stretchr/testify/assert"
)

// TestNew prueba la creación del controlador de mpv
func TestNew(t *testing.T) {
	cfg := config.DefaultConfig()

	player, err := New(cfg)

	// Puede fallar si mpv no está instalado
	if err != nil {
		t.Skipf("mpv no disponible: %v", err)
	}

	assert.NotNil(t, player)
}

// TestCheckMPV prueba verificar mpv
func TestCheckMPV(t *testing.T) {
	version, err := CheckMPV("mpv")

	if err != nil {
		t.Skipf("mpv no instalado: %v", err)
	}

	assert.NotEmpty(t, version)
	assert.Contains(t, version, "mpv")
}

// TestState prueba obtener el estado
func TestState(t *testing.T) {
	cfg := config.DefaultConfig()
	player, err := New(cfg)

	if err != nil {
		t.Skip("mpv no disponible")
	}

	state := player.State()

	assert.False(t, state.Playing)
	assert.False(t, state.Paused)
	assert.Equal(t, 0.0, state.Position)
}

// TestIsRunning prueba verificar si está ejecutándose
func TestIsRunning(t *testing.T) {
	cfg := config.DefaultConfig()
	player, err := New(cfg)

	if err != nil {
		t.Skip("mpv no disponible")
	}

	running := player.IsRunning()
	assert.False(t, running)
}

// TestIsPaused prueba verificar si está pausado
func TestIsPaused(t *testing.T) {
	cfg := config.DefaultConfig()
	player, err := New(cfg)

	if err != nil {
		t.Skip("mpv no disponible")
	}

	paused := player.IsPaused()
	assert.False(t, paused)
}

// TestPosition prueba obtener posición
func TestPosition(t *testing.T) {
	cfg := config.DefaultConfig()
	player, err := New(cfg)

	if err != nil {
		t.Skip("mpv no disponible")
	}

	pos := player.Position()
	assert.Equal(t, 0.0, pos)
}

// TestDuration prueba obtener duración
func TestDuration(t *testing.T) {
	cfg := config.DefaultConfig()
	player, err := New(cfg)

	if err != nil {
		t.Skip("mpv no disponible")
	}

	dur := player.Duration()
	assert.Equal(t, 0.0, dur)
}

// TestSetVolume_NotRunning prueba setear volumen sin reproducir
func TestSetVolume_NotRunning(t *testing.T) {
	cfg := config.DefaultConfig()
	player, err := New(cfg)

	if err != nil {
		t.Skip("mpv no disponible")
	}

	err = player.SetVolume(50)
	assert.Error(t, err, "Debería fallar si no está ejecutándose")
}

// TestSetVolume_InvalidRange prueba volumen fuera de rango
func TestSetVolume_InvalidRange(t *testing.T) {
	cfg := config.DefaultConfig()
	player, err := New(cfg)

	if err != nil {
		t.Skip("mpv no disponible")
	}

	// Simular que está running para probar validación
	player.running = true
	defer func() { player.running = false }()

	err = player.SetVolume(-10)
	assert.Error(t, err)

	err = player.SetVolume(150)
	assert.Error(t, err)

	err = player.SetVolume(50)
	assert.NoError(t, err)
}

// TestSeek_NotRunning prueba seek sin reproducir
func TestSeek_NotRunning(t *testing.T) {
	cfg := config.DefaultConfig()
	player, err := New(cfg)

	if err != nil {
		t.Skip("mpv no disponible")
	}

	err = player.Seek(120)
	assert.Error(t, err)
}

// TestPause_NotRunning prueba pausar sin reproducir
func TestPause_NotRunning(t *testing.T) {
	cfg := config.DefaultConfig()
	player, err := New(cfg)

	if err != nil {
		t.Skip("mpv no disponible")
	}

	err = player.Pause()
	assert.Error(t, err)
}

// TestStop_NotRunning prueba detener sin estar ejecutando
func TestStop_NotRunning(t *testing.T) {
	cfg := config.DefaultConfig()
	player, err := New(cfg)

	if err != nil {
		t.Skip("mpv no disponible")
	}

	err = player.Stop()
	assert.NoError(t, err, "Stop debería ser seguro aunque no esté running")
}

// TestPlay_AlreadyRunning prueba reproducir cuando ya está ejecutándose
func TestPlay_AlreadyRunning(t *testing.T) {
	cfg := config.DefaultConfig()
	player, err := New(cfg)

	if err != nil {
		t.Skip("mpv no disponible")
	}

	// Simular que ya está running
	player.running = true
	defer func() { player.running = false }()

	reader := strings.NewReader("test data")
	err = player.Play(reader)
	assert.Error(t, err, "No debería permitir Play si ya está running")
}

// TestEvents prueba obtener canal de eventos
func TestEvents(t *testing.T) {
	cfg := config.DefaultConfig()
	player, err := New(cfg)

	if err != nil {
		t.Skip("mpv no disponible")
	}

	events := player.Events()
	assert.NotNil(t, events)
}

// TestClose prueba cerrar el player
func TestClose(t *testing.T) {
	cfg := config.DefaultConfig()
	player, err := New(cfg)

	if err != nil {
		t.Skip("mpv no disponible")
	}

	err = player.Close()
	assert.NoError(t, err)
}

// Ejemplo funcional
func TestExample(t *testing.T) {
	cfg := config.DefaultConfig()
	assert.NotEmpty(t, cfg.Player.MPVPath)
}
