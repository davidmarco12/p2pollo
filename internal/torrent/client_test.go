package torrent

import (
	"testing"
	"time"

	"github.com/davidmarco12/p2pollo/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNew prueba la creación de un nuevo cliente torrent
func TestNew(t *testing.T) {
	cfg := config.DefaultConfig()

	client, err := New(cfg)
	require.NoError(t, err)
	assert.NotNil(t, client)

	defer client.Close()

	// Verificar que el cliente está inicializado
	stats := client.Stats()
	assert.Equal(t, 0, stats.TotalTorrents, "Debería empezar sin torrents")
}

// TestNew_WithCustomPort prueba creación con puerto personalizado
func TestNew_WithCustomPort(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Client.Port = 6881

	client, err := New(cfg)
	require.NoError(t, err)
	assert.NotNil(t, client)

	defer client.Close()
}

// TestAddMagnet prueba agregar un torrent desde magnet link
func TestAddMagnet(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test que requiere red")
	}

	cfg := config.DefaultConfig()
	client, err := New(cfg)
	require.NoError(t, err)
	defer client.Close()

	// Magnet link de Big Buck Bunny (Creative Commons)
	magnetLink := "magnet:?xt=urn:btih:dd8255ecdc7ca55fb0bbf81323d87062db1f6d1c&dn=Big+Buck+Bunny"

	torrent, err := client.AddMagnet(magnetLink)
	require.NoError(t, err)
	assert.NotNil(t, torrent)

	// Verificar que se agregó
	stats := client.Stats()
	assert.Equal(t, 1, stats.TotalTorrents)
}

// TestAddMagnet_Invalid prueba magnet link inválido
func TestAddMagnet_Invalid(t *testing.T) {
	cfg := config.DefaultConfig()
	client, err := New(cfg)
	require.NoError(t, err)
	defer client.Close()

	// Magnet link inválido
	_, err = client.AddMagnet("invalid-magnet-link")
	assert.Error(t, err, "Debería fallar con magnet inválido")
}

// TestClientStats prueba obtener estadísticas del cliente
func TestClientStats(t *testing.T) {
	cfg := config.DefaultConfig()
	client, err := New(cfg)
	require.NoError(t, err)
	defer client.Close()

	stats := client.Stats()
	assert.NotNil(t, stats)
	assert.Equal(t, 0, stats.TotalTorrents)
	assert.GreaterOrEqual(t, stats.TotalPeers, 0)
	assert.GreaterOrEqual(t, stats.DownloadRate, 0.0)
}

// TestTorrents prueba obtener lista de torrents
func TestTorrents(t *testing.T) {
	cfg := config.DefaultConfig()
	client, err := New(cfg)
	require.NoError(t, err)
	defer client.Close()

	torrents := client.Torrents()
	assert.NotNil(t, torrents)
	assert.Equal(t, 0, len(torrents))
}

// TestClose prueba cerrar el cliente
func TestClose(t *testing.T) {
	cfg := config.DefaultConfig()
	client, err := New(cfg)
	require.NoError(t, err)

	err = client.Close()
	assert.NoError(t, err)
}

// TestTorrent_WaitForInfo prueba esperar metadata
func TestTorrent_WaitForInfo(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test que requiere red")
	}

	cfg := config.DefaultConfig()
	client, err := New(cfg)
	require.NoError(t, err)
	defer client.Close()

	magnetLink := "magnet:?xt=urn:btih:dd8255ecdc7ca55fb0bbf81323d87062db1f6d1c&dn=Big+Buck+Bunny"
	torrent, err := client.AddMagnet(magnetLink)
	require.NoError(t, err)

	// Esperar metadata (con timeout largo para red lenta)
	err = torrent.WaitForInfo(30 * time.Second)
	if err != nil {
		t.Skip("No se pudo obtener metadata (probablemente sin red o sin peers)")
	}

	// Verificar info
	info := torrent.Info()
	assert.NotEmpty(t, info.Name)
	assert.NotEmpty(t, info.InfoHash)
}

// TestTorrent_Info prueba obtener información del torrent
func TestTorrent_Info(t *testing.T) {
	// Este test solo verifica que Info() no crashee sin metadata
	cfg := config.DefaultConfig()
	cfg.Client.Port = 0 // Puerto aleatorio para evitar conflictos

	client, err := New(cfg)
	require.NoError(t, err)
	defer client.Close()

	magnetLink := "magnet:?xt=urn:btih:1234567890abcdef1234567890abcdef12345678"
	torrent, err := client.AddMagnet(magnetLink)
	require.NoError(t, err)

	// Sin esperar metadata
	info := torrent.Info()
	assert.Empty(t, info.Name, "Sin metadata, Name debería estar vacío")
}

// TestTorrent_Progress prueba calcular progreso
func TestTorrent_Progress(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Client.Port = 0 // Puerto aleatorio

	client, err := New(cfg)
	require.NoError(t, err)
	defer client.Close()

	magnetLink := "magnet:?xt=urn:btih:1234567890abcdef1234567890abcdef12345678"
	torrent, err := client.AddMagnet(magnetLink)
	require.NoError(t, err)

	progress := torrent.Progress()
	assert.GreaterOrEqual(t, progress, 0.0)
	assert.LessOrEqual(t, progress, 100.0)
}

// TestTorrent_BytesCompleted prueba bytes descargados
func TestTorrent_BytesCompleted(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Client.Port = 0 // Puerto aleatorio

	client, err := New(cfg)
	require.NoError(t, err)
	defer client.Close()

	magnetLink := "magnet:?xt=urn:btih:1234567890abcdef1234567890abcdef12345678"
	torrent, err := client.AddMagnet(magnetLink)
	require.NoError(t, err)

	bytes := torrent.BytesCompleted()
	assert.GreaterOrEqual(t, bytes, int64(0))
}

// TestTorrent_Peers prueba contar peers
func TestTorrent_Peers(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Client.Port = 0 // Puerto aleatorio

	client, err := New(cfg)
	require.NoError(t, err)
	defer client.Close()

	magnetLink := "magnet:?xt=urn:btih:1234567890abcdef1234567890abcdef12345678"
	torrent, err := client.AddMagnet(magnetLink)
	require.NoError(t, err)

	peers := torrent.Peers()
	assert.GreaterOrEqual(t, peers, 0)
}

// Ejemplo funcional
func TestExample(t *testing.T) {
	// Test simple para verificar que el paquete funciona
	cfg := config.DefaultConfig()
	assert.NotNil(t, cfg)
}

// ============ BENCHMARKS ============

// BenchmarkClientCreation mide cuánto tarda crear un cliente
func BenchmarkClientCreation(b *testing.B) {
	cfg := config.DefaultConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client, err := New(cfg)
		if err != nil {
			b.Fatal(err)
		}
		client.Close()
	}
}

// BenchmarkStats mide cuánto tarda obtener estadísticas
func BenchmarkStats(b *testing.B) {
	cfg := config.DefaultConfig()
	client, err := New(cfg)
	if err != nil {
		b.Fatal(err)
	}
	defer client.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = client.Stats()
	}
}

// BenchmarkMutexContention simula contención de mutex
// (múltiples goroutines leyendo simultáneamente)
func BenchmarkMutexContention(b *testing.B) {
	cfg := config.DefaultConfig()
	client, err := New(cfg)
	if err != nil {
		b.Fatal(err)
	}
	defer client.Close()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = client.Stats()
		}
	})
}
