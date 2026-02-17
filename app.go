package main

import (
	"context"

	"github.com/davidmarco12/p2pollo/internal/catalog"
	"github.com/davidmarco12/p2pollo/internal/catalog/yts"
	"github.com/davidmarco12/p2pollo/internal/config"
	"github.com/davidmarco12/p2pollo/internal/scraper"
	"github.com/davidmarco12/p2pollo/internal/scraper/providers"
	"github.com/davidmarco12/p2pollo/internal/streaming"
)

// App estructura principal — puente entre el frontend Svelte y el backend Go.
// Los métodos exportados se exponen automáticamente al frontend via Wails bindings.
type App struct {
	ctx      context.Context
	cfg      *config.Config
	streamer *streaming.Service
	scraper  *scraper.Scraper
	catalog  catalog.CatalogProvider
}

// NewApp crea una nueva instancia de la aplicación
func NewApp() *App {
	return &App{}
}

// startup se ejecuta al iniciar la aplicación.
// Inicializa la configuración, el servicio de streaming y el scraper.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Cargar configuración
	cfg, err := config.Load("")
	if err != nil {
		cfg = config.DefaultConfig()
	}
	a.cfg = cfg

	// Inicializar servicio de streaming
	streamer, err := streaming.NewService(cfg)
	if err == nil {
		a.streamer = streamer
	}

	// Inicializar scraper con proveedores
	s := scraper.New()
	s.RegisterProvider("rargb", providers.NewRargb())
	a.scraper = s

	// Inicializar catalogo YTS
	a.catalog = yts.New(cfg.Trackers)
}

// shutdown se ejecuta al cerrar la aplicación
func (a *App) shutdown(ctx context.Context) {
	if a.streamer != nil {
		a.streamer.Close()
	}
}

// --- Métodos expuestos al frontend (Wails bindings) ---

// SearchResult es la estructura que recibe el frontend
type SearchResult struct {
	Name       string `json:"name"`
	MagnetLink string `json:"magnetLink"`
	Size       string `json:"size"`
	Seeds      int    `json:"seeds"`
	Leechers   int    `json:"leechers"`
	Source     string `json:"source"`
	Health     int    `json:"health"`
}

// Search busca torrents en los proveedores registrados
func (a *App) Search(query string) ([]SearchResult, error) {
	results, err := a.scraper.Search(query)
	if err != nil {
		return nil, err
	}

	out := make([]SearchResult, len(results))
	for i, r := range results {
		out[i] = SearchResult{
			Name:       r.Name,
			MagnetLink: r.MagnetLink,
			Size:       r.Size,
			Seeds:      r.Seeds,
			Leechers:   r.Leechers,
			Source:     r.Source,
			Health:     r.HealthScore(),
		}
	}
	return out, nil
}

// PlayMagnet inicia el streaming de un magnet link de forma asíncrona.
// El progreso y la URL del stream se obtienen via GetStreamProgress().
func (a *App) PlayMagnet(magnetLink string) error {
	if a.streamer == nil {
		return nil
	}

	a.streamer.Stop()
	a.streamer.StartStreamAsync(magnetLink, -1)
	return nil
}

// StreamProgress representa el progreso de descarga para el frontend
type StreamProgress struct {
	StreamURL       string  `json:"streamURL"`
	Preparing       bool    `json:"preparing"`
	Error           string  `json:"error"`
	FileExt         string  `json:"fileExt"`
	HeadWritten     int64   `json:"headWritten"`
	TotalSize       int64   `json:"totalSize"`
	SpeedMBps       float64 `json:"speedMBps"`
	Percent         int     `json:"percent"`
	Peers           int     `json:"peers"`
	CanSeekNatively bool    `json:"canSeekNatively"`
	VideoDuration   float64 `json:"videoDuration"`
}

// GetStreamProgress retorna el progreso actual de la descarga,
// incluyendo la URL del stream cuando esté disponible.
func (a *App) GetStreamProgress() StreamProgress {
	if a.streamer == nil {
		return StreamProgress{}
	}

	p := a.streamer.GetProgress()
	return StreamProgress{
		StreamURL:       a.streamer.StreamURL(),
		Preparing:       a.streamer.IsPreparing(),
		Error:           a.streamer.StreamError(),
		FileExt:         a.streamer.FileExt(),
		HeadWritten:     p.HeadWritten,
		TotalSize:       p.TotalSize,
		SpeedMBps:       p.SpeedMBps,
		Percent:         p.Percent,
		Peers:           p.Peers,
		CanSeekNatively: a.streamer.CanSeekNatively(),
		VideoDuration:   a.streamer.VideoDuration(),
	}
}

// StopStream detiene el streaming actual
func (a *App) StopStream() error {
	if a.streamer == nil {
		return nil
	}
	return a.streamer.Stop()
}

// SubtitleTrackInfo es la estructura que recibe el frontend para los subtítulos
type SubtitleTrackInfo struct {
	Index    int    `json:"index"`
	Language string `json:"language"`
	Title    string `json:"title"`
	Type     string `json:"type"`
	FileName string `json:"fileName,omitempty"`
}

// GetSubtitleTracks retorna los tracks de subtítulos disponibles del stream activo
func (a *App) GetSubtitleTracks() []SubtitleTrackInfo {
	if a.streamer == nil {
		return nil
	}

	tracks := a.streamer.GetSubtitleTracks()
	out := make([]SubtitleTrackInfo, len(tracks))
	for i, t := range tracks {
		out[i] = SubtitleTrackInfo{
			Index:    t.Index,
			Language: t.Language,
			Title:    t.Title,
			Type:     t.Type,
			FileName: t.FileName,
		}
	}
	return out
}

// --- Métodos del catálogo de películas (YTS) ---

// MovieCard es la estructura que recibe el frontend para la grilla de peliculas
type MovieCard struct {
	ID        string  `json:"id"`
	ImdbID    string  `json:"imdbId"`
	Title     string  `json:"title"`
	Year      int     `json:"year"`
	Rating    float64 `json:"rating"`
	PosterURL string  `json:"posterUrl"`
	Genres    string  `json:"genres"`
}

// GetPopularMovies retorna peliculas populares para la pagina de inicio
func (a *App) GetPopularMovies() ([]MovieCard, error) {
	if a.catalog == nil {
		return nil, nil
	}

	movies, err := a.catalog.Popular(1)
	if err != nil {
		return nil, err
	}

	return moviesToCards(movies), nil
}

// SearchMovies busca peliculas por titulo
func (a *App) SearchMovies(query string) ([]MovieCard, error) {
	if a.catalog == nil {
		return nil, nil
	}

	movies, err := a.catalog.Search(query, 1)
	if err != nil {
		return nil, err
	}

	return moviesToCards(movies), nil
}

func moviesToCards(movies []catalog.Movie) []MovieCard {
	cards := make([]MovieCard, len(movies))
	for i, m := range movies {
		cards[i] = MovieCard{
			ID:        m.ID,
			ImdbID:    m.ImdbID,
			Title:     m.Title,
			Year:      m.Year,
			Rating:    m.Rating,
			PosterURL: m.PosterURL,
			Genres:    m.Genres,
		}
	}
	return cards
}
