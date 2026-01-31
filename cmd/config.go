package cmd

import (
	"fmt"
	"os"

	"github.com/davidmarco12/p2pollo/internal/config"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	configShow bool
	configInit bool
)

// configCmd representa el comando config
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Gestionar configuración",
	Long: `Gestiona la configuración de la aplicación.

Ejemplos:
  p2pollo config --show
  p2pollo config --init
  p2pollo config --config custom.yaml --show`,
	Run: runConfig,
}

func init() {
	rootCmd.AddCommand(configCmd)

	configCmd.Flags().BoolVar(&configShow, "show", false, "mostrar configuración actual")
	configCmd.Flags().BoolVar(&configInit, "init", false, "crear configuración por defecto")
}

func runConfig(cmd *cobra.Command, args []string) {
	if configInit {
		initializeConfig()
		return
	}

	if configShow {
		showConfig()
		return
	}

	// Si no hay flags, mostrar ayuda
	cmd.Help()
}

func initializeConfig() {
	// Crear configuración por defecto
	cfg := config.DefaultConfig()

	// Determinar ruta
	configPath := cfgFile
	if configPath == "" {
		configPath = cfg.Paths.ConfigFile
	}

	// Verificar si ya existe
	if _, err := os.Stat(configPath); err == nil {
		color.Yellow("⚠️  El archivo de configuración ya existe: %s", configPath)
		fmt.Print("¿Sobrescribir? (s/N): ")

		var response string
		fmt.Scanln(&response)

		if response != "s" && response != "S" {
			fmt.Println("Cancelado")
			return
		}
	}

	// Crear directorios
	err := cfg.EnsureDirectories()
	if err != nil {
		color.Red("❌ Error creando directorios: %v", err)
		return
	}

	// Guardar
	err = cfg.Save(configPath)
	if err != nil {
		color.Red("❌ Error guardando configuración: %v", err)
		return
	}

	color.Green("✓ Configuración creada: %s", configPath)
}

func showConfig() {
	// Cargar configuración
	cfg, err := config.Load(cfgFile)
	if err != nil {
		color.Red("❌ Error cargando configuración: %v", err)
		fmt.Println()
		color.Yellow("💡 Crea una con: p2pollo config --init")
		return
	}

	// Mostrar configuración
	color.Cyan("📋 Configuración Actual")
	fmt.Println()

	// Trackers
	color.Green("🌐 Trackers (%d):", len(cfg.Trackers))
	for i, tracker := range cfg.Trackers {
		fmt.Printf("  [%d] %s\n", i+1, tracker)
	}
	fmt.Println()

	// Cliente
	color.Green("⚙️  Cliente:")
	fmt.Printf("  Puerto: %d", cfg.Client.Port)
	if cfg.Client.Port == 0 {
		color.Yellow(" (aleatorio)")
	}
	fmt.Println()
	fmt.Printf("  Límite descarga: %d KB/s", cfg.Client.DownloadRateLimit)
	if cfg.Client.DownloadRateLimit == 0 {
		color.Yellow(" (sin límite)")
	}
	fmt.Println()
	fmt.Printf("  Límite subida: %d KB/s", cfg.Client.UploadRateLimit)
	if cfg.Client.UploadRateLimit == 0 {
		color.Yellow(" (sin límite)")
	}
	fmt.Println()
	fmt.Printf("  Conexiones máximas: %d\n", cfg.Client.MaxConnections)
	fmt.Println()

	// Player
	color.Green("🎬 Player:")
	fmt.Printf("  MPV: %s\n", cfg.Player.MPVPath)
	fmt.Printf("  Opciones: %v\n", cfg.Player.MPVOptions)
	fmt.Println()

	// Streaming
	color.Green("📡 Streaming:")
	fmt.Printf("  Buffer inicial: %d MB\n", cfg.Streaming.InitialBufferSize)
	fmt.Printf("  Buffer mínimo: %d MB\n", cfg.Streaming.MinBufferSize)
	fmt.Printf("  Descarga secuencial: %v\n", cfg.Streaming.SequentialDownload)
	fmt.Printf("  Readahead pieces: %d\n", cfg.Streaming.ReadaheadPieces)
	fmt.Println()

	// Búsqueda
	color.Green("🔍 Búsqueda:")
	fmt.Printf("  Resultados por tracker: %d\n", cfg.Search.MaxResultsPerTracker)
	fmt.Printf("  Timeout: %d segundos\n", cfg.Search.SearchTimeout)
	fmt.Println()

	// Rutas
	color.Green("📁 Rutas:")
	fmt.Printf("  Caché: %s\n", cfg.Paths.CacheDir)
	fmt.Printf("  Logs: %s\n", cfg.Paths.LogDir)
	fmt.Printf("  Config: %s\n", cfg.Paths.ConfigFile)
	fmt.Println()

	// Logging
	color.Green("📝 Logging:")
	fmt.Printf("  Nivel: %s\n", cfg.Logging.Level)
	fmt.Printf("  Formato: %s\n", cfg.Logging.Format)
	fmt.Printf("  Archivo: %v\n", cfg.Logging.FileLogging)
	fmt.Println()

	// Info de archivo
	if cfgFile != "" {
		color.Cyan("📄 Archivo: %s", cfgFile)
	} else {
		color.Cyan("📄 Archivo: %s", cfg.Paths.ConfigFile)
	}
	fmt.Println()
}
