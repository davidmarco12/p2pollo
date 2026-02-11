package cmd

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/davidmarco12/p2pollo/internal/config"
	"github.com/davidmarco12/p2pollo/internal/player"
	"github.com/davidmarco12/p2pollo/internal/stream"
	"github.com/davidmarco12/p2pollo/internal/torrent"
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
	cfg, err := config.Load(cfgFile)
	if err != nil {
		cfg = config.DefaultConfig()
	}

	// Opción experimental: streaming real via stdin
	if playUsePipe {
		runPlayPipe(cfg, args[0])
		return
	}

	ctx := context.Background()

	// Crear cliente torrent
	fmt.Println("🔧 Iniciando cliente P2P...")
	c, err := torrent.NewWithOptions(cfg, playLowMemory)
	if err != nil {
		color.Red("❌ Error creando cliente: %v", err)
		return
	}
	defer c.Close()

	// Preparar: agregar magnet, esperar metadata, auto-detectar archivo
	m := stream.New(cfg, c)

	fmt.Println("🔗 Agregando torrent...")
	fmt.Println("⏳ Obteniendo información del torrent...")
	info, fileIdx, err := m.Prepare(ctx, args[0], playFileIndex)
	if err != nil {
		color.Red("❌ Error: %v", err)
		return
	}

	printTorrentInfo(info, fileIdx)

	// Iniciar descarga
	opts := stream.Options{NoBuffer: playNoBuffer}
	if err := m.Start(ctx, opts); err != nil {
		color.Red("❌ Error iniciando streaming: %v", err)
		return
	}
	defer m.Stop()

	if playNoBuffer {
		fmt.Println("✓ Modo sin buffer (5MB mínimo)")
	} else {
		fmt.Printf("📽️  Buffer objetivo: 200 MB\n")
	}

	// Mostrar progreso hasta que esté listo
	ready := m.Ready()
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	timeout := time.After(30 * time.Minute)

	for {
		select {
		case tmpPath := <-ready:
			fmt.Println()
			prog := m.Progress()
			fmt.Printf("✓ Buffer listo (%.1f MB) - Iniciando reproducción\n\n", float64(prog.HeadWritten)/(1024*1024))

			// Lanzar mpv
			if err := launchPlayer(cfg, tmpPath, m); err != nil {
				color.Red("❌ %v", err)
				return
			}

			fmt.Printf("🧹 Archivo temporal: %s\n", tmpPath)
			color.Green("✓ Finalizado")
			return

		case <-timeout:
			fmt.Println()
			color.Red("❌ Timeout esperando buffer")
			return

		case <-ticker.C:
			printProgress(m.Progress())
		}
	}
}

// launchPlayer abre mpv y espera a que el usuario cierre la ventana.
func launchPlayer(cfg *config.Config, tmpPath string, m *stream.Manager) error {
	m.Sync()

	p, err := player.New(cfg)
	if err != nil {
		return fmt.Errorf("crear player: %w", err)
	}

	if err := p.PlayFile(tmpPath); err != nil {
		p.Close()
		return fmt.Errorf("abrir MPV: %w", err)
	}

	color.Green("✓ Reproduciendo (streaming mientras descarga)\n")
	fmt.Println("▶️  Reproduciendo... (cierra MPV cuando termines)")

	// Monitorear progreso mientras se reproduce
	stopMonitor := make(chan struct{})
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-stopMonitor:
				return
			case <-ticker.C:
				bytes, peers := m.TorrentProgress()
				pct := 0
				prog := m.Progress()
				if prog.TotalSize > 0 {
					pct = int(float64(bytes) / float64(prog.TotalSize) * 100)
				}
				color.Cyan("  📊 %d%% | %.1f MB | %d peers", pct, float64(bytes)/(1024*1024), peers)
			}
		}
	}()

	p.Wait()
	close(stopMonitor)
	p.Close()
	return nil
}

// printTorrentInfo muestra la información del torrent y la selección de archivo.
func printTorrentInfo(info torrent.TorrentInfo, fileIdx int) {
	color.Green("✓ Información obtenida:")
	fmt.Printf("  Nombre: %s\n", info.Name)
	fmt.Printf("  Tamaño: %.2f MB\n", float64(info.TotalLength)/(1024*1024))
	fmt.Printf("  Archivos: %d\n", len(info.Files))
	fmt.Println()

	if len(info.Files) > 1 {
		color.Cyan("📁 Archivos disponibles:")
		for i, file := range info.Files {
			marker := "  "
			if i == fileIdx {
				marker = "▶ "
			}
			fmt.Printf("  %s[%d] %s (%.2f MB)\n", marker, i, file.Path, float64(file.Length)/(1024*1024))
		}
		fmt.Printf("\nReproduciendo archivo: [%d] %s\n\n", fileIdx, info.Files[fileIdx].Path)
	} else {
		fmt.Printf("📁 Reproduciendo: %s\n\n", info.Files[fileIdx].Path)
	}
}

// printProgress muestra el progreso del buffer en una sola línea.
func printProgress(p stream.Progress) {
	moovStatus := "⏳"
	if p.MoovReady {
		moovStatus = "✓"
	}
	fmt.Printf("\r⏳ Buffer: %.1f/%.1f MB (%d%%) [%.2f MB/s] [moov: %s]",
		float64(p.HeadWritten)/(1024*1024),
		float64(p.TotalSize)/(1024*1024),
		p.Percent,
		p.SpeedMBps,
		moovStatus)
}

// runPlayPipe modo experimental de streaming por stdin (sin disco).
func runPlayPipe(cfg *config.Config, magnetLink string) {
	color.Cyan("🚀 Modo streaming real (stdin)")

	c, err := torrent.NewWithOptions(cfg, playLowMemory)
	if err != nil {
		color.Red("❌ Error creando cliente: %v", err)
		return
	}
	defer c.Close()

	t, err := c.AddMagnet(magnetLink)
	if err != nil {
		color.Red("❌ Error: %v", err)
		return
	}

	if err := t.WaitForInfo(60 * time.Second); err != nil {
		color.Red("❌ Error: %v", err)
		return
	}

	info := t.Info()
	fileIdx := torrent.FindMediaFile(info.Files)

	t.PrioritizeSequential(0, 100)
	t.Download()

	// Buffer mínimo (10 MB)
	bufferSize := int64(10 * 1024 * 1024)
	fmt.Printf("⏳ Buffering %.1f MB...\n", float64(bufferSize)/(1024*1024))
	for t.BytesCompleted() < bufferSize {
		time.Sleep(500 * time.Millisecond)
	}
	color.Green("✓ Buffer listo\n")

	var reader io.ReadSeeker
	if len(info.Files) == 1 {
		reader = t.NewReader()
	} else {
		reader, err = t.NewFileReader(fileIdx)
		if err != nil {
			color.Red("❌ Error: %v", err)
			return
		}
	}

	p, _ := player.New(cfg)
	defer p.Close()

	if err := p.Play(reader); err != nil {
		color.Red("❌ Error: %v", err)
		return
	}

	color.Green("✓ Streaming activo")
	fmt.Println("💡 Datos van directo de torrent → MPV (no usa disco)")

	go func() {
		for {
			time.Sleep(3 * time.Second)
			color.Cyan("  📊 %.1f%% | %d peers", t.Progress(), t.Peers())
		}
	}()

	p.Wait()
	color.Green("✓ Finalizado")
}
