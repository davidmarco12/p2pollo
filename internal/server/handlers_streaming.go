package server

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
)

// streamingReader es un io.ReadSeeker que bloquea lecturas más allá del headWritten.
// Evita que mpv reciba ceros del área pre-alocada pero aún no descargada,
// lo que causaba freeze/corrupción en el parser MKV/MP4.
// Cuando el context se cancela (cliente desconectado), retorna io.EOF.
type streamingReader struct {
	f         *os.File
	pos       int64
	totalSize int64
	getHead   func() int64
	done      <-chan struct{}
}

// maxBlockGap: si el seek está dentro de este margen del frontier, esperamos
// a que el torrent descargue. Si está más lejos (ej: mpv buscando el MKV index
// al final del archivo), retornamos EOF para que mpv use modo lineal sin índice.
const maxBlockGap = 2 * 1024 * 1024 // 2 MB

func (r *streamingReader) Read(buf []byte) (int, error) {
	for {
		select {
		case <-r.done:
			return 0, io.EOF
		default:
		}

		if r.pos >= r.totalSize {
			return 0, io.EOF
		}

		head := r.getHead()
		if r.pos < head {
			available := head - r.pos
			if int64(len(buf)) > available {
				buf = buf[:available]
			}
			n, err := r.f.ReadAt(buf, r.pos)
			r.pos += int64(n)
			return n, err
		}

		// Más allá del frontier: decidir si esperar o fallar rápido
		gap := r.pos - head
		if gap > maxBlockGap {
			// Seek lejano (ej: MKV SeekHead/Cues al final del archivo).
			// Retornar EOF inmediato para que mpv caiga a modo lineal sin índice,
			// en vez de bloquear hasta que el torrent descargue hasta ahí.
			return 0, io.EOF
		}

		// Cerca del frontier — esperar a que el torrent avance
		time.Sleep(150 * time.Millisecond)
	}
}

func (r *streamingReader) Seek(offset int64, whence int) (int64, error) {
	var newPos int64
	switch whence {
	case io.SeekStart:
		newPos = offset
	case io.SeekCurrent:
		newPos = r.pos + offset
	case io.SeekEnd:
		newPos = r.totalSize + offset
	default:
		return 0, fmt.Errorf("seek: whence inválido %d", whence)
	}
	if newPos < 0 {
		return 0, fmt.Errorf("seek: posición negativa")
	}
	r.pos = newPos
	return r.pos, nil
}

// handlePlayMagnet inicia el streaming de un magnet link
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

	// Iniciar nuevo streaming (async) en modo headless
	// El reproductor (Qt o Android mpv) se encarga del playback via /api/stream
	s.streamer.StartStreamAsync(req.MagnetLink, req.FileIndex, true)

	c.JSON(http.StatusOK, gin.H{"status": "streaming started"})
}

// handleGetStreamPath retorna el path del archivo de video temporal
func (s *Server) handleGetStreamPath(c *gin.Context) {
	path := s.streamer.GetTmpPath()
	ready := path != ""

	c.JSON(http.StatusOK, StreamPathResponse{
		Path:  path,
		Ready: ready,
	})
}

// handleGetProgress retorna el progreso de descarga del stream
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

// handleStopStream detiene el streaming actual
func (s *Server) handleStopStream(c *gin.Context) {
	if err := s.streamer.Stop(); err != nil {
		s.log.Errorf("Error deteniendo stream: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "stopped"})
}

// handleStreamVideo sirve el archivo de video vía HTTP con soporte para Range requests.
// Usa streamingReader para bloquear en el límite de descarga en vez de servir
// ceros del área pre-alocada, evitando freeze/corrupción en el parser de mpv.
func (s *Server) handleStreamVideo(c *gin.Context) {
	path := s.streamer.GetTmpPath()
	if path == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "No stream available"})
		return
	}

	f, err := os.Open(path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot open stream file"})
		return
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot stat stream file"})
		return
	}

	reader := &streamingReader{
		f:         f,
		totalSize: info.Size(),
		getHead:   func() int64 { return s.streamer.GetProgress().HeadWritten },
		done:      c.Request.Context().Done(),
	}

	http.ServeContent(c.Writer, c.Request, filepath.Base(path), info.ModTime(), reader)
}
