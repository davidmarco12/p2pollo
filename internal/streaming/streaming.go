// Package streaming proporciona un servicio de streaming HTTP que combina
// el cliente torrent y el stream manager para servir video a un elemento
// <video> HTML5 en el frontend de Wails.
package streaming

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/davidmarco12/p2pollo/internal/config"
	"github.com/davidmarco12/p2pollo/internal/stream"
	"github.com/davidmarco12/p2pollo/internal/torrent"
	"github.com/sirupsen/logrus"
)

// Service coordina el cliente torrent, el stream manager y un servidor HTTP
// para servir el contenido descargado al frontend.
type Service struct {
	cfg    *config.Config
	client *torrent.Client
	mgr    *stream.Manager
	server *http.Server
	log    *logrus.Logger

	// URL local donde se sirve el stream
	streamURL string

	// Ruta al archivo temporal que se está sirviendo
	tmpPath string

	// Info del torrent activo
	torrentInfo torrent.TorrentInfo

	mu sync.RWMutex
}

// NewService crea un nuevo servicio de streaming.
// Inicializa el cliente torrent con la configuración dada.
func NewService(cfg *config.Config) (*Service, error) {
	client, err := torrent.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("error creando cliente torrent: %w", err)
	}

	log := logrus.New()
	log.SetLevel(logrus.InfoLevel)
	if cfg.Logging.Level == "debug" {
		log.SetLevel(logrus.DebugLevel)
	}

	return &Service{
		cfg:    cfg,
		client: client,
		log:    log,
	}, nil
}

// StartStream prepara el torrent, inicia la descarga y levanta el servidor HTTP.
// fileIndex indica qué archivo del torrent reproducir (-1 para auto-detectar).
func (s *Service) StartStream(magnetURI string, fileIndex int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Crear un nuevo stream manager para esta sesión
	mgr := stream.New(s.cfg, s.client)

	// Preparar: agregar magnet, esperar metadata, seleccionar archivo
	ctx := context.Background()
	info, selectedIndex, err := mgr.Prepare(ctx, magnetURI, fileIndex)
	if err != nil {
		return fmt.Errorf("error preparando torrent: %w", err)
	}

	s.mgr = mgr
	s.torrentInfo = info
	s.log.Infof("Torrent preparado: %s (archivo: %s)", info.Name, info.Files[selectedIndex].Path)

	// Iniciar descarga con opciones de buffer desde la config
	opts := stream.Options{
		BufferMB: int64(s.cfg.Streaming.InitialBufferSize),
	}
	if err := mgr.Start(ctx, opts); err != nil {
		return fmt.Errorf("error iniciando descarga: %w", err)
	}

	// Esperar a que el stream esté listo (moov + buffer mínimo)
	s.log.Info("Esperando buffer inicial...")
	select {
	case tmpPath := <-mgr.Ready():
		s.tmpPath = tmpPath
		s.log.Infof("Stream listo, archivo temporal: %s", tmpPath)
	case <-time.After(5 * time.Minute):
		mgr.Stop()
		return fmt.Errorf("timeout esperando buffer inicial")
	}

	// Levantar servidor HTTP en un puerto aleatorio
	if err := s.startHTTPServer(); err != nil {
		mgr.Stop()
		return fmt.Errorf("error iniciando servidor HTTP: %w", err)
	}

	s.log.Infof("Servidor HTTP iniciado en %s", s.streamURL)
	return nil
}

// StreamURL retorna la URL local donde se sirve el video.
// Ejemplo: "http://localhost:12345/stream"
func (s *Service) StreamURL() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.streamURL
}

// GetProgress retorna el progreso actual de la descarga.
func (s *Service) GetProgress() stream.Progress {
	s.mu.RLock()
	mgr := s.mgr
	s.mu.RUnlock()

	if mgr == nil {
		return stream.Progress{}
	}
	return mgr.Progress()
}

// GetTorrentInfo retorna la metadata del torrent activo.
func (s *Service) GetTorrentInfo() torrent.TorrentInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.torrentInfo
}

// Stop detiene el streaming y el servidor HTTP, pero no cierra el cliente torrent.
func (s *Service) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Detener servidor HTTP
	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.server.Shutdown(ctx); err != nil {
			s.log.Warnf("Error deteniendo servidor HTTP: %v", err)
		}
		s.server = nil
		s.streamURL = ""
	}

	// Detener stream manager
	if s.mgr != nil {
		s.mgr.Stop()
		s.mgr.Cleanup()
		s.mgr = nil
	}

	s.tmpPath = ""
	s.torrentInfo = torrent.TorrentInfo{}
	s.log.Info("Streaming detenido")
	return nil
}

// Close cierra todo: streaming activo y cliente torrent.
func (s *Service) Close() error {
	// Detener streaming activo si hay uno
	if err := s.Stop(); err != nil {
		s.log.Warnf("Error deteniendo streaming: %v", err)
	}

	// Cerrar cliente torrent
	if s.client != nil {
		if err := s.client.Close(); err != nil {
			return fmt.Errorf("error cerrando cliente torrent: %w", err)
		}
	}

	s.log.Info("Servicio de streaming cerrado")
	return nil
}

// startHTTPServer levanta un servidor HTTP en un puerto aleatorio (":0")
// y configura el handler para servir el archivo de video.
func (s *Service) startHTTPServer() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/stream", s.handleStream)

	// Escuchar en puerto aleatorio
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("error escuchando en puerto aleatorio: %w", err)
	}

	// Obtener el puerto asignado
	addr := listener.Addr().(*net.TCPAddr)
	s.streamURL = fmt.Sprintf("http://127.0.0.1:%d/stream", addr.Port)

	s.server = &http.Server{
		Handler: mux,
	}

	// Lanzar servidor en goroutine
	go func() {
		if err := s.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			s.log.Errorf("Error en servidor HTTP: %v", err)
		}
	}()

	return nil
}

// handleStream sirve el archivo temporal con soporte para Range requests.
// Usa http.ServeFile que maneja automáticamente Range, Content-Type, etc.
func (s *Service) handleStream(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	tmpPath := s.tmpPath
	s.mu.RUnlock()

	if tmpPath == "" {
		http.Error(w, "Stream no disponible", http.StatusServiceUnavailable)
		return
	}

	// Verificar que el archivo existe
	if _, err := os.Stat(tmpPath); os.IsNotExist(err) {
		http.Error(w, "Archivo no encontrado", http.StatusNotFound)
		return
	}

	// Permitir CORS para que el frontend de Wails pueda acceder
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Range")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	// ServeFile maneja Range requests, Content-Type por extensión, etc.
	http.ServeFile(w, r, tmpPath)
}
