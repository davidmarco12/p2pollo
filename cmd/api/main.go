// Package main proporciona un servidor HTTP REST API para p2pollo.
// Este servidor expone el backend Go (torrent, catálogo, streaming, mpv)
// para que el frontend Qt/QML pueda consumirlo via HTTP.
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/davidmarco12/p2pollo/internal/catalog"
	"github.com/davidmarco12/p2pollo/internal/catalog/yts"
	"github.com/davidmarco12/p2pollo/internal/config"
	"github.com/davidmarco12/p2pollo/internal/scraper"
	"github.com/davidmarco12/p2pollo/internal/scraper/providers"
	"github.com/davidmarco12/p2pollo/internal/streaming"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Server es el servidor HTTP API
type Server struct {
	router   *gin.Engine
	cfg      *config.Config
	streamer *streaming.Service
	scraper  *scraper.Scraper
	catalog  catalog.CatalogProvider
	log      *logrus.Logger
}

// NewServer crea un nuevo servidor HTTP API
func NewServer() (*Server, error) {
	// Cargar configuración
	cfg, err := config.Load("")
	if err != nil {
		cfg = config.DefaultConfig()
	}

	// Logger
	log := logrus.New()
	log.SetLevel(logrus.InfoLevel)
	if cfg.Logging.Level == "debug" {
		log.SetLevel(logrus.DebugLevel)
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Inicializar servicio de streaming
	streamer, err := streaming.NewService(cfg)
	if err != nil {
		return nil, fmt.Errorf("error inicializando streaming: %w", err)
	}

	// Inicializar scraper
	s := scraper.New()
	s.RegisterProvider("rargb", providers.NewRargb())
	s.RegisterProvider("thepiratebay", providers.NewThePirateBay(cfg.Trackers))

	// Inicializar catálogo YTS
	catalogProvider := yts.New(cfg.Trackers)

	// Crear router Gin
	router := gin.Default()

	// CORS para permitir acceso desde Qt/QML
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	srv := &Server{
		router:   router,
		cfg:      cfg,
		streamer: streamer,
		scraper:  s,
		catalog:  catalogProvider,
		log:      log,
	}

	srv.setupRoutes()

	return srv, nil
}

// setupRoutes configura las rutas del API
func (s *Server) setupRoutes() {
	api := s.router.Group("/api")

	// Catálogo de películas (YTS)
	api.GET("/popular", s.handleGetPopular)
	api.GET("/search", s.handleSearchMovies)
	api.POST("/movie/details", s.handleGetMovieDetails)

	// Streaming
	api.POST("/play", s.handlePlayMagnet)
	api.GET("/stream-path", s.handleGetStreamPath)
	api.GET("/stream", s.handleStreamVideo) // Sirve el video vía HTTP
	api.GET("/progress", s.handleGetProgress)
	api.POST("/stop", s.handleStopStream)

	// Control de mpv
	api.GET("/mpv/state", s.handleGetMPVState)
	api.GET("/mpv/tracks", s.handleGetMPVTracks)
	api.POST("/mpv/command", s.handleMPVCommand)
	api.POST("/mpv/property", s.handleSetMPVProperty)
	api.GET("/mpv/property/:name", s.handleGetMPVProperty)

	// Health check
	api.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
}

// --- Handlers del catálogo ---

type MovieCard struct {
	ID        string  `json:"id"`
	ImdbID    string  `json:"imdbId"`
	Title     string  `json:"title"`
	Year      int     `json:"year"`
	Rating    float64 `json:"rating"`
	PosterURL string  `json:"posterUrl"`
	Genres    string  `json:"genres"`
}

func (s *Server) handleGetPopular(c *gin.Context) {
	movies, err := s.catalog.Popular(1)
	if err != nil {
		s.log.Errorf("Error obteniendo películas populares: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

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

	c.JSON(http.StatusOK, cards)
}

func (s *Server) handleSearchMovies(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query parameter 'q' is required"})
		return
	}

	movies, err := s.catalog.Search(query, 1)
	if err != nil {
		s.log.Errorf("Error buscando películas: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

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

	c.JSON(http.StatusOK, cards)
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

func (s *Server) handleGetMovieDetails(c *gin.Context) {
	var req MovieDetailRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	movie := catalog.Movie{
		ID:        req.ID,
		ImdbID:    req.ImdbID,
		Title:     req.Title,
		Year:      req.Year,
		Rating:    req.Rating,
		PosterURL: req.PosterURL,
		Genres:    req.Genres,
	}

	// 1. Obtener descripción de YTS
	detail, err := s.catalog.Details(movie)
	if err != nil {
		s.log.Warnf("Error obteniendo detalles de YTS: %v", err)
		// Continuar con descripción vacía
		detail = &catalog.MovieDetail{
			Movie:       movie,
			Description: "",
			Runtime:     0,
		}
	}

	// 2. Buscar torrents en rargb y thepiratebay
	searchQuery := req.Title
	if req.Year > 0 {
		searchQuery = fmt.Sprintf("%s %d", req.Title, req.Year)
	}

	s.log.Infof("Buscando torrents para: %s", searchQuery)
	searchResults, err := s.scraper.Search(searchQuery)
	if err != nil {
		s.log.Errorf("Error buscando torrents: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 3. Convertir resultados a TorrentOption
	// Nota: No filtramos por extensión porque la mayoría de torrents no la incluyen en el nombre
	// rargb y ThePirateBay ya se especializan en películas/videos
	torrents := []TorrentOption{}
	for _, result := range searchResults {
		torrents = append(torrents, TorrentOption{
			Hash:       "", // rargb/thepiratebay no proveen hash directamente
			Quality:    extractQuality(result.Name),
			Type:       extractType(result.Name),
			Size:       result.Size,
			Seeds:      result.Seeds,
			Peers:      result.Leechers,
			MagnetLink: result.MagnetLink,
			Provider:   result.Source,
			FileName:   result.Name, // Usar el nombre completo del torrent
			Subtitles:  result.Subtitles, // Subtítulos si están disponibles
		})
	}

	s.log.Infof("Encontrados %d torrents de %d resultados", len(torrents), len(searchResults))

	c.JSON(http.StatusOK, MovieDetailResponse{
		MovieCard: MovieCard{
			ID:        req.ID,
			ImdbID:    req.ImdbID,
			Title:     req.Title,
			Year:      req.Year,
			Rating:    req.Rating,
			PosterURL: req.PosterURL,
			Genres:    req.Genres,
		},
		Description: detail.Description,
		Runtime:     detail.Runtime,
		Torrents:    torrents,
	})
}

// --- Handlers de streaming ---

type PlayRequest struct {
	MagnetLink string `json:"magnetLink"`
	FileIndex  int    `json:"fileIndex"` // -1 para auto-selección
}

func (s *Server) handlePlayMagnet(c *gin.Context) {
	var req PlayRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if req.MagnetLink == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "magnetLink is required"})
		return
	}

	// Detener streaming anterior
	s.streamer.Stop()

	// Iniciar nuevo streaming (async) en modo headless (sin reproductor externo)
	// Qt Multimedia se encargará de la reproducción
	s.streamer.StartStreamAsync(req.MagnetLink, req.FileIndex, true)

	c.JSON(http.StatusOK, gin.H{"status": "streaming started"})
}

type StreamPathResponse struct {
	Path  string `json:"path"`
	Ready bool   `json:"ready"`
}

func (s *Server) handleGetStreamPath(c *gin.Context) {
	path := s.streamer.GetTmpPath()
	ready := path != ""

	c.JSON(http.StatusOK, StreamPathResponse{
		Path:  path,
		Ready: ready,
	})
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

func (s *Server) handleGetProgress(c *gin.Context) {
	p := s.streamer.GetProgress()

	c.JSON(http.StatusOK, ProgressResponse{
		Preparing:     s.streamer.IsPreparing(),
		Error:         s.streamer.StreamError(),
		HeadWritten:   p.HeadWritten,
		TotalSize:     p.TotalSize,
		SpeedMBps:     p.SpeedMBps,
		Percent:       p.Percent,
		Peers:         p.Peers,
		VideoDuration: s.streamer.VideoDuration(),
	})
}

func (s *Server) handleStopStream(c *gin.Context) {
	if err := s.streamer.Stop(); err != nil {
		s.log.Errorf("Error deteniendo stream: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "stopped"})
}

// handleStreamVideo sirve el archivo de video vía HTTP con soporte para Range requests
func (s *Server) handleStreamVideo(c *gin.Context) {
	path := s.streamer.GetTmpPath()
	if path == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "No stream available"})
		return
	}

	// Servir el archivo con soporte para Range requests (streaming)
	c.File(path)
}

// --- Handlers de mpv ---

type MPVStateResponse struct {
	TimePos  float64 `json:"timePos"`
	Duration float64 `json:"duration"`
	Paused   bool    `json:"paused"`
	Volume   int     `json:"volume"`
}

func (s *Server) handleGetMPVState(c *gin.Context) {
	mpv := s.streamer.GetPlayer()
	if mpv == nil || !mpv.IsRunning() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "mpv not running"})
		return
	}

	timePos, _ := mpv.GetTimePos()
	duration, _ := mpv.GetDuration()
	paused, _ := mpv.IsPaused()
	volume, _ := mpv.GetVolume()

	c.JSON(http.StatusOK, MPVStateResponse{
		TimePos:  timePos,
		Duration: duration,
		Paused:   paused,
		Volume:   volume,
	})
}

type TrackInfo struct {
	ID       int    `json:"id"`
	Type     string `json:"type"`
	Language string `json:"language"`
	Title    string `json:"title"`
	Selected bool   `json:"selected"`
}

func (s *Server) handleGetMPVTracks(c *gin.Context) {
	mpv := s.streamer.GetPlayer()
	if mpv == nil || !mpv.IsRunning() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "mpv not running"})
		return
	}

	tracks, err := mpv.GetTracks()
	if err != nil {
		s.log.Errorf("Error obteniendo tracks: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	out := make([]TrackInfo, len(tracks))
	for i, t := range tracks {
		out[i] = TrackInfo{
			ID:       t.ID,
			Type:     t.Type,
			Language: t.Language,
			Title:    t.Title,
			Selected: t.Selected,
		}
	}

	c.JSON(http.StatusOK, out)
}

type MPVCommandRequest struct {
	Args []interface{} `json:"args"`
}

func (s *Server) handleMPVCommand(c *gin.Context) {
	var req MPVCommandRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if len(req.Args) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "args array is empty"})
		return
	}

	mpv := s.streamer.GetPlayer()
	if mpv == nil || !mpv.IsRunning() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "mpv not running"})
		return
	}

	cmd, ok := req.Args[0].(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "first arg must be command name (string)"})
		return
	}

	if err := mpv.Command(cmd, req.Args[1:]...); err != nil {
		s.log.Errorf("Error ejecutando comando mpv: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

type SetPropertyRequest struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
}

func (s *Server) handleSetMPVProperty(c *gin.Context) {
	var req SetPropertyRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	mpv := s.streamer.GetPlayer()
	if mpv == nil || !mpv.IsRunning() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "mpv not running"})
		return
	}

	if err := mpv.SetProperty(req.Name, req.Value); err != nil {
		s.log.Errorf("Error estableciendo propiedad mpv: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) handleGetMPVProperty(c *gin.Context) {
	name := c.Param("name")

	mpv := s.streamer.GetPlayer()
	if mpv == nil || !mpv.IsRunning() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "mpv not running"})
		return
	}

	value, err := mpv.GetProperty(name)
	if err != nil {
		s.log.Errorf("Error obteniendo propiedad mpv: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"value": value})
}

// Run inicia el servidor HTTP
func (s *Server) Run(addr string) error {
	s.log.Infof("Iniciando servidor HTTP API en %s", addr)

	srv := &http.Server{
		Addr:    addr,
		Handler: s.router,
	}

	// Graceful shutdown
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
		<-sigint

		s.log.Info("Recibida señal de interrupción, apagando servidor...")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			s.log.Errorf("Error en shutdown: %v", err)
		}

		// Cerrar servicio de streaming
		if s.streamer != nil {
			s.streamer.Close()
		}
	}()

	return srv.ListenAndServe()
}

// extractQuality extrae la calidad del nombre del torrent
func extractQuality(name string) string {
	name = strings.ToUpper(name)

	if strings.Contains(name, "2160P") || strings.Contains(name, "4K") || strings.Contains(name, "UHD") {
		return "2160p"
	}
	if strings.Contains(name, "1080P") {
		return "1080p"
	}
	if strings.Contains(name, "720P") {
		return "720p"
	}
	if strings.Contains(name, "480P") {
		return "480p"
	}

	return "Unknown"
}

// extractType extrae el tipo de release del nombre del torrent
func extractType(name string) string {
	name = strings.ToUpper(name)

	if strings.Contains(name, "BLURAY") || strings.Contains(name, "BLU-RAY") || strings.Contains(name, "BDRIP") {
		return "bluray"
	}
	if strings.Contains(name, "WEBRIP") || strings.Contains(name, "WEB-DL") || strings.Contains(name, "WEBDL") {
		return "web"
	}
	if strings.Contains(name, "DVDRIP") || strings.Contains(name, "DVD") {
		return "dvd"
	}
	if strings.Contains(name, "HDTV") {
		return "hdtv"
	}
	if strings.Contains(name, "CAM") || strings.Contains(name, "TS") || strings.Contains(name, "TELESYNC") {
		return "cam"
	}

	return "other"
}

func main() {
	server, err := NewServer()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creando servidor: %v\n", err)
		os.Exit(1)
	}

	// Escuchar en localhost:9876 (puerto por defecto)
	if err := server.Run("127.0.0.1:9876"); err != nil && err != http.ErrServerClosed {
		fmt.Fprintf(os.Stderr, "Error ejecutando servidor: %v\n", err)
		os.Exit(1)
	}
}
