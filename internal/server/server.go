// Package server proporciona el servidor HTTP REST API de p2pollo.
// Exportado como package para uso en cmd/api (desktop) y golib (Android gomobile).
package server

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"p2pollo/internal/catalog"
	"p2pollo/internal/catalog/eztv"
	"p2pollo/internal/catalog/justwatch"
	"p2pollo/internal/catalog/tvmaze"
	"p2pollo/internal/catalog/yts"
	"p2pollo/internal/config"
	"p2pollo/internal/scraper"
	"p2pollo/internal/scraper/providers"
	"p2pollo/internal/streaming"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Server es el servidor HTTP API
type Server struct {
	router    *gin.Engine
	cfg       *config.Config
	streamer  *streaming.Service
	scraper   *scraper.Scraper
	catalog   catalog.CatalogProvider
	justwatch *justwatch.Client
	tvmaze    *tvmaze.Client
	eztv      *eztv.Client
	log       *logrus.Logger
	httpSrv   *http.Server // Para Shutdown() programático (Android)
}

// NewServer crea un servidor HTTP API para desktop (con libmpv).
func NewServer() (*Server, error) {
	cfg, err := config.Load("")
	if err != nil {
		cfg = config.DefaultConfig()
	}
	return newServerFromConfig(cfg, false)
}

// NewServerWithDir crea un servidor HTTP API en modo headless (sin libmpv).
// Para uso en Android: storageDir es el directorio privado de la app (getFilesDir()).
func NewServerWithDir(storageDir string) (*Server, error) {
	cfg := config.DefaultConfig()
	cfg.Paths.CacheDir = storageDir + "/cache"
	cfg.Paths.LogDir = storageDir + "/logs"
	cfg.Logging.FileLogging = false
	return newServerFromConfig(cfg, true)
}

func newServerFromConfig(cfg *config.Config, headless bool) (*Server, error) {
	log := logrus.New()
	log.SetLevel(logrus.InfoLevel)
	if cfg.Logging.Level == "debug" {
		log.SetLevel(logrus.DebugLevel)
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	var streamer *streaming.Service
	var err error
	if headless {
		streamer, err = streaming.NewServiceHeadless(cfg)
	} else {
		streamer, err = streaming.NewService(cfg)
	}
	if err != nil {
		return nil, err
	}

	s := scraper.New()
	s.RegisterProvider("rargb", providers.NewRargb())
	s.RegisterProvider("thepiratebay", providers.NewThePirateBay(cfg.Trackers))
	s.RegisterProvider("yts", providers.NewYTS(cfg.Trackers))

	catalogProvider := yts.New(cfg.Trackers)
	justwatchClient := justwatch.New()
	tvmazeClient := tvmaze.New()
	eztvClient := eztv.New()

	router := gin.Default()
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	srv := &Server{
		router:    router,
		cfg:       cfg,
		streamer:  streamer,
		scraper:   s,
		catalog:   catalogProvider,
		justwatch: justwatchClient,
		tvmaze:    tvmazeClient,
		eztv:      eztvClient,
		log:       log,
	}
	srv.setupRoutes()
	return srv, nil
}

// setupRoutes configura las rutas del API
func (s *Server) setupRoutes() {
	api := s.router.Group("/api")

	// Catálogo de películas
	api.GET("/popular", s.handleGetPopular)
	api.GET("/search", s.handleSearchMovies)
	api.POST("/movie/details", s.handleGetMovieDetails)

	// Catálogo de series
	api.GET("/series/popular", s.handleGetPopularSeries)
	api.GET("/series/search", s.handleSearchSeries)
	api.POST("/series/details", s.handleGetSeriesDetails)
	api.GET("/series/episode/torrents", s.handleGetEpisodeTorrents)

	// Streaming
	api.POST("/play", s.handlePlayMagnet)
	api.GET("/stream-path", s.handleGetStreamPath)
	api.GET("/stream", s.handleStreamVideo)
	api.GET("/progress", s.handleGetProgress)
	api.POST("/stop", s.handleStopStream)

	// Control de mpv (no-op en modo headless/Android)
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

// Run inicia el servidor HTTP. Bloquea hasta que el servidor para.
func (s *Server) Run(addr string) error {
	s.log.Infof("Iniciando servidor HTTP API en %s", addr)

	s.httpSrv = &http.Server{
		Addr:    addr,
		Handler: s.router,
	}

	// Graceful shutdown por señales del OS (desktop)
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
		<-sigint
		s.log.Info("Recibida señal de interrupción, apagando servidor...")
		s.Shutdown()
	}()

	return s.httpSrv.ListenAndServe()
}

// Shutdown para el servidor y limpia recursos.
// Puede llamarse desde Android (sin OS signals) o desde el graceful shutdown del desktop.
func (s *Server) Shutdown() {
	if s.httpSrv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := s.httpSrv.Shutdown(ctx); err != nil {
			s.log.Errorf("Error en shutdown HTTP: %v", err)
		}
	}
	if s.streamer != nil {
		s.streamer.Close()
	}
}
