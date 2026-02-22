// +build ignore

package main

import (
	"fmt"
	"os"
	"path/filepath"

	mpv "github.com/gen2brain/go-mpv"
)

func init() {
	// Agregar lib/ al PATH para que go-mpv encuentre libmpv.dll
	libDir, _ := filepath.Abs("lib")
	if _, err := os.Stat(libDir); err == nil {
		os.Setenv("PATH", libDir+";"+os.Getenv("PATH"))
	}

	// Agregar directorio actual al PATH (para desarrollo)
	if cwd, err := os.Getwd(); err == nil {
		os.Setenv("PATH", cwd+";"+os.Getenv("PATH"))
	}
}

func main() {
	fmt.Println("Probando go-mpv...")

	// Intentar crear instancia de mpv
	fmt.Println("Llamando mpv.New()...")
	m := mpv.New()
	if m == nil {
		fmt.Println("ERROR: mpv.New() retornó nil")
		os.Exit(1)
	}
	fmt.Printf("OK: mpv.New() retornó: %p\n", m)

	// Intentar inicializar
	fmt.Println("Llamando mpv.Initialize()...")
	if err := m.Initialize(); err != nil {
		fmt.Printf("ERROR: mpv.Initialize() falló: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("OK: mpv inicializado correctamente")

	// Limpiar
	m.TerminateDestroy()
	fmt.Println("OK: Test completado exitosamente")
}
