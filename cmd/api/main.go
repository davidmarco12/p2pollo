// Package main proporciona un servidor HTTP REST API para p2pollo.
// Este servidor expone el backend Go (torrent, catálogo, streaming, mpv)
// para que el frontend Qt/QML pueda consumirlo via HTTP.
package main

import (
	"fmt"
	"net/http"
	"os"
)

func main() {
	server, err := NewServer()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creando servidor: %v\n", err)
		os.Exit(1)
	}

	// Escuchar en localhost:9876 (puerto por defecto)
	if err := server.Run("127.0.0.1:9876"); err != nil && err != http.ErrServerClosed {
		fmt.Fprintf(os.Stderr, "Error ejecutando servidor: %v\n", err)
		os.Exit(1)
	}
}
