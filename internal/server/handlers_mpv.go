package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// handleGetMPVState retorna el estado actual de reproducción de mpv
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

// handleGetMPVTracks retorna los tracks disponibles (audio/video/subtítulos)
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

	c.JSON(http.StatusOK, tracks)
}

// handleMPVCommand envía un comando a mpv
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

// handleSetMPVProperty establece una propiedad de mpv
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

// handleGetMPVProperty obtiene el valor de una propiedad de mpv
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
