// Package main proporciona el punto de entrada del servidor HTTP REST API de p2pollo.
// El servidor expone el backend Go (torrent, catálogo, streaming)
// para que el frontend Qt/QML pueda consumirlo via HTTP.
package main

import (
	"fmt"
	"net/http"
	"os"

	"p2pollo/internal/server"
)

func main() {
	srv, err := server.NewServer()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creando servidor: %v\n", err)
		os.Exit(1)
	}

	// Escuchar en localhost:9876 (puerto por defecto)
	if err := srv.Run("127.0.0.1:9876"); err != nil && err != http.ErrServerClosed {
		fmt.Fprintf(os.Stderr, "Error ejecutando servidor: %v\n", err)
		os.Exit(1)
	}
}
