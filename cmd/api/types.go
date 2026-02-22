package main

// --- Tipos del catálogo ---

type MovieCard struct {
	ID        string  `json:"id"`
	ImdbID    string  `json:"imdbId"`
	Title     string  `json:"title"`
	Year      int     `json:"year"`
	Rating    float64 `json:"rating"`
	PosterURL string  `json:"posterUrl"`
	Genres    string  `json:"genres"`
}

type MovieDetailRequest struct {
	ID        string  `json:"id"`
	ImdbID    string  `json:"imdbId"`
	Title     string  `json:"title"`
	Year      int     `json:"year"`
	Rating    float64 `json:"rating"`
	PosterURL string  `json:"posterUrl"`
	Genres    string  `json:"genres"`
}

type TorrentOption struct {
	Hash       string   `json:"hash"`
	Quality    string   `json:"quality"`
	Type       string   `json:"type"`
	Size       string   `json:"size"`
	Seeds      int      `json:"seeds"`
	Peers      int      `json:"peers"`
	MagnetLink string   `json:"magnetLink"`
	Provider   string   `json:"provider"`   // rargb, thepiratebay, yts
	FileName   string   `json:"fileName"`   // Nombre del archivo
	Subtitles  []string `json:"subtitles"`  // Idiomas de subtítulos
}

type MovieDetailResponse struct {
	MovieCard
	Description string          `json:"description"`
	Runtime     int             `json:"runtime"`
	Torrents    []TorrentOption `json:"torrents"`
}

// --- Tipos de streaming ---

type PlayRequest struct {
	MagnetLink string `json:"magnetLink"`
	FileIndex  int    `json:"fileIndex"` // -1 para auto-selección
}

type StreamPathResponse struct {
	Path  string `json:"path"`
	Ready bool   `json:"ready"`
}

type ProgressResponse struct {
	Preparing     bool    `json:"preparing"`
	Error         string  `json:"error"`
	HeadWritten   int64   `json:"headWritten"`
	TotalSize     int64   `json:"totalSize"`
	SpeedMBps     float64 `json:"speedMBps"`
	Percent       int     `json:"percent"`
	Peers         int     `json:"peers"`
	VideoDuration float64 `json:"videoDuration"`
}

// --- Tipos de mpv ---

type MPVStateResponse struct {
	TimePos  float64 `json:"timePos"`
	Duration float64 `json:"duration"`
	Paused   bool    `json:"paused"`
	Volume   int     `json:"volume"`
}

type MPVCommandRequest struct {
	Args []interface{} `json:"args"`
}

type SetPropertyRequest struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
}
