package stream

import (
	"testing"
	"time"

	"github.com/davidmarco12/p2pollo/internal/client"
	"github.com/davidmarco12/p2pollo/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNew prueba la creación del gestor de streaming
func TestNew(t *testing.T) {
	cfg := config.DefaultConfig()
	c, err := client.New(cfg)
	require.NoError(t, err)
	defer c.Close()

	sm := New(c, cfg)
	assert.NotNil(t, sm)
	assert.NotNil(t, sm.client)
	assert.NotNil(t, sm.config)
}

// TestPrepareStream prueba la preparación del stream
func TestPrepareStream(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test que requiere red")
	}

	cfg := config.DefaultConfig()
	c, err := client.New(cfg)
	require.NoError(t, err)
	defer c.Close()

	sm := New(c, cfg)

	// Agregar torrent
	magnet := "magnet:?xt=urn:btih:dd8255ecdc7ca55fb0bbf81323d87062db1f6d1c"
	torrent, err := c.AddMagnet(magnet)
	require.NoError(t, err)

	// Esperar metadata
	err = torrent.WaitForInfo(30 * time.Second)
	if err != nil {
		t.Skip("No se pudo obtener metadata")
	}

	// Preparar stream
	streamInfo, err := sm.PrepareStream(torrent, 0)
	require.NoError(t, err)
	assert.NotNil(t, streamInfo)
	assert.NotNil(t, streamInfo.Reader)
	assert.NotNil(t, streamInfo.Torrent)
}

// TestPrepareStream_InvalidFileIndex prueba índice inválido
func TestPrepareStream_InvalidFileIndex(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test que requiere red")
	}

	cfg := config.DefaultConfig()
	c, err := client.New(cfg)
	require.NoError(t, err)
	defer c.Close()

	sm := New(c, cfg)

	magnet := "magnet:?xt=urn:btih:dd8255ecdc7ca55fb0bbf81323d87062db1f6d1c"
	torrent, err := c.AddMagnet(magnet)
	require.NoError(t, err)

	err = torrent.WaitForInfo(30 * time.Second)
	if err != nil {
		t.Skip("No se pudo obtener metadata")
	}

	// Índice inválido
	_, err = sm.PrepareStream(torrent, 999)
	assert.Error(t, err)
}

// TestCalculateHealth prueba el cálculo de salud
func TestCalculateHealth(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test que requiere red")
	}

	cfg := config.DefaultConfig()
	c, err := client.New(cfg)
	require.NoError(t, err)
	defer c.Close()

	sm := New(c, cfg)

	magnet := "magnet:?xt=urn:btih:dd8255ecdc7ca55fb0bbf81323d87062db1f6d1c"
	torrent, err := c.AddMagnet(magnet)
	require.NoError(t, err)

	err = torrent.WaitForInfo(30 * time.Second)
	if err != nil {
		t.Skip("No se pudo obtener metadata")
	}

	streamInfo, err := sm.PrepareStream(torrent, 0)
	require.NoError(t, err)

	// Calcular salud
	health := sm.calculateHealth(streamInfo)

	assert.GreaterOrEqual(t, health.BufferPercent, 0.0)
	assert.LessOrEqual(t, health.BufferPercent, 100.0)
	assert.GreaterOrEqual(t, health.Peers, 0)
	assert.GreaterOrEqual(t, health.PiecesCompleted, 0)
	assert.Greater(t, health.PiecesTotal, 0)
}

// TestStats prueba las estadísticas del stream
func TestStats(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test que requiere red")
	}

	cfg := config.DefaultConfig()
	c, err := client.New(cfg)
	require.NoError(t, err)
	defer c.Close()

	sm := New(c, cfg)

	magnet := "magnet:?xt=urn:btih:dd8255ecdc7ca55fb0bbf81323d87062db1f6d1c"
	torrent, err := c.AddMagnet(magnet)
	require.NoError(t, err)

	err = torrent.WaitForInfo(30 * time.Second)
	if err != nil {
		t.Skip("No se pudo obtener metadata")
	}

	streamInfo, err := sm.PrepareStream(torrent, 0)
	require.NoError(t, err)

	stats := sm.Stats(streamInfo)

	assert.GreaterOrEqual(t, stats.BytesBuffered, int64(0))
	assert.GreaterOrEqual(t, stats.Progress, 0.0)
	assert.GreaterOrEqual(t, stats.Peers, 0)
}

// TestCalculateFilePieces prueba el cálculo de pieces
func TestCalculateFilePieces(t *testing.T) {
	cfg := config.DefaultConfig()
	c, err := client.New(cfg)
	require.NoError(t, err)
	defer c.Close()

	sm := New(c, cfg)

	// Crear info mock
	info := client.TorrentInfo{
		NumPieces:   100,
		PieceLength: 1024 * 1024, // 1 MB
		Files: []client.FileInfo{
			{Length: 10 * 1024 * 1024}, // 10 MB
			{Length: 20 * 1024 * 1024}, // 20 MB
		},
	}

	// Primer archivo
	start, end := sm.calculateFilePieces(info, 0)
	assert.Equal(t, 0, start)
	assert.GreaterOrEqual(t, end, 9) // 10 MB / 1 MB por piece

	// Segundo archivo
	start, end = sm.calculateFilePieces(info, 1)
	assert.GreaterOrEqual(t, start, 10) // Después del primer archivo
}

// TestStop prueba detener el stream
func TestStop(t *testing.T) {
	cfg := config.DefaultConfig()
	c, err := client.New(cfg)
	require.NoError(t, err)
	defer c.Close()

	sm := New(c, cfg)

	magnet := "magnet:?xt=urn:btih:1234567890abcdef1234567890abcdef12345678"
	torrent, err := c.AddMagnet(magnet)
	require.NoError(t, err)

	streamInfo := &StreamInfo{
		Torrent: torrent,
		IsReady: true,
	}

	// No debería generar error
	sm.Stop(streamInfo)
}

// Ejemplo funcional
func TestExample(t *testing.T) {
	cfg := config.DefaultConfig()
	assert.NotNil(t, cfg)

	// Verificar configuración de streaming
	assert.Greater(t, cfg.Streaming.InitialBufferSize, 0)
	assert.Greater(t, cfg.Streaming.MinBufferSize, 0)
	assert.LessOrEqual(t, cfg.Streaming.MinBufferSize, cfg.Streaming.InitialBufferSize)
}
