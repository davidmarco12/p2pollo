package player

import "fmt"

// TrackInfo representa un track de audio/video/subtítulos de mpv
type TrackInfo struct {
	ID       int    `json:"id"`
	Type     string `json:"type"`     // "video", "audio", "sub"
	Language string `json:"language"` // Código de idioma (eng, spa, etc)
	Title    string `json:"title"`    // Título descriptivo
	Selected bool   `json:"selected"` // Si está actualmente seleccionado
}

// --- Comandos comunes de mpv ---

// Play resume la reproducción
func (m *MPV) Play() error {
	return m.SetProperty("pause", false)
}

// Pause pausa la reproducción
func (m *MPV) Pause() error {
	return m.SetProperty("pause", true)
}

// TogglePause alterna entre play y pausa
func (m *MPV) TogglePause() error {
	return m.Command("cycle", "pause")
}

// Seek salta a una posición específica (en segundos)
func (m *MPV) Seek(seconds float64, absolute bool) error {
	mode := "relative"
	if absolute {
		mode = "absolute"
	}
	return m.Command("seek", seconds, mode)
}

// SetVolume establece el volumen (0-100)
func (m *MPV) SetVolume(volume int) error {
	if volume < 0 {
		volume = 0
	}
	if volume > 100 {
		volume = 100
	}
	return m.SetProperty("volume", volume)
}

// ToggleFullscreen alterna pantalla completa
func (m *MPV) ToggleFullscreen() error {
	return m.Command("cycle", "fullscreen")
}

// CycleSubtitle cambia al siguiente track de subtítulos
func (m *MPV) CycleSubtitle() error {
	return m.Command("cycle", "sub")
}

// SetSubtitle selecciona un track de subtítulos específico por ID
func (m *MPV) SetSubtitle(trackID int) error {
	return m.SetProperty("sid", trackID)
}

// DisableSubtitle desactiva los subtítulos
func (m *MPV) DisableSubtitle() error {
	return m.SetProperty("sid", "no")
}

// SetSubtitleDelay ajusta el delay de los subtítulos (en segundos)
func (m *MPV) SetSubtitleDelay(seconds float64) error {
	return m.SetProperty("sub-delay", seconds)
}

// AddSubtitleDelay suma delay a los subtítulos actuales
func (m *MPV) AddSubtitleDelay(seconds float64) error {
	return m.Command("add", "sub-delay", seconds)
}

// CycleAudio cambia al siguiente track de audio
func (m *MPV) CycleAudio() error {
	return m.Command("cycle", "audio")
}

// SetAudio selecciona un track de audio específico por ID
func (m *MPV) SetAudio(trackID int) error {
	return m.SetProperty("aid", trackID)
}

// --- Propiedades de estado ---

// GetTimePos obtiene la posición actual de reproducción (en segundos)
func (m *MPV) GetTimePos() (float64, error) {
	val, err := m.GetProperty("time-pos")
	if err != nil {
		return 0, err
	}

	switch v := val.(type) {
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	default:
		return 0, fmt.Errorf("time-pos no es número: %T", val)
	}
}

// GetDuration obtiene la duración total del video (en segundos)
func (m *MPV) GetDuration() (float64, error) {
	val, err := m.GetProperty("duration")
	if err != nil {
		return 0, err
	}

	switch v := val.(type) {
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	default:
		return 0, fmt.Errorf("duration no es número: %T", val)
	}
}

// IsPaused indica si la reproducción está pausada
func (m *MPV) IsPaused() (bool, error) {
	val, err := m.GetProperty("pause")
	if err != nil {
		return false, err
	}

	paused, ok := val.(bool)
	if !ok {
		return false, fmt.Errorf("pause no es booleano: %T", val)
	}

	return paused, nil
}

// GetVolume obtiene el volumen actual (0-100)
func (m *MPV) GetVolume() (int, error) {
	val, err := m.GetProperty("volume")
	if err != nil {
		return 0, err
	}

	switch v := val.(type) {
	case float64:
		return int(v), nil
	case int:
		return v, nil
	default:
		return 0, fmt.Errorf("volume no es número: %T", val)
	}
}

// GetTracks obtiene la lista de todos los tracks (audio, video, subtítulos)
func (m *MPV) GetTracks() ([]TrackInfo, error) {
	val, err := m.GetProperty("track-list")
	if err != nil {
		return nil, err
	}

	// track-list es un array de objetos
	trackList, ok := val.([]interface{})
	if !ok {
		return nil, fmt.Errorf("track-list no es array: %T", val)
	}

	var tracks []TrackInfo
	for _, t := range trackList {
		trackMap, ok := t.(map[string]interface{})
		if !ok {
			continue
		}

		track := TrackInfo{}

		if id, ok := trackMap["id"].(float64); ok {
			track.ID = int(id)
		}

		if typ, ok := trackMap["type"].(string); ok {
			track.Type = typ
		}

		if lang, ok := trackMap["lang"].(string); ok {
			track.Language = lang
		}

		if title, ok := trackMap["title"].(string); ok {
			track.Title = title
		}

		if selected, ok := trackMap["selected"].(bool); ok {
			track.Selected = selected
		}

		tracks = append(tracks, track)
	}

	return tracks, nil
}

// GetSubtitleTracks obtiene solo los tracks de subtítulos
func (m *MPV) GetSubtitleTracks() ([]TrackInfo, error) {
	allTracks, err := m.GetTracks()
	if err != nil {
		return nil, err
	}

	var subs []TrackInfo
	for _, track := range allTracks {
		if track.Type == "sub" {
			subs = append(subs, track)
		}
	}

	return subs, nil
}

// GetAudioTracks obtiene solo los tracks de audio
func (m *MPV) GetAudioTracks() ([]TrackInfo, error) {
	allTracks, err := m.GetTracks()
	if err != nil {
		return nil, err
	}

	var audios []TrackInfo
	for _, track := range allTracks {
		if track.Type == "audio" {
			audios = append(audios, track)
		}
	}

	return audios, nil
}
