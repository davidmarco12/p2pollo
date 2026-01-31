package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/davidmarco12/p2pollo/internal/client"
	"github.com/davidmarco12/p2pollo/internal/config"
	"github.com/davidmarco12/p2pollo/internal/player"
	"github.com/fatih/color"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

var (
	playFileIndex int
	playNoBuffer  bool
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

	playCmd.Flags().IntVar(&playFileIndex, "file", 0, "índice del archivo a reproducir")
	playCmd.Flags().BoolVar(&playNoBuffer, "no-buffer", false, "reproducir sin esperar buffer completo")
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
	c, err := client.New(cfg)
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
	err = t.WaitForInfo(30 * time.Second)
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
	fmt.Println()

	// Listar archivos si hay múltiples
	if len(info.Files) > 1 {
		color.Cyan("📁 Archivos disponibles:")
		for i, file := range info.Files {
			fmt.Printf("  [%d] %s (%.2f MB)\n", i, file.Path, float64(file.Length)/(1024*1024))
		}
		fmt.Printf("\nReproduciendo archivo: [%d] %s\n\n", playFileIndex, info.Files[playFileIndex].Path)
	}

	// Priorizar para streaming
	fmt.Println("📡 Priorizando para streaming...")
	readahead := 100
	if readahead > info.NumPieces {
		readahead = info.NumPieces
	}
	t.PrioritizeSequential(0, readahead-1)
	t.Download()

	// Buffer inicial
	if !playNoBuffer {
		bufferSize := int64(50 * 1024 * 1024) // 50MB para mejor compatibilidad con MP4
		fmt.Printf("⏳ Buffering (%.1f MB)...\n", float64(bufferSize)/(1024*1024))

		bar := progressbar.NewOptions(100,
			progressbar.OptionSetDescription("Buffer  "),
			progressbar.OptionSetWidth(50),
			progressbar.OptionShowCount(),
		)

		timeout := time.After(2 * time.Minute)
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		bufferReady := false
		for !bufferReady {
			select {
			case <-timeout:
				fmt.Println()
				color.Red("❌ Timeout - usa --no-buffer")
				return
			case <-ticker.C:
				buffered := t.BytesCompleted()
				progress := int((float64(buffered) / float64(bufferSize)) * 100)
				if progress > 100 {
					progress = 100
				}
				bar.Set(progress)

				if buffered >= bufferSize {
					bar.Finish()
					bufferReady = true
				}
			}
		}
		fmt.Println()
		color.Green("✓ Buffer listo (%.1f MB)\n", float64(t.BytesCompleted())/(1024*1024))
	}

	// Crear archivo temporal
	fmt.Println("🎬 Preparando para reproducción...")
	tmpPath := filepath.Join(os.TempDir(), fmt.Sprintf("p2pollo-stream-%d.mp4", time.Now().Unix()))
	tmpFileHandle, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		color.Red("❌ Error creando archivo temporal: %v", err)
		return
	}

	// Goroutine para copiar datos del torrent al temporal
	stopCopy := make(chan bool)

	go func() {
		fmt.Println("[DEBUG] Iniciando goroutine de copia")

		// Encontrar el archivo .part descargado por anacrolix
		cacheDir := filepath.Join(os.Getenv("USERPROFILE"), ".cache", "p2pollo", info.Name)

		// Obtener la ruta del archivo que queremos
		targetFile := filepath.Join(cacheDir, info.Files[playFileIndex].Path)
		// Anacrolix añade ".part" a los archivos que está descargando
		partFile := targetFile + ".part"

		fmt.Printf("[DEBUG] Intentando leer desde: %s\n", partFile)

		srcFile, err := os.Open(partFile)
		if err != nil {
			fmt.Printf("[DEBUG] ❌ Error abriendo parte file: %v\n", err)
			// Fallback: intentar leer desde el archivo completo
			srcFile, err = os.Open(targetFile)
			if err != nil {
				fmt.Printf("[DEBUG] ❌ Error abriendo archivo: %v\n", err)
				tmpFileHandle.Close()
				return
			}
		}
		defer srcFile.Close()

		fmt.Println("[DEBUG] ✓ Archivo fuente abierto, iniciando copia")
		buf := make([]byte, 64*1024)
		bytesWritten := int64(0)
		readCounter := 0
		for {
			select {
			case <-stopCopy:
				fmt.Printf("[DEBUG] ⏹ Deteniendo goroutine de copia (escribí %d bytes en %d reads)\n", bytesWritten, readCounter)
				tmpFileHandle.Sync()
				tmpFileHandle.Close()
				return
			default:
				n, err := srcFile.Read(buf)
				if n > 0 {
					readCounter++
					m, writeErr := tmpFileHandle.Write(buf[:n])
					bytesWritten += int64(m)
					if writeErr != nil {
						fmt.Printf("[DEBUG] ❌ Error escribiendo: %v\n", writeErr)
						tmpFileHandle.Close()
						return
					}
					tmpFileHandle.Sync()
				}
				if err != nil {
					if err == io.EOF {
						fmt.Printf("[DEBUG] ✓ EOF alcanzado (total: %d bytes en %d reads)\n", bytesWritten, readCounter)
						tmpFileHandle.Sync()
						tmpFileHandle.Close()
						return
					}
					fmt.Printf("[DEBUG] ❌ Error de lectura: %v\n", err)
					tmpFileHandle.Close()
					return
				}
			}
		}
	}()

	// Esperar a que TODO el archivo esté completamente descargado
	fmt.Println("⏳ Esperando descarga completa del archivo...")
	downloadTimeout := time.After(10 * time.Minute)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	downloadComplete := false
	targetSize := info.Files[playFileIndex].Length
	for !downloadComplete {
		select {
		case <-downloadTimeout:
			fmt.Println()
			close(stopCopy)
			color.Red("❌ Timeout esperando descarga")
			os.Remove(tmpPath)
			return
		case <-ticker.C:
			stat, _ := os.Stat(tmpPath)
			if stat != nil {
				currentSize := stat.Size()
				progress := int((float64(currentSize) / float64(targetSize)) * 100)
				if progress > 100 {
					progress = 100
				}
				fmt.Printf("\r⏳ Descarga: %.1f/%.1f MB (%d%%)",
					float64(currentSize)/(1024*1024),
					float64(targetSize)/(1024*1024),
					progress)

				if currentSize >= targetSize {
					fmt.Println()
					fmt.Printf("✓ Descarga completa: %.1f MB\n\n", float64(currentSize)/(1024*1024))
					downloadComplete = true
				}
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

	// Monitorear
	done := make(chan bool)
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				color.Cyan("  📊 %.1f%% | %.1f MB | %d peers",
					t.Progress(),
					float64(t.BytesCompleted())/(1024*1024),
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
	p.Close()

	// Esperar un poco más para asegurar que MPV liberó el archivo
	fmt.Println("⏳ Limpiando... (esperando 2 segundos)...")
	time.Sleep(2 * time.Second)

	fmt.Printf("🧹 Archivo temporal guardado (NO se eliminó): %s\n", tmpPath)
	// os.Remove(tmpPath)  // COMENTADO PARA DEBUGGING

	fmt.Println()
	color.Green("✓ Finalizado")
}
