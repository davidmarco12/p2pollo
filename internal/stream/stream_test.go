package stream

import (
	"context"
	"testing"

	"github.com/davidmarco12/p2pollo/internal/config"
	"github.com/davidmarco12/p2pollo/internal/torrent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNew prueba la creación del Manager
func TestNew(t *testing.T) {
	cfg := config.DefaultConfig()
	c, err := torrent.New(cfg)
	require.NoError(t, err)
	defer c.Close()

	m := New(cfg, c)
	assert.NotNil(t, m)
	assert.NotNil(t, m.client)
	assert.NotNil(t, m.cfg)
}

// TestPrepare prueba la preparación con magnet
func TestPrepare(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test que requiere red")
	}

	cfg := config.DefaultConfig()
	c, err := torrent.New(cfg)
	require.NoError(t, err)
	defer c.Close()

	m := New(cfg, c)

	ctx := context.Background()
	info, fileIdx, err := m.Prepare(ctx, "magnet:?xt=urn:btih:dd8255ecdc7ca55fb0bbf81323d87062db1f6d1c", -1)
	if err != nil {
		t.Skip("No se pudo obtener metadata")
	}

	assert.NotEmpty(t, info.Name)
	assert.GreaterOrEqual(t, fileIdx, 0)
	assert.Greater(t, len(info.Files), 0)
}

// TestPrepare_InvalidMagnet prueba magnet inválido
func TestPrepare_InvalidMagnet(t *testing.T) {
	cfg := config.DefaultConfig()
	c, err := torrent.New(cfg)
	require.NoError(t, err)
	defer c.Close()

	m := New(cfg, c)

	ctx := context.Background()
	_, _, err = m.Prepare(ctx, "invalid-magnet", 0)
	assert.Error(t, err)
}

// TestProgress retorna progreso inicial correcto
func TestProgress(t *testing.T) {
	cfg := config.DefaultConfig()
	c, err := torrent.New(cfg)
	require.NoError(t, err)
	defer c.Close()

	m := New(cfg, c)

	prog := m.Progress()
	assert.Equal(t, int64(0), prog.HeadWritten)
	assert.Equal(t, int64(0), prog.TotalSize)
	assert.Equal(t, 0, prog.Percent)
	assert.False(t, prog.MoovReady)
}

// TestOptions prueba los defaults de Options
func TestOptions(t *testing.T) {
	// Default
	opts := Options{}
	assert.Equal(t, int64(200*1024*1024), opts.bufferBytes())
	assert.Equal(t, int64(5*1024*1024), opts.moovBytes())

	// NoBuffer
	opts = Options{NoBuffer: true}
	assert.Equal(t, int64(5*1024*1024), opts.bufferBytes())

	// Custom
	opts = Options{BufferMB: 100, MoovMB: 10}
	assert.Equal(t, int64(100*1024*1024), opts.bufferBytes())
	assert.Equal(t, int64(10*1024*1024), opts.moovBytes())
}

// TestStop_NoStart no debería paniquear si se llama Stop sin Start
func TestStop_NoStart(t *testing.T) {
	cfg := config.DefaultConfig()
	c, err := torrent.New(cfg)
	require.NoError(t, err)
	defer c.Close()

	m := New(cfg, c)

	// No debería paniquear
	assert.NotPanics(t, func() {
		m.Stop()
	})
}

// TestExample verifica configuración de streaming
func TestExample(t *testing.T) {
	cfg := config.DefaultConfig()
	assert.NotNil(t, cfg)

	assert.Greater(t, cfg.Streaming.InitialBufferSize, 0)
	assert.Greater(t, cfg.Streaming.MinBufferSize, 0)
	assert.LessOrEqual(t, cfg.Streaming.MinBufferSize, cfg.Streaming.InitialBufferSize)
}
