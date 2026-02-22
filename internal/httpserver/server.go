// Package httpserver proporciona un servidor HTTP local para servir
// archivos de video al tag <video> del frontend
package httpserver

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
)

// Server es un servidor HTTP local para streaming
type Server struct {
	server   *http.Server
	port     int
	filePath string
	log      *logrus.Logger
	mu       sync.RWMutex
}

// New crea un nuevo servidor HTTP
func New(log *logrus.Logger) *Server {
	return &Server{
		log: log,
	}
}

// Start inicia el servidor HTTP en un puerto disponible
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Encontrar puerto disponible
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("error creando listener: %w", err)
	}

	s.port = listener.Addr().(*net.TCPAddr).Port

	// Crear servidor HTTP
	mux := http.NewServeMux()
	mux.HandleFunc("/video", s.handleVideo)

	s.server = &http.Server{
		Handler: mux,
	}

	// Iniciar servidor en goroutine
	go func() {
		if err := s.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			s.log.Errorf("Error en servidor HTTP: %v", err)
		}
	}()

	s.log.Infof("Servidor HTTP iniciado en puerto %d", s.port)
	return nil
}

// SetFile establece el archivo a servir
func (s *Server) SetFile(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.filePath = path
}

// GetURL retorna la URL del video
func (s *Server) GetURL() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.port == 0 {
		return ""
	}
	return fmt.Sprintf("http://127.0.0.1:%d/video", s.port)
}

// handleVideo maneja las peticiones de video con soporte para Range requests
func (s *Server) handleVideo(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	path := s.filePath
	s.mu.RUnlock()

	if path == "" {
		http.Error(w, "No file set", http.StatusNotFound)
		return
	}

	// Abrir archivo
	file, err := os.Open(path)
	if err != nil {
		s.log.Errorf("Error abriendo archivo: %v", err)
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	defer file.Close()

	// Obtener info del archivo
	stat, err := file.Stat()
	if err != nil {
		http.Error(w, "Error getting file info", http.StatusInternalServerError)
		return
	}

	size := stat.Size()

	// Headers básicos
	w.Header().Set("Content-Type", "video/mp4")
	w.Header().Set("Accept-Ranges", "bytes")

	// Manejar Range request
	rangeHeader := r.Header.Get("Range")
	if rangeHeader == "" {
		// Sin range, enviar todo el archivo
		w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
		w.WriteHeader(http.StatusOK)
		io.Copy(w, file)
		return
	}

	// Parsear range header
	ranges := strings.TrimPrefix(rangeHeader, "bytes=")
	parts := strings.Split(ranges, "-")
	if len(parts) != 2 {
		http.Error(w, "Invalid range", http.StatusBadRequest)
		return
	}

	start, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		http.Error(w, "Invalid range start", http.StatusBadRequest)
		return
	}

	end := size - 1
	if parts[1] != "" {
		end, err = strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			http.Error(w, "Invalid range end", http.StatusBadRequest)
			return
		}
	}

	if start > end || start < 0 || end >= size {
		w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", size))
		http.Error(w, "Range not satisfiable", http.StatusRequestedRangeNotSatisfiable)
		return
	}

	contentLength := end - start + 1

	// Seek al inicio del range
	_, err = file.Seek(start, 0)
	if err != nil {
		http.Error(w, "Seek error", http.StatusInternalServerError)
		return
	}

	// Enviar headers de partial content
	w.Header().Set("Content-Length", strconv.FormatInt(contentLength, 10))
	w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, size))
	w.WriteHeader(http.StatusPartialContent)

	// Enviar contenido
	io.CopyN(w, file, contentLength)
}

// Stop detiene el servidor HTTP
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.server == nil {
		return nil
	}

	s.log.Info("Deteniendo servidor HTTP")
	err := s.server.Close()
	s.server = nil
	s.port = 0
	s.filePath = ""
	return err
}
