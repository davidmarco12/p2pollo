package stream

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/davidmarco12/p2pollo/internal/config"
	"github.com/davidmarco12/p2pollo/internal/torrent"
)

// Options configura el comportamiento del streaming.
type Options struct {
	NoBuffer  bool  // No esperar buffer completo (5MB mínimo)
	BufferMB  int64 // Buffer objetivo en MB (default 200)
	MoovMB    int64 // Tamaño del moov atom en MB (default 5)
}

func (o Options) bufferBytes() int64 {
	if o.NoBuffer {
		return 5 * 1024 * 1024
	}
	mb := o.BufferMB
	if mb <= 0 {
		mb = 200
	}
	return mb * 1024 * 1024
}

func (o Options) moovBytes() int64 {
	mb := o.MoovMB
	if mb <= 0 {
		mb = 5
	}
	return mb * 1024 * 1024
}

// Progress representa el estado actual de la descarga.
type Progress struct {
	HeadWritten int64   // Bytes escritos secuencialmente desde el inicio
	TotalSize   int64   // Tamaño total del archivo
	SpeedMBps   float64 // Velocidad en MB/s
	MoovReady   bool    // Si el moov atom ya se descargó
	Peers       int     // Peers conectados
	Percent     int     // Progreso 0-100
}

// Manager gestiona la descarga y preparación de un archivo torrent para reproducción.
// Es agnóstico a la presentación: no imprime nada, solo expone estado via Progress() y channels.
type Manager struct {
	cfg    *config.Config
	client *torrent.Client

	// Estado del torrent activo
	t         *torrent.Torrent
	info      torrent.TorrentInfo
	fileIndex int
	tmpPath   string
	tmpFile   *os.File

	// Progreso atómico (safe para lectura concurrente)
	headWritten atomic.Int64
	moovReady   atomic.Bool
	totalSize   int64

	// Sincronización
	readyCh chan string // Se cierra cuando el archivo está listo para reproducir (envía tmpPath)
	cancel  context.CancelFunc
	wg      sync.WaitGroup

	// Speed tracking
	lastBytes atomic.Int64
	lastTime  atomic.Int64 // UnixNano

	mu sync.Mutex // Protege campos mutables (tmpFile, tmpPath)
}

// New crea un nuevo Manager.
func New(cfg *config.Config, client *torrent.Client) *Manager {
	return &Manager{
		cfg:    cfg,
		client: client,
	}
}

// Prepare agrega el magnet, espera metadata y auto-detecta el archivo multimedia.
// Retorna la info del torrent y el índice del archivo seleccionado.
func (m *Manager) Prepare(ctx context.Context, magnetURI string, fileIndex int) (torrent.TorrentInfo, int, error) {
	t, err := m.client.AddMagnet(magnetURI)
	if err != nil {
		return torrent.TorrentInfo{}, 0, fmt.Errorf("agregar magnet: %w", err)
	}

	if err := t.WaitForInfo(60 * time.Second); err != nil {
		return torrent.TorrentInfo{}, 0, fmt.Errorf("obtener metadata: %w", err)
	}

	info := t.Info()

	// Auto-detectar archivo multimedia si fileIndex == -1
	if fileIndex < 0 {
		fileIndex = torrent.FindMediaFile(info.Files)
	}

	if fileIndex < 0 || fileIndex >= len(info.Files) {
		return torrent.TorrentInfo{}, 0, fmt.Errorf("índice de archivo inválido: %d (total: %d)", fileIndex, len(info.Files))
	}

	m.t = t
	m.info = info
	m.fileIndex = fileIndex
	m.totalSize = info.Files[fileIndex].Length

	return info, fileIndex, nil
}

// Start inicia la descarga, prioriza piezas, y lanza las goroutines de escritura.
// Retorna inmediatamente. Usar Ready() para esperar a que esté listo para reproducir.
func (m *Manager) Start(ctx context.Context, opts Options) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.t == nil {
		return fmt.Errorf("llamar a Prepare() antes de Start()")
	}

	ctx, m.cancel = context.WithCancel(ctx)

	// Priorizar piezas iniciales para streaming secuencial
	readahead := 100
	if readahead > m.info.NumPieces {
		readahead = m.info.NumPieces
	}
	m.t.PrioritizeSequential(0, readahead-1)

	// Priorizar últimas piezas para el moov atom
	moovSize := opts.moovBytes()
	if moovSize > m.totalSize {
		moovSize = m.totalSize
	}
	moovPieces := int(moovSize/m.info.PieceLength) + 1
	lastPiece := m.info.NumPieces - 1
	firstMoovPiece := lastPiece - moovPieces
	if firstMoovPiece < 0 {
		firstMoovPiece = 0
	}
	m.t.PrioritizeSequential(firstMoovPiece, lastPiece)

	m.t.Download()

	// Crear archivo temporal
	if err := m.createTempFile(); err != nil {
		return fmt.Errorf("crear archivo temporal: %w", err)
	}

	// Pre-allocar para poder escribir moov atom al final con WriteAt
	if err := m.tmpFile.Truncate(m.totalSize); err != nil {
		return fmt.Errorf("pre-allocar archivo: %w", err)
	}

	// Calcular buffer mínimo
	bufferBytes := opts.bufferBytes()
	if bufferBytes > m.totalSize {
		bufferBytes = m.totalSize
	}

	m.readyCh = make(chan string, 1)

	// Inicializar speed tracking
	m.lastTime.Store(time.Now().UnixNano())
	m.lastBytes.Store(0)

	// Goroutine: descargar moov atom (final del archivo)
	m.wg.Add(1)
	go m.downloadMoov(ctx, moovSize)

	// Goroutine: copia secuencial desde el inicio
	m.wg.Add(1)
	go m.downloadSequential(ctx)

	// Goroutine: monitorear y señalar cuando esté listo
	m.wg.Add(1)
	go m.monitor(ctx, bufferBytes)

	return nil
}

// Ready retorna un channel que envía el tmpPath cuando el archivo está listo para reproducir
// (moov descargado + buffer mínimo alcanzado).
func (m *Manager) Ready() <-chan string {
	return m.readyCh
}

// Progress retorna el estado actual de la descarga.
func (m *Manager) Progress() Progress {
	head := m.headWritten.Load()
	now := time.Now().UnixNano()
	lastT := m.lastTime.Load()
	lastB := m.lastBytes.Load()

	var speed float64
	elapsed := float64(now-lastT) / float64(time.Second)
	if elapsed > 0 {
		speed = float64(head-lastB) / (1024 * 1024) / elapsed
	}

	pct := 0
	if m.totalSize > 0 {
		pct = int(float64(head) / float64(m.totalSize) * 100)
		if pct > 100 {
			pct = 100
		}
	}

	peers := 0
	if m.t != nil {
		peers = m.t.Peers()
	}

	return Progress{
		HeadWritten: head,
		TotalSize:   m.totalSize,
		SpeedMBps:   speed,
		MoovReady:   m.moovReady.Load(),
		Peers:       peers,
		Percent:     pct,
	}
}

// TmpPath retorna la ruta del archivo temporal.
func (m *Manager) TmpPath() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.tmpPath
}

// TorrentProgress retorna los bytes completados del torrent (para monitoreo durante reproducción).
func (m *Manager) TorrentProgress() (bytesCompleted int64, peers int) {
	if m.t == nil {
		return 0, 0
	}
	return m.t.BytesCompleted(), m.t.Peers()
}

// Sync sincroniza el archivo temporal al disco.
func (m *Manager) Sync() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.tmpFile != nil {
		m.tmpFile.Sync()
	}
}

// Stop detiene la descarga y limpia recursos.
// No elimina el archivo temporal (el caller decide si hacerlo).
func (m *Manager) Stop() {
	if m.cancel != nil {
		m.cancel()
	}
	m.wg.Wait()

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.tmpFile != nil {
		m.tmpFile.Close()
		m.tmpFile = nil
	}
}

// Cleanup elimina el archivo temporal.
func (m *Manager) Cleanup() {
	m.mu.Lock()
	path := m.tmpPath
	m.mu.Unlock()

	if path != "" {
		os.Remove(path)
	}
}

// TorrentFiles retorna la lista de archivos del torrent activo.
func (m *Manager) TorrentFiles() []torrent.FileInfo {
	return m.info.Files
}

// ReadFile lee el contenido completo de un archivo del torrent por su índice.
// Bloquea hasta que las piezas del archivo estén descargadas.
func (m *Manager) ReadFile(fileIndex int) ([]byte, error) {
	if m.t == nil {
		return nil, fmt.Errorf("torrent no inicializado")
	}
	if fileIndex < 0 || fileIndex >= len(m.info.Files) {
		return nil, fmt.Errorf("índice de archivo inválido: %d", fileIndex)
	}

	reader, err := m.t.NewFileReader(fileIndex)
	if err != nil {
		return nil, fmt.Errorf("error creando reader: %w", err)
	}
	if closer, ok := reader.(io.Closer); ok {
		defer closer.Close()
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("error leyendo archivo: %w", err)
	}

	return data, nil
}

// --- métodos internos ---

func (m *Manager) createTempFile() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("obtener home directory: %w", err)
	}

	cacheDir := filepath.Join(home, ".cache", "p2pollo", "temp")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("crear directorio de caché: %w", err)
	}

	fileExt := filepath.Ext(m.info.Files[m.fileIndex].Path)
	if fileExt == "" {
		fileExt = ".mp4"
	}

	m.tmpPath = filepath.Join(cacheDir, fmt.Sprintf("p2pollo-stream-%d%s", time.Now().Unix(), fileExt))
	m.tmpFile, err = os.OpenFile(m.tmpPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("abrir archivo temporal: %w", err)
	}

	return nil
}

func (m *Manager) newFileReader() (io.ReadSeeker, error) {
	if len(m.info.Files) == 1 {
		return m.t.NewReader(), nil
	}
	return m.t.NewFileReader(m.fileIndex)
}

// downloadMoov descarga los últimos N bytes del archivo (moov atom para MP4 no-faststart).
func (m *Manager) downloadMoov(ctx context.Context, moovSize int64) {
	defer m.wg.Done()

	moovOffset := m.totalSize - moovSize

	reader, err := m.newFileReader()
	if err != nil {
		return
	}

	if _, err := reader.Seek(moovOffset, io.SeekStart); err != nil {
		return
	}

	buf := make([]byte, 64*1024)
	written := int64(0)
	for written < moovSize {
		select {
		case <-ctx.Done():
			return
		default:
		}

		n, readErr := reader.Read(buf)
		if n > 0 {
			m.mu.Lock()
			if m.tmpFile != nil {
				m.tmpFile.WriteAt(buf[:n], moovOffset+written)
			}
			m.mu.Unlock()
			written += int64(n)
		}
		if readErr != nil {
			break
		}
	}

	// Solo marcar como listo si realmente se descargó una parte significativa.
	// Si el reader falló inmediatamente (ej: "torrent data downloading disabled"),
	// written será 0 y no debemos marcar moov como listo.
	if written >= moovSize*9/10 {
		m.moovReady.Store(true)
	}
}

// downloadSequential copia el torrent secuencialmente al archivo temporal desde el inicio.
func (m *Manager) downloadSequential(ctx context.Context) {
	defer m.wg.Done()

	reader, err := m.newFileReader()
	if err != nil {
		return
	}

	buf := make([]byte, 64*1024)
	for {
		select {
		case <-ctx.Done():
			m.syncFile()
			return
		default:
		}

		n, readErr := reader.Read(buf)
		if n > 0 {
			offset := m.headWritten.Load()
			m.mu.Lock()
			if m.tmpFile != nil {
				m.tmpFile.WriteAt(buf[:n], offset)
			}
			m.mu.Unlock()
			m.headWritten.Add(int64(n))
		}
		if readErr != nil {
			m.syncFile()
			return
		}
	}
}

// monitor espera hasta que el moov atom esté listo y el buffer mínimo se haya alcanzado.
func (m *Manager) monitor(ctx context.Context, bufferBytes int64) {
	defer m.wg.Done()

	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Actualizar speed tracking
			m.lastBytes.Store(m.headWritten.Load())
			m.lastTime.Store(time.Now().UnixNano())

			if m.moovReady.Load() && m.headWritten.Load() >= bufferBytes {
				m.syncFile()
				select {
				case m.readyCh <- m.tmpPath:
				default:
				}
				return
			}
		}
	}
}

func (m *Manager) syncFile() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.tmpFile != nil {
		m.tmpFile.Sync()
	}
}
