package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

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

	// Iniciar nuevo streaming (async) en modo headless (sin reproductor externo)
	// Qt Multimedia se encargará de la reproducción
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
