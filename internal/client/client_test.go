package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewClient prueba la creación de un nuevo cliente torrent
func TestNewClient(t *testing.T) {
	t.Skip("Pendiente de implementación")

	// TODO: Implementar cuando tengamos la función NewClient
	// cfg := &Config{Port: 0}
	// client, err := NewClient(cfg)
	// require.NoError(t, err)
	// assert.NotNil(t, client)
	// defer client.Close()
}

// TestAddMagnet prueba agregar un torrent desde magnet link
func TestAddMagnet(t *testing.T) {
	t.Skip("Pendiente de implementación")

	// TODO: Implementar cuando tengamos AddMagnet
	// client, _ := NewClient(&Config{})
	// defer client.Close()
	//
	// magnetLink := "magnet:?xt=urn:btih:test"
	// torrent, err := client.AddMagnet(magnetLink)
	// require.NoError(t, err)
	// assert.NotNil(t, torrent)
}

// TestClientStats prueba obtener estadísticas del cliente
func TestClientStats(t *testing.T) {
	t.Skip("Pendiente de implementación")

	// TODO: Implementar cuando tengamos Stats
	// client, _ := NewClient(&Config{})
	// defer client.Close()
	//
	// stats := client.Stats()
	// assert.NotNil(t, stats)
	// assert.GreaterOrEqual(t, stats.TotalPeers, 0)
}

// TestPrioritizePieces prueba la priorización de pieces
func TestPrioritizePieces(t *testing.T) {
	t.Skip("Pendiente de implementación")

	// TODO: Implementar cuando tengamos PrioritizePieces
	// tests := []struct {
	// 	name       string
	// 	startPiece int
	// 	endPiece   int
	// 	wantErr    bool
	// }{
	// 	{"rango válido", 0, 10, false},
	// 	{"rango inválido", -1, 5, true},
	// }
	//
	// for _, tt := range tests {
	// 	t.Run(tt.name, func(t *testing.T) {
	// 		// Test implementation
	// 	})
	// }
}

// Ejemplo funcional
func TestExample(t *testing.T) {
	// Este test funciona para verificar que el framework está bien configurado
	result := 2 + 2
	assert.Equal(t, 4, result, "2 + 2 debería ser 4")
}
