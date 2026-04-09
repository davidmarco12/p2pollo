//go:build !windows
// +build !windows

package player

import "fmt"

// GetMainWindowHWND no está soportado en sistemas no-Windows
func GetMainWindowHWND() (uintptr, error) {
	return 0, fmt.Errorf("HWND solo disponible en Windows")
}

// CreateMPVChildWindow no está soportado en sistemas no-Windows
func CreateMPVChildWindow() (uintptr, error) {
	return 0, fmt.Errorf("CreateMPVChildWindow solo disponible en Windows")
}

// DestroyMPVChildWindow no está soportado en sistemas no-Windows
func DestroyMPVChildWindow(hwnd uintptr) error {
	return nil
}
