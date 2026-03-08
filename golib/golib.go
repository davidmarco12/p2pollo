// Package golib expone el backend de p2pollo para uso con gomobile.
// Compilar con: gomobile bind -target=android/arm64 -o android/app/libs/golib.aar ./golib
package golib

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"p2pollo/internal/server"
)

var (
	mu  sync.Mutex
	srv *server.Server
)

// Start inicia el backend de p2pollo en localhost:9876.
// storageDir debe ser el path del directorio privado de la app Android (context.getFilesDir()).
// Retorna "" si arrancó correctamente, o un mensaje de error.
func Start(storageDir string) string {
	mu.Lock()
	defer mu.Unlock()

	if srv != nil {
		return "already running"
	}

	// Limitar el runtime de Go a 2 cores en el Flowbox F1 (4 cores, 2GB RAM).
	// Sin este límite, el cliente torrent satura los 4 cores → el scheduler de Android
	// no puede darle tiempo al main thread → ANR al presionar teclas durante la descarga.
	// Dejamos 2 cores libres para Android UI, mpv y MediaCodec.
	runtime.GOMAXPROCS(2)

	// Limpiar archivos temporales huérfanos de sesiones anteriores (crasheos, etc.)
	cleanTempFiles(storageDir)

	s, err := server.NewServerWithDir(storageDir)
	if err != nil {
		return fmt.Sprintf("error inicializando servidor: %v", err)
	}

	srv = s

	errCh := make(chan error, 1)
	go func() {
		if err := srv.Run("127.0.0.1:9876"); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// Esperar hasta 3 segundos a que el servidor inicie
	select {
	case err := <-errCh:
		srv = nil
		return fmt.Sprintf("error iniciando servidor: %v", err)
	case <-time.After(3 * time.Second):
		return "" // OK — servidor corriendo
	}
}

// Stop para el backend de p2pollo y libera recursos.
func Stop() {
	mu.Lock()
	defer mu.Unlock()

	if srv == nil {
		return
	}

	srv.Shutdown()
	srv = nil
}

// cleanTempFiles borra archivos p2pollo-stream-* del directorio temporal.
// Se llama al arrancar para limpiar restos de sesiones anteriores que terminaron de forma abrupta.
func cleanTempFiles(storageDir string) {
	tempDir := filepath.Join(storageDir, "cache", "temp")
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "p2pollo-stream-") {
			os.Remove(filepath.Join(tempDir, e.Name()))
		}
	}
}

// IsRunning retorna si el backend está activo.
func IsRunning() bool {
	mu.Lock()
	defer mu.Unlock()
	return srv != nil
}
