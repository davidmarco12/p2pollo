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

// TestExample funcional
func TestExample(t *testing.T) {
	cfg := config.DefaultConfig()
	assert.NotEmpty(t, cfg.Player.MPVPath)
}

// BenchmarkPlayerCreation mide cuánto tarda crear un player
func BenchmarkPlayerCreation(b *testing.B) {
	cfg := config.DefaultConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		player, err := New(cfg)
		if err != nil {
			b.Skip("mpv no disponible")
		}
		player.Close()
	}
}

// BenchmarkCheckMPV mide cuánto tarda verificar mpv
func BenchmarkCheckMPV(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := CheckMPV("mpv")
		if err != nil {
			b.Skip("mpv no disponible")
		}
	}
}
