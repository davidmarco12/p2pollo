// Package streaming proporciona un servicio de streaming que combina
// el cliente torrent, el stream manager y mpv para reproducción de video.
package streaming

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/davidmarco12/p2pollo/internal/config"
	"github.com/davidmarco12/p2pollo/internal/player"
	"github.com/davidmarco12/p2pollo/internal/stream"
	"github.com/davidmarco12/p2pollo/internal/torrent"
	"github.com/sirupsen/logrus"
)

// Service coordina el cliente torrent, el stream manager y mpv.
type Service struct {
	cfg    *config.Config
	client *torrent.Client
	mgr    *stream.Manager
	player *player.MPV
	log    *logrus.Logger

	// Ruta al archivo temporal que se está descargando
	tmpPath string

	// Info del torrent activo
	torrentInfo torrent.TorrentInfo

	// Estado async
	preparing     bool
	streamErr     error
	prepareCancel context.CancelFunc

	mu sync.RWMutex
}

// NewService crea un nuevo servicio de streaming.
// Inicializa el cliente torrent y mpv.
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

	// Crear controlador mpv (pero no iniciarlo aún)
	mpv, err := player.New(cfg)
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("error creando controlador mpv: %w", err)
	}

	return &Service{
		cfg:    cfg,
		client: client,
		player: mpv,
		log:    log,
	}, nil
}

// StartStreamAsync inicia el streaming en segundo plano.
// Descarga el buffer inicial del torrent y luego carga el archivo en mpv.
func (s *Service) StartStreamAsync(magnetURI string, fileIndex int) {
	s.mu.Lock()
	s.preparing = true
	s.streamErr = nil
	ctx, cancel := context.WithCancel(context.Background())
	s.prepareCancel = cancel
	s.mu.Unlock()

	go func() {
		err := s.startStreamInternal(ctx, magnetURI, fileIndex)
		s.mu.Lock()
		s.preparing = false
		s.streamErr = err
		s.prepareCancel = nil
		s.mu.Unlock()
	}()
}

// startStreamInternal hace el trabajo de preparar torrent y lanzar mpv
func (s *Service) startStreamInternal(ctx context.Context, magnetURI string, fileIndex int) error {
	// Crear manager
	mgr := stream.New(s.cfg, s.client)

	// Preparar: agregar magnet, esperar metadata, seleccionar archivo
	info, selectedIndex, err := mgr.Prepare(ctx, magnetURI, fileIndex)
	if err != nil {
		return fmt.Errorf("error preparando torrent: %w", err)
	}

	// Verificar cancelación
	select {
	case <-ctx.Done():
		return fmt.Errorf("streaming cancelado")
	default:
	}

	// Guardar manager e info
	s.mu.Lock()
	s.mgr = mgr
	s.torrentInfo = info
	s.mu.Unlock()

	s.log.Infof("Torrent preparado: %s (archivo: %s)", info.Name, info.Files[selectedIndex].Path)

	// Iniciar descarga con buffer
	opts := stream.Options{
		BufferMB: int64(s.cfg.Streaming.InitialBufferSize),
		MoovMB:   10,
	}
	if err := mgr.Start(ctx, opts); err != nil {
		return fmt.Errorf("error iniciando descarga: %w", err)
	}

	// Esperar a que el stream esté listo (buffer inicial)
	s.log.Info("Descargando buffer inicial...")
	select {
	case tmpPath := <-mgr.Ready():
		s.mu.Lock()
		s.tmpPath = tmpPath
		s.mu.Unlock()
		s.log.Infof("Buffer listo, iniciando mpv con archivo: %s", tmpPath)

		// AHORA sí iniciar mpv (solo cuando el buffer esté listo)
		if err := s.player.Start(); err != nil {
			mgr.Stop()
			return fmt.Errorf("error iniciando mpv: %w", err)
		}

		// Cargar archivo inmediatamente
		if err := s.player.LoadFile(tmpPath); err != nil {
			s.player.Stop()
			mgr.Stop()
			return fmt.Errorf("error cargando archivo en mpv: %w", err)
		}

		s.log.Info("mpv abierto y reproduciendo")
		return nil

	case <-ctx.Done():
		mgr.Stop()
		return fmt.Errorf("streaming cancelado")

	case <-time.After(5 * time.Minute):
		mgr.Stop()
		return fmt.Errorf("timeout esperando buffer inicial")
	}
}

// IsPreparing indica si hay un stream en preparación
func (s *Service) IsPreparing() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.preparing
}

// StreamError retorna el error del último intento de streaming
func (s *Service) StreamError() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.streamErr != nil {
		return s.streamErr.Error()
	}
	return ""
}

// GetProgress retorna el progreso actual de la descarga
func (s *Service) GetProgress() stream.Progress {
	s.mu.RLock()
	mgr := s.mgr
	s.mu.RUnlock()

	if mgr == nil {
		return stream.Progress{}
	}
	return mgr.Progress()
}

// GetTorrentInfo retorna la metadata del torrent activo
func (s *Service) GetTorrentInfo() torrent.TorrentInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.torrentInfo
}

// GetPlayer retorna el controlador mpv para que app.go pueda exponer comandos
func (s *Service) GetPlayer() *player.MPV {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.player
}

// Stop detiene el streaming, mpv y limpia recursos
func (s *Service) Stop() error {
	s.mu.Lock()

	// Cancelar preparación en curso
	if s.prepareCancel != nil {
		s.prepareCancel()
		s.prepareCancel = nil
	}
	s.preparing = false
	s.streamErr = nil

	// Capturar referencias
	mpv := s.player
	mgr := s.mgr
	s.mgr = nil
	s.tmpPath = ""
	s.torrentInfo = torrent.TorrentInfo{}

	s.mu.Unlock()

	// Detener mpv (fuera del lock)
	if mpv != nil && mpv.IsRunning() {
		if err := mpv.Stop(); err != nil {
			s.log.Warnf("Error deteniendo mpv: %v", err)
		}
	}

	// Detener stream manager (fuera del lock)
	if mgr != nil {
		mgr.Stop()
		mgr.Cleanup()
	}

	s.log.Info("Streaming detenido")
	return nil
}

// Close cierra todo: streaming activo y cliente torrent
func (s *Service) Close() error {
	// Detener streaming activo
	if err := s.Stop(); err != nil {
		s.log.Warnf("Error deteniendo streaming: %v", err)
	}

	// Cerrar cliente torrent con timeout
	if s.client != nil {
		done := make(chan struct{})
		go func() {
			s.client.Close()
			close(done)
		}()
		select {
		case <-done:
			s.log.Info("Cliente torrent cerrado")
		case <-time.After(3 * time.Second):
			s.log.Warn("Timeout cerrando cliente torrent, forzando cierre")
		}
	}

	s.log.Info("Servicio de streaming cerrado")
	return nil
}

// --- Métodos legacy para compatibilidad (deprecar después) ---

// StreamURL retorna una URL vacía (legacy, mpv no usa HTTP)
func (s *Service) StreamURL() string {
	return ""
}

// FileExt retorna la extensión del archivo (legacy)
func (s *Service) FileExt() string {
	return ""
}

// CanSeekNatively siempre true con mpv (legacy)
func (s *Service) CanSeekNatively() bool {
	return true
}

// VideoDuration obtiene duración desde mpv
func (s *Service) VideoDuration() float64 {
	s.mu.RLock()
	mpv := s.player
	s.mu.RUnlock()

	if mpv == nil || !mpv.IsRunning() {
		return 0
	}

	duration, err := mpv.GetDuration()
	if err != nil {
		return 0
	}
	return duration
}

// GetSubtitleTracks obtiene tracks de mpv (legacy, usar GetPlayer().GetSubtitleTracks())
func (s *Service) GetSubtitleTracks() []player.TrackInfo {
	s.mu.RLock()
	mpv := s.player
	s.mu.RUnlock()

	if mpv == nil || !mpv.IsRunning() {
		return nil
	}

	tracks, err := mpv.GetSubtitleTracks()
	if err != nil {
		s.log.Warnf("Error obteniendo tracks de subtítulos: %v", err)
		return nil
	}

	return tracks
}
