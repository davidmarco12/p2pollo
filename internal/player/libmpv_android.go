//go:build android

// Stub de LibMPV para Android — el playback lo maneja mpv-android vía JNI,
// no libmpv embebido. Estos métodos nunca se llaman (headless mode siempre activo).
package player

import (
	"fmt"
	"p2pollo/internal/config"
)

// LibMPV stub vacío para que el código compile en Android.
// En Android, NewServiceHeadless() establece player=nil, así que
// ninguno de estos métodos se llama jamás en runtime.
type LibMPV struct{}

func NewLibMPV(cfg *config.Config) (*LibMPV, error) {
	return nil, fmt.Errorf("libmpv no disponible en Android")
}

func (l *LibMPV) SetHWND(hwnd uintptr) {}

// Player interface — stubs que nunca se ejecutan en Android

func (l *LibMPV) Start() error                                   { return fmt.Errorf("n/a") }
func (l *LibMPV) Stop() error                                    { return fmt.Errorf("n/a") }
func (l *LibMPV) IsRunning() bool                                { return false }
func (l *LibMPV) LoadFile(path string) error                     { return fmt.Errorf("n/a") }
func (l *LibMPV) Play() error                                    { return fmt.Errorf("n/a") }
func (l *LibMPV) Pause() error                                   { return fmt.Errorf("n/a") }
func (l *LibMPV) TogglePause() error                             { return fmt.Errorf("n/a") }
func (l *LibMPV) Seek(seconds float64, absolute bool) error      { return fmt.Errorf("n/a") }
func (l *LibMPV) Command(name string, args ...interface{}) error  { return fmt.Errorf("n/a") }
func (l *LibMPV) GetProperty(name string) (interface{}, error)   { return nil, fmt.Errorf("n/a") }
func (l *LibMPV) SetProperty(name string, value interface{}) error { return fmt.Errorf("n/a") }
func (l *LibMPV) GetTimePos() (float64, error)                   { return 0, fmt.Errorf("n/a") }
func (l *LibMPV) GetDuration() (float64, error)                  { return 0, fmt.Errorf("n/a") }
func (l *LibMPV) IsPaused() (bool, error)                        { return false, fmt.Errorf("n/a") }
func (l *LibMPV) GetVolume() (int, error)                        { return 0, fmt.Errorf("n/a") }
func (l *LibMPV) GetTracks() ([]TrackInfo, error)                { return nil, fmt.Errorf("n/a") }
func (l *LibMPV) GetSubtitleTracks() ([]TrackInfo, error)        { return nil, fmt.Errorf("n/a") }
func (l *LibMPV) SetVolume(volume int) error                     { return fmt.Errorf("n/a") }
func (l *LibMPV) ToggleFullscreen() error                        { return fmt.Errorf("n/a") }
