package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/davidmarco12/p2pollo/internal/client"
	"github.com/davidmarco12/p2pollo/internal/config"
	"github.com/davidmarco12/p2pollo/internal/player"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	playFileIndex int
	playNoBuffer  bool
	playUsePipe   bool // Modo experimental: streaming real
	playLowMemory bool // Modo bajo consumo de memoria
)

// playCmd representa el comando play
var playCmd = &cobra.Command{
	Use:   "play <magnet-link>",
	Short: "Reproducir un torrent",
	Long: `Descarga y reproduce un torrent en streaming.

Ejemplos:
  p2pollo play "magnet:?xt=urn:btih:..."
  p2pollo play "magnet:?xt=..." --file 0
  p2pollo play "magnet:?xt=..." --no-buffer`,
	Args: cobra.ExactArgs(1),
	Run:  runPlay,
}

func init() {
	rootCmd.AddCommand(playCmd)

	playCmd.Flags().IntVar(&playFileIndex, "file", -1, "índice del archivo a reproducir (-1 = auto-detectar)")
	playCmd.Flags().BoolVar(&playNoBuffer, "no-buffer", false, "reproducir sin esperar buffer completo")
	playCmd.Flags().BoolVar(&playLowMemory, "low-memory", false, "modo bajo consumo de memoria (para máquinas con < 100MB RAM)")
}

func runPlay(cmd *cobra.Command, args []string) {
	magnetLink := args[0]

	// Cargar configuración
	cfg, err := config.Load(cfgFile)
	if err != nil {
		cfg = config.DefaultConfig()
	}

	// Crear cliente
	fmt.Println("🔧 Iniciando cliente P2P...")
	if playLowMemory {
		fmt.Println("💾 Modo bajo consumo de memoria activado")
	}
	c, err := client.NewWithOptions(cfg, playLowMemory)
	if err != nil {
		color.Red("❌ Error creando cliente: %v", err)
		return
	}
	defer c.Close()

	// Agregar torrent
	fmt.Println("🔗 Agregando torrent...")
	t, err := c.AddMagnet(magnetLink)
	if err != nil {
		color.Red("❌ Error agregando torrent: %v", err)
		return
	}

	// Esperar metadata
	fmt.Println("⏳ Obteniendo información del torrent...")
	err = t.WaitForInfo(60 * time.Second) // Aumentado a 60 segundos para conexiones lentas
	if err != nil {
		color.Red("❌ Error obteniendo metadata: %v", err)
		return
	}

	// Mostrar info
	info := t.Info()
	color.Green("✓ Información obtenida:")
	fmt.Printf("  Nombre: %s\n", info.Name)
	fmt.Printf("  Tamaño: %.2f MB\n", float64(info.TotalLength)/(1024*1024))
	fmt.Printf("  Archivos: %d\n", len(info.Files))
	fmt.Printf("  Peers: %d\n", t.Peers())
	fmt.Println()

	// Auto-detectar archivo multimedia si no se especificó --file
	if playFileIndex == -1 {
		playFileIndex = findMediaFile(info.Files)
	}

	// Validar que el índice del archivo sea válido
	if playFileIndex < 0 || playFileIndex >= len(info.Files) {
		color.Red("❌ Error: Índice de archivo inválido. Debe estar entre 0 y %d", len(info.Files)-1)
		return
	}

	// Listar archivos
	if len(info.Files) > 1 {
		color.Cyan("📁 Archivos disponibles:")
		for i, file := range info.Files {
			marker := "  "
			if i == playFileIndex {
				marker = "▶ "
			}
			fmt.Printf("  %s[%d] %s (%.2f MB)\n", marker, i, file.Path, float64(file.Length)/(1024*1024))
		}
		fmt.Printf("\nReproduciendo archivo: [%d] %s\n\n", playFileIndex, info.Files[playFileIndex].Path)
	} else {
		fmt.Printf("📁 Reproduciendo: %s\n\n", info.Files[playFileIndex].Path)
	}

	// Opción experimental: streaming real
	if playUsePipe {
		playWithPipeStreaming(t, cfg, info, playFileIndex)
		return
	}

	// Priorizar para streaming
	fmt.Println("📡 Priorizando para streaming...")
	readahead := 100
	if readahead > info.NumPieces {
		readahead = info.NumPieces
	}
	t.PrioritizeSequential(0, readahead-1)

	// Priorizar últimas piezas para el moov atom (MP4 no-faststart lo tienen al final)
	targetSize := info.Files[playFileIndex].Length
	moovSize := int64(5 * 1024 * 1024)
	if moovSize > targetSize {
		moovSize = targetSize
	}
	moovPieces := int(moovSize/info.PieceLength) + 1
	lastPiece := info.NumPieces - 1
	firstMoovPiece := lastPiece - moovPieces
	if firstMoovPiece < 0 {
		firstMoovPiece = 0
	}
	t.PrioritizeSequential(firstMoovPiece, lastPiece)
	fmt.Printf("  Piezas inicio: 0-%d, moov: %d-%d\n", readahead-1, firstMoovPiece, lastPiece)

	t.Download()

	// Crear archivo temporal en ~/.cache/p2pollo/temp/
	fmt.Println("🎬 Preparando para reproducción...")

	// Crear directorio de caché si no existe
	home, err := os.UserHomeDir()
	if err != nil {
		color.Red("❌ Error obteniendo home directory: %v", err)
		return
	}
	cacheDir := filepath.Join(home, ".cache", "p2pollo", "temp")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		color.Red("❌ Error creando directorio de caché: %v", err)
		return
	}

	// Usar la extensión real del archivo del torrent
	fileExt := filepath.Ext(info.Files[playFileIndex].Path)
	if fileExt == "" {
		fileExt = ".mp4"
	}
	tmpPath := filepath.Join(cacheDir, fmt.Sprintf("p2pollo-stream-%d%s", time.Now().Unix(), fileExt))
	tmpFileHandle, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		color.Red("❌ Error creando archivo temporal: %v", err)
		return
	}

	// Pre-allocar archivo para poder escribir el moov atom al final
	if err := tmpFileHandle.Truncate(targetSize); err != nil {
		color.Red("❌ Error pre-allocando archivo: %v", err)
		tmpFileHandle.Close()
		return
	}

	// Crear reader para el archivo específico usando anacrolix
	var fileReader io.ReadSeeker
	if len(info.Files) == 1 {
		fileReader = t.NewReader()
	} else {
		fileReader, err = t.NewFileReader(playFileIndex)
		if err != nil {
			color.Red("❌ Error creando reader: %v", err)
			tmpFileHandle.Close()
			return
		}
	}

	// Goroutine: descargar moov atom (final del archivo)
	// Los MP4 no-faststart tienen el índice al final, sin él mpv no puede abrir el archivo
	moovOffset := targetSize - moovSize
	moovDone := make(chan bool, 1)
	go func() {
		fmt.Printf("📡 Descargando índice del archivo (últimos %.1f MB)...\n", float64(moovSize)/(1024*1024))
		var moovReader io.ReadSeeker
		if len(info.Files) == 1 {
			moovReader = t.NewReader()
		} else {
			moovReader, _ = t.NewFileReader(playFileIndex)
		}
		moovReader.Seek(moovOffset, io.SeekStart)
		buf := make([]byte, 64*1024)
		written := int64(0)
		for written < moovSize {
			n, readErr := moovReader.Read(buf)
			if n > 0 {
				tmpFileHandle.WriteAt(buf[:n], moovOffset+written)
				written += int64(n)
			}
			if readErr != nil {
				break
			}
		}
		fmt.Printf("\n✓ Moov atom listo (%.1f MB)\n", float64(written)/(1024*1024))
		moovDone <- true
	}()

	// Goroutine: copia secuencial desde el inicio del archivo
	var headWritten int64
	stopCopy := make(chan bool)
	go func() {
		buf := make([]byte, 64*1024)
		for {
			select {
			case <-stopCopy:
				fmt.Printf("[DEBUG] ⏹ Copia detenida (%d bytes)\n", atomic.LoadInt64(&headWritten))
				tmpFileHandle.Sync()
				return
			default:
				n, readErr := fileReader.Read(buf)
				if n > 0 {
					offset := atomic.LoadInt64(&headWritten)
					tmpFileHandle.WriteAt(buf[:n], offset)
					atomic.AddInt64(&headWritten, int64(n))
				}
				if readErr != nil {
					if readErr == io.EOF {
						fmt.Printf("[DEBUG] ✓ Copia completa (%d bytes)\n", atomic.LoadInt64(&headWritten))
					}
					tmpFileHandle.Sync()
					return
				}
			}
		}
	}()

	// Determinar buffer mínimo para lanzar MPV
	var minBufferToPlay int64
	if playNoBuffer {
		minBufferToPlay = 5 * 1024 * 1024
		fmt.Println("✓ Modo sin buffer (5MB mínimo)")
	} else {
		minBufferToPlay = 200 * 1024 * 1024
		if minBufferToPlay > targetSize {
			minBufferToPlay = targetSize
		}
		fmt.Printf("📽️  Buffer objetivo: %.0f MB\n", float64(minBufferToPlay)/(1024*1024))
	}

	downloadTimeout := time.After(30 * time.Minute)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	moovIsReady := false
	bufferReady := false
	lastSize := int64(0)
	lastTime := time.Now()

	// Esperar: moov atom descargado + buffer mínimo alcanzado
	for !bufferReady {
		select {
		case <-moovDone:
			moovIsReady = true
		case <-downloadTimeout:
			fmt.Println()
			close(stopCopy)
			color.Red("❌ Timeout esperando buffer")
			tmpFileHandle.Close()
			os.Remove(tmpPath)
			return
		case <-ticker.C:
			currentHead := atomic.LoadInt64(&headWritten)

			elapsed := time.Since(lastTime).Seconds()
			if elapsed > 0 {
				speed := float64(currentHead-lastSize) / (1024 * 1024) / elapsed
				progress := int((float64(currentHead) / float64(targetSize)) * 100)
				if progress > 100 {
					progress = 100
				}
				moovStatus := "⏳"
				if moovIsReady {
					moovStatus = "✓"
				}
				fmt.Printf("\r⏳ Buffer: %.1f/%.1f MB (%d%%) [%.2f MB/s] [moov: %s]",
					float64(currentHead)/(1024*1024),
					float64(targetSize)/(1024*1024),
					progress,
					speed,
					moovStatus)

				lastSize = currentHead
				lastTime = time.Now()
			}

			if moovIsReady && currentHead >= minBufferToPlay {
				fmt.Println()
				fmt.Printf("✓ Buffer listo (%.1f MB) - Iniciando reproducción\n\n", float64(currentHead)/(1024*1024))
				tmpFileHandle.Sync()
				bufferReady = true
			}
		}
	}

	// Abrir MPV
	fmt.Println("🎬 Abriendo MPV...")

	// Hacer Sync() del archivo antes de lanzar MPV
	if f, err := os.OpenFile(tmpPath, os.O_RDONLY, 0644); err == nil {
		f.Sync()
		f.Close()
	}

	p, err := player.New(cfg)
	if err != nil {
		color.Red("❌ Error: %v", err)
		close(stopCopy)
		os.Remove(tmpPath)
		return
	}

	err = p.PlayFile(tmpPath)
	if err != nil {
		color.Red("❌ Error al abrir MPV: %v", err)
		close(stopCopy)
		os.Remove(tmpPath)
		p.Close()
		return
	}

	color.Green("✓ Reproduciendo (streaming mientras descarga)\n")

	// Monitorear progreso de descarga mientras se reproduce
	done := make(chan bool)
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				bytesCompleted := t.BytesCompleted()
				progress := int((float64(bytesCompleted) / float64(targetSize)) * 100)
				if progress > 100 {
					progress = 100
				}

				color.Cyan("  📊 %d%% | %.1f MB | %d peers",
					progress,
					float64(bytesCompleted)/(1024*1024),
					t.Peers())
			}
		}
	}()

	fmt.Println("▶️  Reproduciendo... (cierra MPV cuando termines)\n")
	fmt.Printf("📁 Archivo: %s\n", tmpPath)
	fmt.Println("Esperando a que cierres MPV...")

	// Debug: dar tiempo para que MPV se abra
	time.Sleep(1 * time.Second)
	fmt.Println("[DEBUG] Llamando a p.Wait()...")

	waitErr := p.Wait() // Espera a que el usuario cierre MPV
	if waitErr != nil {
		fmt.Printf("[DEBUG] Error en Wait: %v\n", waitErr)
	} else {
		fmt.Println("[DEBUG] p.Wait() completó sin error")
	}
	close(done)

	close(stopCopy)
	tmpFileHandle.Close()
	p.Close()

	// Esperar un poco más para asegurar que MPV liberó el archivo
	fmt.Println("⏳ Limpiando... (esperando 2 segundos)...")
	time.Sleep(2 * time.Second)

	fmt.Printf("🧹 Archivo temporal guardado (NO se eliminó): %s\n", tmpPath)
	// os.Remove(tmpPath)  // COMENTADO PARA DEBUGGING

	fmt.Println()
	color.Green("✓ Finalizado")
}

func playWithPipeStreaming(t *client.Torrent, cfg *config.Config, info client.TorrentInfo, fileIdx int) {
	color.Cyan("🚀 Modo streaming real (stdin)")

	// Priorizar piezas
	fmt.Println("📡 Priorizando...")
	t.PrioritizeSequential(0, 100)
	t.Download()

	// Buffer mínimo (solo 10 MB)
	bufferSize := int64(10 * 1024 * 1024)
	fmt.Printf("⏳ Buffering %.1f MB...\n", float64(bufferSize)/(1024*1024))

	for t.BytesCompleted() < bufferSize {
		time.Sleep(500 * time.Millisecond)
	}

	color.Green("✓ Buffer listo\n")

	// Crear reader directo del torrent
	var reader io.ReadSeeker
	if len(info.Files) == 1 {
		reader = t.NewReader()
	} else {
		var err error
		reader, err = t.NewFileReader(fileIdx)
		if err != nil {
			color.Red("❌ Error: %v", err)
			return
		}
	}

	// Iniciar MPV
	p, _ := player.New(cfg)
	defer p.Close()

	// CLAVE: Play con reader (stdin) en lugar de PlayFile
	err := p.Play(reader)
	if err != nil {
		color.Red("❌ Error: %v", err)
		return
	}

	color.Green("✓ Streaming activo")
	fmt.Println("💡 Datos van directo de torrent → MPV (no usa disco)\n")

	// Monitor
	go func() {
		for {
			time.Sleep(3 * time.Second)
			color.Cyan("  📊 %.1f%% | %d peers", t.Progress(), t.Peers())
		}
	}()

	p.Wait()
	color.Green("✓ Finalizado")
}

// findMediaFile busca el archivo multimedia más grande en la lista de archivos.
// Retorna el índice del archivo con extensión de video más grande, o 0 si no encuentra ninguno.
func findMediaFile(files []client.FileInfo) int {
	videoExts := map[string]bool{
		".mkv": true, ".mp4": true, ".avi": true, ".webm": true,
		".mov": true, ".flv": true, ".wmv": true, ".m4v": true,
		".ts": true, ".mpg": true, ".mpeg": true,
	}

	bestIdx := 0
	bestSize := int64(0)
	for i, f := range files {
		ext := strings.ToLower(filepath.Ext(f.Path))
		if videoExts[ext] && f.Length > bestSize {
			bestIdx = i
			bestSize = f.Length
		}
	}
	return bestIdx
}
