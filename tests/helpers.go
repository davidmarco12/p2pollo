package tests

import (
	"os"
	"path/filepath"
	"testing"
)

// CreateTempDir crea un directorio temporal para tests
func CreateTempDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return dir
}

// CreateTempFile crea un archivo temporal con contenido
func CreateTempFile(t *testing.T, content string) string {
	t.Helper()
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")

	err := os.WriteFile(tmpFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Error creando archivo temporal: %v", err)
	}

	return tmpFile
}

// CreateTempYAML crea un archivo YAML temporal
func CreateTempYAML(t *testing.T, content string) string {
	t.Helper()
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "config.yaml")

	err := os.WriteFile(tmpFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Error creando archivo YAML: %v", err)
	}

	return tmpFile
}

// MockConfig ejemplo de configuración mock para tests
type MockConfig struct {
	Port     int
	Trackers []string
}

// NewMockConfig crea una configuración de prueba
func NewMockConfig() *MockConfig {
	return &MockConfig{
		Port: 6881,
		Trackers: []string{
			"udp://tracker.test:1337",
			"udp://tracker.test2:1337",
		},
	}
}
