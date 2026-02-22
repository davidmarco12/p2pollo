// Package player proporciona controladores de mpv para reproducción de video
package player

// Player es la interfaz común para todos los controladores de mpv
type Player interface {
	// Lifecycle
	Start() error
	Stop() error
	IsRunning() bool

	// Playback
	LoadFile(path string) error
	Play() error
	Pause() error
	TogglePause() error
	Seek(seconds float64, absolute bool) error

	// Properties
	Command(name string, args ...interface{}) error
	GetProperty(name string) (interface{}, error)
	SetProperty(name string, value interface{}) error

	// State queries
	GetTimePos() (float64, error)
	GetDuration() (float64, error)
	IsPaused() (bool, error)
	GetVolume() (int, error)

	// Tracks
	GetTracks() ([]TrackInfo, error)
	GetSubtitleTracks() ([]TrackInfo, error)

	// Volume
	SetVolume(volume int) error
	ToggleFullscreen() error
}
