package stream

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/davidmarco12/p2pollo/internal/client"
	"github.com/davidmarco12/p2pollo/internal/config"
	"github.com/sirupsen/logrus"
)

// Manager gestiona el streaming de torrents
type Manager struct {
	client *client.Client
	config *config.Config
	log    *logrus.Logger
	mu     sync.RWMutex
}

// StreamInfo información del stream activo
type StreamInfo struct {
	Torrent       *client.Torrent
	Reader        io.ReadSeeker
	BufferPercent float64
	BytesBuffered int64
	IsReady       bool
}

// Health salud del stream
type Health struct {
	BufferPercent   float64
	DownloadRate    float64 // MB/s
	Peers           int
	PiecesCompleted int
	PiecesTotal     int
	IsHealthy       bool
}

// New crea un nuevo gestor de streaming
func New(c *client.Client, cfg *config.Config) *Manager {
	log := logrus.New()
	log.SetLevel(logrus.InfoLevel)
	if cfg.Logging.Level == "debug" {
		log.SetLevel(logrus.DebugLevel)
	}

	return &Manager{
		client: c,
		config: cfg,
		log:    log,
	}
}

// PrepareStream prepara un torrent para streaming
func (m *Manager) PrepareStream(torrent *client.Torrent, fileIndex int) (*StreamInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.log.Infof("Preparando stream para archivo %d", fileIndex)

	// Obtener información del torrent
	info := torrent.Info()
	if info.Name == "" {
		return nil, fmt.Errorf("metadata del torrent no disponible")
	}

	// Verificar índice de archivo
	if fileIndex < 0 || fileIndex >= len(info.Files) {
		return nil, fmt.Errorf("índice de archivo inválido: %d", fileIndex)
	}

	// Crear reader para el archivo
	var reader io.ReadSeeker
	var err error

	if len(info.Files) == 1 {
		// Archivo único, usar reader principal
		reader = torrent.NewReader()
	} else {
		// Múltiples archivos, usar reader específico
		reader, err = torrent.NewFileReader(fileIndex)
		if err != nil {
			return nil, fmt.Errorf("error creando reader: %w", err)
		}
	}

	// Calcular pieces para el archivo
	startPiece, endPiece := m.calculateFilePieces(info, fileIndex)

	// Priorizar pieces iniciales para buffering
	readahead := m.config.Streaming.ReadaheadPieces
	if readahead > (endPiece - startPiece + 1) {
		readahead = endPiece - startPiece + 1
	}

	err = torrent.PrioritizeSequential(startPiece, startPiece+readahead-1)
	if err != nil {
		m.log.Warnf("Error priorizando pieces: %v", err)
	}

	m.log.Infof("Priorizados pieces %d-%d para buffering inicial",
		startPiece, startPiece+readahead-1)

	streamInfo := &StreamInfo{
		Torrent:       torrent,
		Reader:        reader,
		BufferPercent: 0,
		BytesBuffered: 0,
		IsReady:       false,
	}

	return streamInfo, nil
}

// WaitForBuffer espera hasta que haya suficiente buffer
func (m *Manager) WaitForBuffer(streamInfo *StreamInfo) error {
	m.log.Info("Esperando buffer inicial...")

	initialBufferMB := int64(m.config.Streaming.InitialBufferSize * 1024 * 1024)
	timeout := time.After(2 * time.Minute)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout esperando buffer")
		case <-ticker.C:
			buffered := streamInfo.Torrent.BytesCompleted()
			progress := streamInfo.Torrent.Progress()

			streamInfo.BytesBuffered = buffered
			streamInfo.BufferPercent = progress

			if buffered >= initialBufferMB {
				streamInfo.IsReady = true
				m.log.Infof("Buffer listo: %.2f MB (%.2f%%)",
					float64(buffered)/(1024*1024), progress)
				return nil
			}

			m.log.Debugf("Buffering: %.2f MB / %.2f MB (%.2f%%)",
				float64(buffered)/(1024*1024),
				float64(initialBufferMB)/(1024*1024),
				(float64(buffered)/float64(initialBufferMB))*100)
		}
	}
}

// MonitorHealth monitorea la salud del stream
func (m *Manager) MonitorHealth(streamInfo *StreamInfo) <-chan Health {
	healthChan := make(chan Health, 10)

	go func() {
		defer close(healthChan)

		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				health := m.calculateHealth(streamInfo)
				healthChan <- health

				// Si la salud es crítica, re-priorizar
				if !health.IsHealthy {
					m.log.Warn("Salud del stream crítica, re-priorizando...")
					m.adjustPriorities(streamInfo)
				}
			}
		}
	}()

	return healthChan
}

// calculateHealth calcula la salud actual del stream
func (m *Manager) calculateHealth(streamInfo *StreamInfo) Health {
	info := streamInfo.Torrent.Info()
	stats := streamInfo.Torrent.Stats()

	buffered := streamInfo.Torrent.BytesCompleted()
	progress := streamInfo.Torrent.Progress()
	peers := streamInfo.Torrent.Peers()

	// Calcular piezas completadas
	piecesCompleted := int(float64(info.NumPieces) * (progress / 100.0))

	// Calcular velocidad de descarga (aproximada)
	downloadRate := float64(stats.BytesReadUsefulData.Int64()) / (1024 * 1024) // MB

	// Determinar si está saludable
	minBufferMB := int64(m.config.Streaming.MinBufferSize * 1024 * 1024)
	isHealthy := buffered >= minBufferMB && peers > 0

	return Health{
		BufferPercent:   progress,
		DownloadRate:    downloadRate,
		Peers:           peers,
		PiecesCompleted: piecesCompleted,
		PiecesTotal:     info.NumPieces,
		IsHealthy:       isHealthy,
	}
}

// adjustPriorities ajusta prioridades cuando la salud es crítica
func (m *Manager) adjustPriorities(streamInfo *StreamInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()

	info := streamInfo.Torrent.Info()
	readahead := m.config.Streaming.ReadaheadPieces

	// Calcular posición actual basada en progreso
	progress := streamInfo.Torrent.Progress()
	currentPiece := int(float64(info.NumPieces) * (progress / 100.0))

	// Priorizar adelante
	endPiece := currentPiece + readahead
	if endPiece >= info.NumPieces {
		endPiece = info.NumPieces - 1
	}

	err := streamInfo.Torrent.PrioritizeSequential(currentPiece, endPiece)
	if err != nil {
		m.log.Errorf("Error ajustando prioridades: %v", err)
	} else {
		m.log.Debugf("Prioridades ajustadas: pieces %d-%d", currentPiece, endPiece)
	}
}

// calculateFilePieces calcula el rango de pieces para un archivo
func (m *Manager) calculateFilePieces(info client.TorrentInfo, fileIndex int) (int, int) {
	if fileIndex < 0 || fileIndex >= len(info.Files) {
		return 0, info.NumPieces - 1
	}

	// Calcular offset del archivo
	var offset int64
	for i := 0; i < fileIndex; i++ {
		offset += info.Files[i].Length
	}

	fileLength := info.Files[fileIndex].Length

	// Calcular pieces
	startPiece := int(offset / info.PieceLength)
	endPiece := int((offset + fileLength) / info.PieceLength)

	if endPiece >= info.NumPieces {
		endPiece = info.NumPieces - 1
	}

	return startPiece, endPiece
}

// Seek maneja el seek en el stream (saltar a posición)
func (m *Manager) Seek(streamInfo *StreamInfo, position int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.log.Infof("Seek a posición: %d", position)

	// Seek en el reader
	_, err := streamInfo.Reader.Seek(position, io.SeekStart)
	if err != nil {
		return fmt.Errorf("error en seek: %w", err)
	}

	// Re-calcular prioridades basado en nueva posición
	info := streamInfo.Torrent.Info()

	// Calcular piece en la nueva posición
	newPiece := int(position / info.PieceLength)
	readahead := m.config.Streaming.ReadaheadPieces

	endPiece := newPiece + readahead
	if endPiece >= info.NumPieces {
		endPiece = info.NumPieces - 1
	}

	// Priorizar nuevo rango
	err = streamInfo.Torrent.PrioritizeSequential(newPiece, endPiece)
	if err != nil {
		m.log.Warnf("Error priorizando después de seek: %v", err)
	}

	m.log.Infof("Priorizados pieces %d-%d después de seek", newPiece, endPiece)

	return nil
}

// Stop detiene el streaming
func (m *Manager) Stop(streamInfo *StreamInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.log.Info("Deteniendo stream")

	// El reader se cerrará automáticamente
	// El torrent seguirá activo en el cliente
}

// Stats estadísticas del stream
func (m *Manager) Stats(streamInfo *StreamInfo) StreamStats {
	buffered := streamInfo.Torrent.BytesCompleted()
	progress := streamInfo.Torrent.Progress()
	peers := streamInfo.Torrent.Peers()

	return StreamStats{
		BytesBuffered: buffered,
		Progress:      progress,
		Peers:         peers,
		IsReady:       streamInfo.IsReady,
	}
}

// StreamStats estadísticas del streaming
type StreamStats struct {
	BytesBuffered int64
	Progress      float64
	Peers         int
	IsReady       bool
}
