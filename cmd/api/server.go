package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"p2pollo/internal/catalog"
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
