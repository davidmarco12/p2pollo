// Package streaming proporciona un servicio de streaming HTTP que combina
// el cliente torrent y el stream manager para servir video a un elemento
// <video> HTML5 en el frontend de Wails.
package streaming

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/davidmarco12/p2pollo/internal/config"
	"github.com/davidmarco12/p2pollo/internal/stream"
	"github.com/davidmarco12/p2pollo/internal/torrent"
	"github.com/sirupsen/logrus"
)

// Codecs de video que HTML5 <video> en WebView2/Chromium soporta de forma nativa.
// Si el codec del archivo no está en esta lista, se transcodea a H.264.
var webSafeVideoCodecs = map[string]bool{
	"h264": true,
	"vp8":  true,
	"vp9":  true,
	"av1":  true,
}

// SubtitleTrack representa un track de subtítulos disponible (embebido o externo).
type SubtitleTrack struct {
	Index    int    `json:"index"`
	Language string `json:"language"`
	Title    string `json:"title"`
	Type     string `json:"type"` // "embedded" o "external"
	FileName string `json:"fileName,omitempty"`
}

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

	// Info del video analizado
	videoCodec      string
	videoDuration   float64
	canSeekNatively bool

	// Subtítulos detectados
	subtitleTracks []SubtitleTrack

	// Cache de VTT extraídos por ffmpeg. Clave: "<tmpPath>:<streamIndex>".
	// Evita re-ejecutar ffmpeg en cada seek o re-selección del mismo track.
	subtitleCache map[string]string

	// Estado async
	preparing     bool
	streamErr     error
	prepareCancel context.CancelFunc

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
		cfg:           cfg,
		client:        client,
		log:           log,
		subtitleCache: make(map[string]string),
	}, nil
}

// StartStreamAsync inicia el streaming en segundo plano.
// El progreso se obtiene via GetProgress(), StreamURL(), IsPreparing() y StreamError().
func (s *Service) StartStreamAsync(magnetURI string, fileIndex int) {
	s.mu.Lock()
	s.preparing = true
	s.streamErr = nil
	s.streamURL = ""
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

// startStreamInternal hace todo el trabajo pesado con locking fino
// para que GetProgress() pueda funcionar durante la preparación.
func (s *Service) startStreamInternal(ctx context.Context, magnetURI string, fileIndex int) error {
	// Crear manager (no necesita lock)
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

	// Guardar manager e info (lock breve)
	s.mu.Lock()
	s.mgr = mgr
	s.torrentInfo = info
	s.mu.Unlock()

	s.log.Infof("Torrent preparado: %s (archivo: %s)", info.Name, info.Files[selectedIndex].Path)

	// Iniciar descarga con buffer reducido para arranque rápido
	opts := stream.Options{
		BufferMB: int64(s.cfg.Streaming.InitialBufferSize),
		MoovMB:   10,
	}
	if err := mgr.Start(ctx, opts); err != nil {
		return fmt.Errorf("error iniciando descarga: %w", err)
	}

	// Esperar a que el stream esté listo (moov + buffer mínimo)
	s.log.Info("Esperando buffer inicial...")
	select {
	case tmpPath := <-mgr.Ready():
		s.mu.Lock()
		s.tmpPath = tmpPath
		s.mu.Unlock()
		s.log.Infof("Stream listo, archivo temporal: %s", tmpPath)

		// Analizar codec y duración para determinar modo de seek
		videoCodec := s.probeVideoCodec(tmpPath)
		videoDuration := s.probeVideoDuration(tmpPath)
		ext := strings.ToLower(filepath.Ext(tmpPath))
		canSeekNatively := webSafeVideoCodecs[videoCodec] && ext == ".mp4"

		s.mu.Lock()
		s.videoCodec = videoCodec
		s.videoDuration = videoDuration
		s.canSeekNatively = canSeekNatively
		s.mu.Unlock()

		s.log.Infof("Video: codec=%s duration=%.1fs nativeSeek=%v", videoCodec, videoDuration, canSeekNatively)

		// Detectar subtítulos disponibles
		tracks := s.detectSubtitleTracks(tmpPath)
		s.mu.Lock()
		s.subtitleTracks = tracks
		s.mu.Unlock()
		s.log.Infof("Subtítulos detectados: %d tracks", len(tracks))

		// Estrategia: Pre-extraer subtítulos durante el tiempo de espera del buffer inicial.
		// El usuario ya está esperando que descargue el buffer (200MB), así que aprovechamos
		// ese tiempo para extraer subtítulos en background. Cuando el video esté listo,
		// los subtítulos también lo estarán (click instantáneo).
		if len(tracks) > 0 {
			s.log.Infof("Pre-extrayendo subtítulos durante descarga del buffer inicial...")
			go s.prewarmSubtitleCache(ctx, tmpPath, tracks)
		}
	case <-ctx.Done():
		mgr.Stop()
		return fmt.Errorf("streaming cancelado")
	case <-time.After(5 * time.Minute):
		mgr.Stop()
		return fmt.Errorf("timeout esperando buffer inicial")
	}

	// Levantar servidor HTTP (lock breve)
	s.mu.Lock()
	err = s.startHTTPServer()
	s.mu.Unlock()
	if err != nil {
		mgr.Stop()
		return fmt.Errorf("error iniciando servidor HTTP: %w", err)
	}

	s.log.Infof("Servidor HTTP iniciado en %s", s.StreamURL())
	return nil
}

// StreamURL retorna la URL local donde se sirve el video.
func (s *Service) StreamURL() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.streamURL
}

// FileExt retorna la extensión del archivo que se está sirviendo (ej: ".mkv", ".mp4").
func (s *Service) FileExt() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.tmpPath == "" {
		return ""
	}
	return strings.ToLower(filepath.Ext(s.tmpPath))
}

// IsPreparing indica si hay un stream en preparación.
func (s *Service) IsPreparing() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.preparing
}

// StreamError retorna el error del último intento de streaming (vacío si no hay error).
func (s *Service) StreamError() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.streamErr != nil {
		return s.streamErr.Error()
	}
	return ""
}

// CanSeekNatively indica si el video se sirve directo con seek via Range requests.
func (s *Service) CanSeekNatively() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.canSeekNatively
}

// VideoDuration retorna la duración del video en segundos (obtenida via ffprobe).
func (s *Service) VideoDuration() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.videoDuration
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
	// Cancelar preparación en curso
	if s.prepareCancel != nil {
		s.prepareCancel()
		s.prepareCancel = nil
	}
	s.preparing = false
	s.streamErr = nil

	// Capturar referencias y limpiar estado
	server := s.server
	mgr := s.mgr
	s.server = nil
	s.mgr = nil
	s.streamURL = ""
	s.tmpPath = ""
	s.torrentInfo = torrent.TorrentInfo{}
	s.videoCodec = ""
	s.videoDuration = 0
	s.canSeekNatively = false
	s.subtitleTracks = nil
	s.subtitleCache = make(map[string]string)
	s.mu.Unlock()

	// Detener servidor HTTP (fuera del lock)
	if server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			s.log.Warnf("Error deteniendo servidor HTTP: %v", err)
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

// Close cierra todo: streaming activo y cliente torrent.
// Usa timeout para no bloquear el cierre de la ventana.
func (s *Service) Close() error {
	// Detener streaming activo
	if err := s.Stop(); err != nil {
		s.log.Warnf("Error deteniendo streaming: %v", err)
	}

	// Cerrar cliente torrent con timeout (puede tardar mucho desconectando peers)
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

// startHTTPServer levanta un servidor HTTP en un puerto aleatorio (":0")
// y configura los handlers para video y subtítulos.
// DEBE ser llamado con s.mu.Lock() activo.
func (s *Service) startHTTPServer() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/stream", s.handleStream)
	mux.HandleFunc("/subtitles", s.handleSubtitles)

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

// handleStream sirve el archivo de video.
// - Si el archivo es MP4 con codec compatible (H.264, etc): sirve directo con seek nativo
// - Si necesita ffmpeg: soporta seek via ?t=<seconds> que pasa -ss a ffmpeg
// - Audio siempre a AAC, contenedor siempre fragmented MP4
func (s *Service) handleStream(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	tmpPath := s.tmpPath
	videoCodec := s.videoCodec
	canSeek := s.canSeekNatively
	s.mu.RUnlock()

	if tmpPath == "" {
		http.Error(w, "Stream no disponible", http.StatusServiceUnavailable)
		return
	}

	if _, err := os.Stat(tmpPath); os.IsNotExist(err) {
		http.Error(w, "Archivo no encontrado", http.StatusNotFound)
		return
	}

	// CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Range")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Si el archivo es MP4 con codec compatible, servir directo (seek nativo via Range)
	if canSeek {
		s.log.Info("Sirviendo MP4 directamente (seek nativo)")
		http.ServeFile(w, r, tmpPath)
		return
	}

	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		s.log.Warn("ffmpeg no encontrado, sirviendo archivo directamente")
		http.ServeFile(w, r, tmpPath)
		return
	}

	s.log.Infof("handleStream: codec=%s ext=%s", videoCodec, strings.ToLower(filepath.Ext(tmpPath)))

	// Decidir argumentos de video según el codec
	var videoArgs []string
	if webSafeVideoCodecs[videoCodec] {
		videoArgs = []string{"-c:v", "copy"}
		s.log.Infof("Codec %s compatible, copiando video sin transcodear", videoCodec)
	} else {
		videoArgs = []string{"-c:v", "libx264", "-preset", "ultrafast", "-crf", "23"}
		s.log.Infof("Codec %s no compatible, transcodificando a H.264", videoCodec)
	}

	w.Header().Set("Content-Type", "video/mp4")

	// Construir comando ffmpeg con seek opcional
	args := []string{}

	// Seek si se especifica ?t=<seconds>
	if seekTime := r.URL.Query().Get("t"); seekTime != "" {
		args = append(args, "-ss", seekTime)
		s.log.Infof("Seek a %s segundos", seekTime)
	}

	args = append(args, "-i", tmpPath)
	args = append(args, videoArgs...)
	args = append(args,
		"-c:a", "aac",
		"-b:a", "192k",
		"-f", "mp4",
		"-movflags", "frag_keyframe+empty_moov+default_base_moof",
		"-v", "warning",
		"pipe:1",
	)

	cmd := exec.CommandContext(r.Context(), ffmpegPath, args...)
	cmd.Stdout = w

	var stderrBuf strings.Builder
	cmd.Stderr = &stderrBuf

	if err := cmd.Run(); err != nil {
		if r.Context().Err() == nil {
			s.log.Warnf("Error en ffmpeg: %v | stderr: %s", err, stderrBuf.String())
		}
	} else if stderrBuf.Len() > 0 {
		s.log.Infof("ffmpeg stderr: %s", stderrBuf.String())
	}
}

// probeVideoCodec usa ffprobe para detectar el codec de video del archivo.
func (s *Service) probeVideoCodec(path string) string {
	ffprobePath, err := exec.LookPath("ffprobe")
	if err != nil {
		return ""
	}

	cmd := exec.Command(ffprobePath,
		"-v", "quiet",
		"-select_streams", "v:0",
		"-show_entries", "stream=codec_name",
		"-of", "csv=p=0",
		path,
	)

	output, err := cmd.Output()
	if err != nil {
		s.log.Warnf("Error en ffprobe: %v", err)
		return ""
	}
	return strings.TrimSpace(string(output))
}

// probeVideoDuration usa ffprobe para obtener la duración del video en segundos.
func (s *Service) probeVideoDuration(path string) float64 {
	ffprobePath, err := exec.LookPath("ffprobe")
	if err != nil {
		return 0
	}

	cmd := exec.Command(ffprobePath,
		"-v", "quiet",
		"-show_entries", "format=duration",
		"-of", "csv=p=0",
		path,
	)

	output, err := cmd.Output()
	if err != nil {
		s.log.Warnf("Error obteniendo duración: %v", err)
		return 0
	}

	d, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
	if err != nil {
		return 0
	}
	return d
}

// GetSubtitleTracks retorna los tracks de subtítulos detectados.
func (s *Service) GetSubtitleTracks() []SubtitleTrack {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.subtitleTracks
}

// prewarmSubtitleCache extrae todos los tracks embebidos en background de forma paralela
// para que el primer click del usuario sea instantáneo (respuesta desde cache, sin ffmpeg).
func (s *Service) prewarmSubtitleCache(ctx context.Context, tmpPath string, tracks []SubtitleTrack) {
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		s.log.Warnf("ffmpeg no encontrado - subtítulos no se pre-extraerán")
		return
	}

	// Filtrar solo embebidos
	var embeddedTracks []SubtitleTrack
	for _, track := range tracks {
		if track.Type == "embedded" {
			embeddedTracks = append(embeddedTracks, track)
		}
	}

	if len(embeddedTracks) == 0 {
		return
	}

	s.log.Infof("Pre-extrayendo %d subtítulos embebidos en paralelo (3 simultáneos)...", len(embeddedTracks))
	startTime := time.Now()

	// Paralelizar con semáforo (máximo 3 ffmpeg simultáneos para no sobrecargar CPU)
	sem := make(chan struct{}, 3)
	var wg sync.WaitGroup
	completed := atomic.Int32{}

	for _, track := range embeddedTracks {
		track := track // capturar variable de loop

		select {
		case <-ctx.Done():
			wg.Wait()
			return
		default:
		}

		cacheKey := fmt.Sprintf("%s:%d", tmpPath, track.Index)

		s.mu.RLock()
		_, cached := s.subtitleCache[cacheKey]
		s.mu.RUnlock()
		if cached {
			continue
		}

		wg.Add(1)
		go func() {
			defer wg.Done()

			sem <- struct{}{}        // adquirir slot
			defer func() { <-sem }() // liberar slot

			trackStart := time.Now()
			mapArg := fmt.Sprintf("0:s:%d", track.Index)
			cmd := exec.CommandContext(ctx, ffmpegPath, "-i", tmpPath, "-map", mapArg, "-f", "webvtt", "-v", "quiet", "pipe:1")
			output, err := cmd.Output()
			if err != nil || len(output) == 0 {
				s.log.Warnf("Error pre-extrayendo subtítulo %d (%s): %v", track.Index, track.Language, err)
				return
			}

			s.mu.Lock()
			s.subtitleCache[cacheKey] = string(output)
			s.mu.Unlock()

			count := completed.Add(1)
			s.log.Infof("✓ Pre-extraído %d/%d: %s (%d bytes) en %.2fs",
				count, len(embeddedTracks), track.Language, len(output), time.Since(trackStart).Seconds())
		}()
	}

	wg.Wait()
	s.log.Infof("✓ Precalentamiento completado en %.2fs - todos los subtítulos listos", time.Since(startTime).Seconds())
}

// detectSubtitleTracks detecta subtítulos embebidos (ffprobe) y externos (.srt del torrent).
func (s *Service) detectSubtitleTracks(videoPath string) []SubtitleTrack {
	var tracks []SubtitleTrack

	// 1. Embebidos via ffprobe
	tracks = append(tracks, s.probeEmbeddedSubtitles(videoPath)...)

	// 2. Externos del torrent (.srt, .ass, etc)
	s.mu.RLock()
	info := s.torrentInfo
	s.mu.RUnlock()

	subtitleIndices := torrent.FindSubtitleFiles(info.Files)
	for _, idx := range subtitleIndices {
		f := info.Files[idx]
		name := filepath.Base(f.Path)
		tracks = append(tracks, SubtitleTrack{
			Index:    idx,
			Language: guessLanguageFromFilename(name),
			Title:    name,
			Type:     "external",
			FileName: f.Path,
		})
	}

	return tracks
}

// probeEmbeddedSubtitles usa ffprobe para listar los tracks de subtítulos embebidos.
func (s *Service) probeEmbeddedSubtitles(path string) []SubtitleTrack {
	ffprobePath, err := exec.LookPath("ffprobe")
	if err != nil {
		return nil
	}

	cmd := exec.Command(ffprobePath,
		"-v", "quiet",
		"-select_streams", "s",
		"-show_entries", "stream=index:stream_tags=language,title",
		"-of", "json",
		path,
	)

	output, err := cmd.Output()
	if err != nil {
		s.log.Debugf("ffprobe subtitles error: %v", err)
		return nil
	}

	var result struct {
		Streams []struct {
			Index int `json:"index"`
			Tags  struct {
				Language string `json:"language"`
				Title    string `json:"title"`
			} `json:"tags"`
		} `json:"streams"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		s.log.Debugf("ffprobe JSON parse error: %v", err)
		return nil
	}

	var tracks []SubtitleTrack
	for i, stream := range result.Streams {
		lang := stream.Tags.Language
		if lang == "" {
			lang = "und"
		}
		title := stream.Tags.Title
		if title == "" {
			title = fmt.Sprintf("Track %d (%s)", i+1, lang)
		}
		tracks = append(tracks, SubtitleTrack{
			Index:    i,
			Language: lang,
			Title:    title,
			Type:     "embedded",
		})
	}

	return tracks
}

// handleSubtitles sirve un track de subtítulos como WebVTT.
// Query params: type=embedded|external, index=N, t=<seekOffset>
func (s *Service) handleSubtitles(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	q := r.URL.Query()
	trackType := q.Get("type")
	index, _ := strconv.Atoi(q.Get("index"))
	seekOffset, _ := strconv.ParseFloat(q.Get("t"), 64)

	if trackType == "external" {
		s.serveExternalSubtitle(w, r, index, seekOffset)
		return
	}

	// Default: embedded
	s.serveEmbeddedSubtitle(w, r, index, seekOffset)
}

// serveEmbeddedSubtitle extrae un track de subtítulos embebido via ffmpeg.
// seekOffset: segundos desde donde se está reproduciendo el video (para re-sincronizar).
func (s *Service) serveEmbeddedSubtitle(w http.ResponseWriter, r *http.Request, streamIndex int, seekOffset float64) {
	s.mu.RLock()
	tmpPath := s.tmpPath
	s.mu.RUnlock()

	if tmpPath == "" {
		http.NotFound(w, r)
		return
	}

	cacheKey := fmt.Sprintf("%s:%d", tmpPath, streamIndex)

	// Buscar en cache primero para evitar re-ejecutar ffmpeg en cada seek.
	s.mu.RLock()
	cached, found := s.subtitleCache[cacheKey]
	s.mu.RUnlock()

	if !found {
		// Primera vez: extraer el VTT completo desde el inicio (sin -ss).
		// Se cachea el resultado; el offset de seek se aplica en Go.
		s.log.Warnf("⚠️ Subtítulo stream %d NO en cache - extrayendo con ffmpeg (puede tardar)", streamIndex)
		startTime := time.Now()

		ffmpegPath, err := exec.LookPath("ffmpeg")
		if err != nil {
			http.NotFound(w, r)
			return
		}

		mapArg := fmt.Sprintf("0:s:%d", streamIndex)
		args := []string{"-i", tmpPath, "-map", mapArg, "-f", "webvtt", "-v", "quiet", "pipe:1"}

		cmd := exec.CommandContext(r.Context(), ffmpegPath, args...)
		output, err := cmd.Output()
		if err != nil || len(output) == 0 {
			http.NotFound(w, r)
			return
		}

		cached = string(output)
		s.mu.Lock()
		s.subtitleCache[cacheKey] = cached
		s.mu.Unlock()

		s.log.Infof("✓ Subtítulo stream %d extraído en %.2fs (%d bytes)", streamIndex, time.Since(startTime).Seconds(), len(output))
	} else {
		s.log.Debugf("✓ Subtítulo stream %d servido desde cache (%d bytes)", streamIndex, len(cached))
	}

	// Ajustar timestamps al punto de seek actual (operación en memoria, instantánea).
	vtt := cached
	if seekOffset > 0 {
		vtt = offsetVTT(cached, seekOffset)
	}

	w.Header().Set("Content-Type", "text/vtt; charset=utf-8")
	w.Write([]byte(vtt))
}

// serveExternalSubtitle lee un archivo de subtítulos del torrent y lo convierte a WebVTT.
// seekOffset: segundos desde donde se está reproduciendo el video.
func (s *Service) serveExternalSubtitle(w http.ResponseWriter, r *http.Request, fileIndex int, seekOffset float64) {
	s.mu.RLock()
	mgr := s.mgr
	info := s.torrentInfo
	s.mu.RUnlock()

	if mgr == nil {
		http.NotFound(w, r)
		return
	}

	if fileIndex < 0 || fileIndex >= len(info.Files) {
		http.NotFound(w, r)
		return
	}

	data, err := mgr.ReadFile(fileIndex)
	if err != nil {
		s.log.Warnf("Error leyendo subtítulo externo: %v", err)
		http.Error(w, "Error leyendo archivo", http.StatusInternalServerError)
		return
	}

	content := string(data)
	ext := strings.ToLower(filepath.Ext(info.Files[fileIndex].Path))
	if ext == ".srt" {
		content = srtToVTT(content)
	}

	// Ajustar timestamps si hay seek activo
	if seekOffset > 0 {
		content = offsetVTT(content, seekOffset)
	}

	w.Header().Set("Content-Type", "text/vtt; charset=utf-8")
	w.Write([]byte(content))
}

// srtToVTT convierte subtítulos SRT a formato WebVTT.
func srtToVTT(srt string) string {
	var b strings.Builder
	b.WriteString("WEBVTT\n\n")
	lines := strings.Split(srt, "\n")
	for _, line := range lines {
		if strings.Contains(line, "-->") {
			line = strings.ReplaceAll(line, ",", ".")
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}

// offsetVTT ajusta los timestamps de un archivo WebVTT restando el seekOffset.
// Descarta cues que terminan antes del punto de inicio.
func offsetVTT(vtt string, seekOffset float64) string {
	var b strings.Builder
	lines := strings.Split(vtt, "\n")
	skipCue := false

	for _, line := range lines {
		if strings.Contains(line, "-->") {
			parts := strings.SplitN(line, "-->", 2)
			if len(parts) == 2 {
				start := parseVTTTimestamp(strings.TrimSpace(parts[0])) - seekOffset
				end := parseVTTTimestamp(strings.TrimSpace(parts[1])) - seekOffset
				if end <= 0 {
					skipCue = true
					continue
				}
				skipCue = false
				if start < 0 {
					start = 0
				}
				line = formatVTTTimestamp(start) + " --> " + formatVTTTimestamp(end)
			}
		} else if line == "" {
			skipCue = false
		}

		if !skipCue {
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
	return b.String()
}

// parseVTTTimestamp parsea "HH:MM:SS.mmm" o "MM:SS.mmm" a segundos.
func parseVTTTimestamp(ts string) float64 {
	// Separar parte de horas/minutos/segundos de milisegundos
	dotIdx := strings.LastIndex(ts, ".")
	ms := 0.0
	if dotIdx >= 0 {
		msStr := ts[dotIdx+1:]
		for len(msStr) < 3 {
			msStr += "0"
		}
		msVal, _ := strconv.Atoi(msStr[:3])
		ms = float64(msVal) / 1000.0
		ts = ts[:dotIdx]
	}

	parts := strings.Split(ts, ":")
	secs := 0.0
	for _, p := range parts {
		v, _ := strconv.Atoi(p)
		secs = secs*60 + float64(v)
	}
	return secs + ms
}

// formatVTTTimestamp formatea segundos a "HH:MM:SS.mmm".
func formatVTTTimestamp(secs float64) string {
	if secs < 0 {
		secs = 0
	}
	h := int(secs) / 3600
	m := (int(secs) % 3600) / 60
	s := int(secs) % 60
	ms := int((secs-float64(int(secs)))*1000 + 0.5)
	return fmt.Sprintf("%02d:%02d:%02d.%03d", h, m, s, ms)
}

// guessLanguageFromFilename intenta detectar el idioma de un archivo de subtítulos por su nombre.
func guessLanguageFromFilename(name string) string {
	name = strings.ToLower(name)
	base := strings.TrimSuffix(name, filepath.Ext(name))
	parts := strings.FieldsFunc(base, func(r rune) bool {
		return r == '.' || r == '_' || r == '-' || r == ' '
	})

	langCodes := map[string]string{
		"en": "eng", "eng": "eng", "english": "eng",
		"es": "spa", "spa": "spa", "spanish": "spa", "espanol": "spa",
		"fr": "fre", "fre": "fre", "french": "fre", "fra": "fre",
		"de": "ger", "ger": "ger", "german": "ger", "deu": "ger",
		"it": "ita", "ita": "ita", "italian": "ita",
		"pt": "por", "por": "por", "portuguese": "por",
		"ja": "jpn", "jpn": "jpn", "japanese": "jpn",
		"ko": "kor", "kor": "kor", "korean": "kor",
		"zh": "chi", "chi": "chi", "chinese": "chi",
		"ru": "rus", "rus": "rus", "russian": "rus",
		"ar": "ara", "ara": "ara", "arabic": "ara",
	}

	for _, part := range parts {
		if code, ok := langCodes[part]; ok {
			return code
		}
	}
	return "und"
}
