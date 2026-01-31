package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/davidmarco12/p2pollo/internal/client"
	"github.com/davidmarco12/p2pollo/internal/config"
	"github.com/davidmarco12/p2pollo/internal/player"
	"github.com/davidmarco12/p2pollo/internal/stream"
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

	// Crear gestor de streaming
	sm := stream.New(c, cfg)

	// Preparar stream
	fmt.Println("📡 Preparando stream...")
	streamInfo, err := sm.PrepareStream(t, playFileIndex)
	if err != nil {
		color.Red("❌ Error preparando stream: %v", err)
		return
	}

	// Buffering
	if !playNoBuffer {
		fmt.Printf("⏳ Buffering inicial (%d MB)...\n", cfg.Streaming.InitialBufferSize)

		bar := progressbar.NewOptions(100,
			progressbar.OptionSetDescription("Buffer"),
			progressbar.OptionShowBytes(false),
			progressbar.OptionSetWidth(50),
			progressbar.OptionShowCount(),
			progressbar.OptionSetPredictTime(false),
		)

		// Monitorear buffer
		done := make(chan bool)
		go func() {
			ticker := time.NewTicker(500 * time.Millisecond)
			defer ticker.Stop()
			defer close(done)

			for {
				select {
				case <-ticker.C:
					progress := streamInfo.Torrent.Progress()
					bar.Set(int(progress))

					if streamInfo.IsReady {
						bar.Finish()
						return
					}
				}
			}
		}()

		err = sm.WaitForBuffer(streamInfo)
		<-done

		if err != nil {
			color.Red("❌ Error en buffering: %v", err)
			return
		}

		fmt.Println()
		color.Green("✓ Buffer listo")
	}

	// Iniciar reproductor
	fmt.Println("🎬 Iniciando reproductor...")
	p, err := player.New(cfg)
	if err != nil {
		color.Red("❌ Error iniciando player: %v", err)
		fmt.Println("💡 Asegúrate de tener mpv instalado: https://mpv.io")
		return
	}
	defer p.Close()

	// Reproducir
	err = p.Play(streamInfo.Reader)
	if err != nil {
		color.Red("❌ Error reproduciendo: %v", err)
		return
	}

	color.Green("✓ Reproduciendo...")
	fmt.Println()

	// Monitorear reproducción
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Estadísticas
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	fmt.Println("📊 Estadísticas (Ctrl+C para detener):")
	fmt.Println()

	for {
		select {
		case <-sigChan:
			fmt.Println()
			color.Yellow("⏹  Deteniendo reproducción...")
			return
		case <-ticker.C:
			stats := sm.Stats(streamInfo)

			fmt.Printf("\r  Progreso: %.2f%% | Buffer: %.2f MB | Peers: %d   ",
				stats.Progress,
				float64(stats.BytesBuffered)/(1024*1024),
				stats.Peers,
			)
		}
	}
}
