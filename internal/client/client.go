package client

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/davidmarco12/p2pollo/internal/config"
	"github.com/sirupsen/logrus"
)

// Client wrapper alrededor de anacrolix/torrent
type Client struct {
	tc     *torrent.Client
	config *config.Config
	log    *logrus.Logger
	mu     sync.RWMutex
}

// Stats estadísticas del cliente
type Stats struct {
	TotalPeers      int
	ActivePeers     int
	TotalTorrents   int
	DownloadRate    float64 // MB/s
	UploadRate      float64 // MB/s
	TotalDownloaded int64   // bytes
	TotalUploaded   int64   // bytes
}

// TorrentInfo información de un torrent
type TorrentInfo struct {
	Name        string
	InfoHash    string
	TotalLength int64
	NumPieces   int
	PieceLength int64
	Files       []FileInfo
}

// FileInfo información de un archivo en el torrent
type FileInfo struct {
	Path   string
	Length int64
	Index  int
}

// New crea un nuevo cliente torrent
func New(cfg *config.Config) (*Client, error) {
	return NewWithOptions(cfg, false)
}

// NewWithOptions crea un nuevo cliente torrent con opciones personalizadas
// lowMemory: activa modo ultra-conservador para máquinas con < 100MB RAM
func NewWithOptions(cfg *config.Config, lowMemory bool) (*Client, error) {
	// Configurar cliente torrent
	clientCfg := torrent.NewDefaultClientConfig()
	clientCfg.DataDir = cfg.Paths.CacheDir

	// Para streaming: no hacer upload durante la descarga
	clientCfg.NoUpload = true
	clientCfg.Seed = false

	// Limitar conexiones simultáneas para evitar "Not enough memory" en Windows
	if lowMemory {
		// Modo ultra-bajo para máquinas con < 100MB RAM
		clientCfg.HalfOpenConnsPerTorrent = 2    // Mínimo absoluto
		clientCfg.TorrentPeersLowWater = 1       // Solo 1 peer mínimo
		clientCfg.TorrentPeersHighWater = 3      // Máximo 3 peers
		clientCfg.EstablishedConnsPerTorrent = 2 // 2 conexiones máximo
	} else {
		// Modo normal - valores que funcionaban bien
		clientCfg.HalfOpenConnsPerTorrent = 16    // Estándar
		clientCfg.TorrentPeersLowWater = 5        // Mantener mínimo de peers
		clientCfg.TorrentPeersHighWater = 30      // Máximo de peers
		clientCfg.EstablishedConnsPerTorrent = 20 // Conexiones establecidas
	}

	// Limitar chunks en paralelo
	clientCfg.DisableAcceptRateLimiting = false

	// Configurar puerto
	if cfg.Client.Port > 0 {
		clientCfg.ListenPort = cfg.Client.Port
	} // Configurar límites de velocidad
	// Nota: anacrolix/torrent no tiene rate limiter built-in en versiones recientes
	// Se puede implementar a nivel de aplicación si es necesario
	// Por ahora dejamos sin límites y lo implementaremos después si es requerido

	// Crear cliente
	tc, err := torrent.NewClient(clientCfg)
	if err != nil {
		return nil, fmt.Errorf("error creando cliente torrent: %w", err)
	}

	// Configurar logger
	log := logrus.New()
	log.SetLevel(logrus.InfoLevel)
	if cfg.Logging.Level == "debug" {
		log.SetLevel(logrus.DebugLevel)
	}

	return &Client{
		tc:     tc,
		config: cfg,
		log:    log,
	}, nil
}

// AddMagnet agrega un torrent desde magnet link
func (c *Client) AddMagnet(magnetURI string) (*Torrent, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	t, err := c.tc.AddMagnet(magnetURI)
	if err != nil {
		return nil, fmt.Errorf("error agregando magnet: %w", err)
	}

	c.log.Infof("Torrent agregado desde magnet: %s", t.InfoHash().String())

	return &Torrent{
		t:      t,
		client: c,
	}, nil
}

// AddTorrentFile agrega un torrent desde archivo .torrent
func (c *Client) AddTorrentFile(filename string) (*Torrent, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Leer archivo
	mi, err := metainfo.LoadFromFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error leyendo archivo torrent: %w", err)
	}

	// Agregar torrent
	t, err := c.tc.AddTorrent(mi)
	if err != nil {
		return nil, fmt.Errorf("error agregando torrent: %w", err)
	}

	c.log.Infof("Torrent agregado desde archivo: %s", t.InfoHash().String())

	return &Torrent{
		t:      t,
		client: c,
	}, nil
}

// Stats retorna estadísticas del cliente
func (c *Client) Stats() Stats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var stats Stats

	torrents := c.tc.Torrents()
	stats.TotalTorrents = len(torrents)

	for _, t := range torrents {
		tStats := t.Stats()

		// Contar peers
		peers := t.PeerConns()
		stats.TotalPeers += len(peers)
		stats.ActivePeers += len(peers) // Asumimos que todos están activos

		// Acumular bytes
		stats.TotalDownloaded += tStats.BytesRead.Int64()
		stats.TotalUploaded += tStats.BytesWritten.Int64()

		// Calcular tasas (aproximado)
		stats.DownloadRate += float64(tStats.BytesReadUsefulData.Int64()) / (1024 * 1024) // MB
		stats.UploadRate += float64(tStats.BytesWritten.Int64()) / (1024 * 1024)          // MB
	}

	return stats
}

// Torrents retorna lista de todos los torrents
func (c *Client) Torrents() []*Torrent {
	c.mu.RLock()
	defer c.mu.RUnlock()

	torrents := c.tc.Torrents()
	result := make([]*Torrent, len(torrents))

	for i, t := range torrents {
		result[i] = &Torrent{
			t:      t,
			client: c,
		}
	}

	return result
}

// Close cierra el cliente y libera recursos
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.tc != nil {
		c.tc.Close()
		c.log.Info("Cliente torrent cerrado")
	}

	return nil
}

// Torrent wrapper alrededor de anacrolix/torrent.Torrent
type Torrent struct {
	t      *torrent.Torrent
	client *Client
	mu     sync.RWMutex
}

// WaitForInfo espera a que se descargue la metadata del torrent
func (t *Torrent) WaitForInfo(timeout time.Duration) error {
	select {
	case <-t.t.GotInfo():
		t.client.log.Infof("Metadata obtenida para: %s", t.t.Name())
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("timeout esperando metadata")
	}
}

// Info retorna información del torrent
func (t *Torrent) Info() TorrentInfo {
	t.mu.RLock()
	defer t.mu.RUnlock()

	info := t.t.Info()
	if info == nil {
		return TorrentInfo{}
	}

	var files []FileInfo
	if len(info.Files) == 0 {
		// Torrent de un solo archivo: info.Files está vacío,
		// los datos están en info.Name e info.Length directamente
		files = []FileInfo{
			{
				Path:   t.t.Name(),
				Length: t.t.Length(),
				Index:  0,
			},
		}
	} else {
		files = make([]FileInfo, len(info.Files))
		for i, f := range info.Files {
			files[i] = FileInfo{
				Path:   f.DisplayPath(info),
				Length: f.Length,
				Index:  i,
			}
		}
	}

	return TorrentInfo{
		Name:        t.t.Name(),
		InfoHash:    t.t.InfoHash().String(),
		TotalLength: t.t.Length(),
		NumPieces:   t.t.NumPieces(),
		PieceLength: info.PieceLength,
		Files:       files,
	}
}

// Download inicia la descarga de todo el torrent
func (t *Torrent) Download() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.t.DownloadAll()
	t.client.log.Infof("Descarga iniciada: %s", t.t.Name())
}

// NewReader crea un reader para el archivo principal del torrent
func (t *Torrent) NewReader() io.ReadSeeker {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.t.NewReader()
}

// NewFileReader crea un reader para un archivo específico
func (t *Torrent) NewFileReader(fileIndex int) (io.ReadSeeker, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	info := t.t.Info()
	if info == nil {
		return nil, fmt.Errorf("metadata no disponible")
	}

	if fileIndex < 0 || fileIndex >= len(info.Files) {
		return nil, fmt.Errorf("índice de archivo inválido: %d", fileIndex)
	}

	// Crear reader para el archivo específico
	// En anacrolix/torrent, creamos un reader y hacemos seek a la posición del archivo
	files := t.t.Files()
	if fileIndex >= len(files) {
		return nil, fmt.Errorf("índice de archivo fuera de rango")
	}

	return files[fileIndex].NewReader(), nil
}

// PrioritizeSequential prioriza pieces de manera secuencial para streaming
func (t *Torrent) PrioritizeSequential(startPiece, endPiece int) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	numPieces := t.t.NumPieces()

	if startPiece < 0 || endPiece >= numPieces {
		return fmt.Errorf("rango de pieces inválido: %d-%d (total: %d)",
			startPiece, endPiece, numPieces)
	}

	// Priorizar rango de pieces
	for i := startPiece; i <= endPiece; i++ {
		t.t.Piece(i).SetPriority(torrent.PiecePriorityNow)
	}

	t.client.log.Debugf("Priorizados pieces %d-%d", startPiece, endPiece)

	return nil
}

// BytesCompleted retorna bytes descargados
func (t *Torrent) BytesCompleted() int64 {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.t.BytesCompleted()
}

// Progress retorna progreso de descarga (0-100)
func (t *Torrent) Progress() float64 {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.t.Length() == 0 {
		return 0
	}

	return (float64(t.t.BytesCompleted()) / float64(t.t.Length())) * 100
}

// Stats retorna estadísticas del torrent
func (t *Torrent) Stats() torrent.TorrentStats {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.t.Stats()
}

// Peers retorna número de peers conectados
func (t *Torrent) Peers() int {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return len(t.t.PeerConns())
}

// Drop elimina el torrent del cliente
func (t *Torrent) Drop() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.t.Drop()
	t.client.log.Infof("Torrent eliminado: %s", t.t.InfoHash().String())
}
