//go:build windows
// +build windows

package player

import (
	"os"
	"path/filepath"
)

func init() {
	// IMPORTANTE: Este archivo debe procesarse ANTES que libmpv.go
	// para que el PATH esté configurado cuando go-mpv intente cargar libmpv.dll

	// Agregar lib/ al PATH
	libDir, _ := filepath.Abs("lib")
	if _, err := os.Stat(libDir); err == nil {
		os.Setenv("PATH", libDir+string(filepath.ListSeparator)+os.Getenv("PATH"))
	}

	// Agregar directorio actual al PATH
	if cwd, err := os.Getwd(); err == nil {
		os.Setenv("PATH", cwd+string(filepath.ListSeparator)+os.Getenv("PATH"))
	}
}
