package torrent

import (
	"fmt"
	"io"
	"sync"
	"time"

	libtorrent "github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/anacrolix/torrent/storage"
	"p2pollo/internal/config"
	"github.com/sirupsen/logrus"
)

// Client wrapper alrededor de anacrolix/torrent
type Client struct {
	tc     *libtorrent.Client
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

// NewForAndroid crea un cliente torrent optimizado para Android TV (Flowbox F1, 2 GB RAM).
// Usa menos conexiones que el modo normal para reducir presión de memoria y escrituras a eMMC.
func NewForAndroid(cfg *config.Config) (*Client, error) {
	clientCfg := libtorrent.NewDefaultClientConfig()

	clientCfg.DefaultStorage = NewFileStorage(
		cfg.Paths.CacheDir,
		storage.NewMapPieceCompletion(),
	)

	clientCfg.ExtendedHandshakeClientVersion = "p2pollo"
	clientCfg.Bep20 = "-p2p001-"
	clientCfg.HTTPUserAgent = "p2pollo/1.0"
	clientCfg.NoUpload = false
	clientCfg.Seed = false

	// Límites reducidos respecto al modo normal (32 half-open, 200 peers, 100 conn):
	// con 2 GB RAM y eMMC lenta, mantener 100 conexiones activas genera demasiado
	// overhead de memoria y I/O incluso cuando el stream ya está en reproducción.
	clientCfg.HalfOpenConnsPerTorrent = 8
	clientCfg.TorrentPeersLowWater = 5
	clientCfg.TorrentPeersHighWater = 30
	clientCfg.EstablishedConnsPerTorrent = 15

	clientCfg.DisableAcceptRateLimiting = false

	if cfg.Client.Port > 0 {
		clientCfg.ListenPort = cfg.Client.Port
	}

	tc, err := libtorrent.NewClient(clientCfg)
	if err != nil {
		return nil, fmt.Errorf("error creando cliente torrent (android): %w", err)
	}

	log := logrus.New()
	log.SetLevel(logrus.InfoLevel)
	if cfg.Logging.Level == "debug" {
		log.SetLevel(logrus.DebugLevel)
	}

	return &Client{tc: tc, config: cfg, log: log}, nil
}

// NewWithOptions crea un nuevo cliente torrent con opciones personalizadas
// lowMemory: activa modo ultra-conservador para máquinas con < 100MB RAM
func NewWithOptions(cfg *config.Config, lowMemory bool) (*Client, error) {
	// Configurar cliente torrent
	clientCfg := libtorrent.NewDefaultClientConfig()

	// NO setear DataDir: cuando DefaultStorage está seteado, DataDir es ignorado.
	// Si DataDir queda vacío y DefaultStorage es nil (nunca en nuestro caso),
	// el fallback usaría mmap. Dejarlo vacío para evitar cualquier ambigüedad.

	// Storage custom sin mmap. El default de anacrolix usa mmap (MapViewOfFile)
	// para datos y BoltDB para piece completion, ambos fallan en Windows
	// con archivos grandes (>1GB): "Not enough memory resources".
	// Nuestro NewFileStorage usa puro os.File.ReadAt/WriteAt.
	clientCfg.DefaultStorage = NewFileStorage(
		cfg.Paths.CacheDir,
		storage.NewMapPieceCompletion(),
	)

	// Identidad del cliente visible para otros peers (qBittorrent, trackers, etc.)
	// Por defecto anacrolix usa el path del módulo Go (p2pollo...).
	clientCfg.ExtendedHandshakeClientVersion = "p2pollo"
	clientCfg.Bep20 = "-p2p001-" // 8 chars: prefijo BEP 20 en el peer ID
	clientCfg.HTTPUserAgent = "p2pollo/1.0"

	// Habilitar upload durante la descarga: el protocolo BitTorrent premia a quienes
	// suben (tit-for-tat). Con NoUpload=true los peers nos "chokean" y la velocidad
	// de descarga cae drásticamente. Seed=false evita que sigamos subiendo después
	// de que el stream termina y se llama Stop().
	clientCfg.NoUpload = false
	clientCfg.Seed = false

	// Conexiones por torrent según modo de memoria
	if lowMemory {
		// Modo ultra-bajo para máquinas con < 100MB RAM
		clientCfg.HalfOpenConnsPerTorrent = 2
		clientCfg.TorrentPeersLowWater = 1
		clientCfg.TorrentPeersHighWater = 3
		clientCfg.EstablishedConnsPerTorrent = 2
	} else {
		// Modo normal: más peers = más fuentes simultáneas = más velocidad
		clientCfg.HalfOpenConnsPerTorrent = 32
		clientCfg.TorrentPeersLowWater = 20
		clientCfg.TorrentPeersHighWater = 200
		clientCfg.EstablishedConnsPerTorrent = 100
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
	tc, err := libtorrent.NewClient(clientCfg)
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
	t      *libtorrent.Torrent
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
		t.t.Piece(i).SetPriority(libtorrent.PiecePriorityNow)
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
func (t *Torrent) Stats() libtorrent.TorrentStats {
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
