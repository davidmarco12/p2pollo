package cmd

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	verbose bool
	profile bool
	pprof   string
)

// rootCmd representa el comando base
var rootCmd = &cobra.Command{
	Use:   "p2pollo",
	Short: "Cliente de streaming P2P",
	Long: `Streaming CLI - Cliente de streaming P2P para BitTorrent

Permite buscar, descargar y reproducir contenido multimedia
usando la red BitTorrent de forma optimizada para streaming.

Ejemplos:
  p2pollo search "big buck bunny"
  p2pollo play <magnet-link>
  p2pollo config --show`,
	Version: "0.1.0",
}

// Execute ejecuta el comando root
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// startProfiling inicia el servidor de pprof si está habilitado
func startProfiling() {
	if profile {
		go func() {
			fmt.Fprintf(os.Stderr, "\n🔍 PROFILING ENABLED\n")
			fmt.Fprintf(os.Stderr, "   Servidor en http://localhost:%s/debug/pprof/\n\n", pprof)
			fmt.Fprintf(os.Stderr, "   Comandos:\n")
			fmt.Fprintf(os.Stderr, "   Heap:      go tool pprof http://localhost:%s/debug/pprof/heap\n", pprof)
			fmt.Fprintf(os.Stderr, "   CPU (10s): go tool pprof http://localhost:%s/debug/pprof/profile?seconds=10\n", pprof)
			fmt.Fprintf(os.Stderr, "   Goroutine: go tool pprof http://localhost:%s/debug/pprof/goroutine\n", pprof)
			fmt.Fprintf(os.Stderr, "   Mutex:     go tool pprof http://localhost:%s/debug/pprof/mutex\n\n", pprof)

			if err := http.ListenAndServe("localhost:"+pprof, nil); err != nil {
				fmt.Fprintf(os.Stderr, "❌ Profiling error: %v\n", err)
			}
		}()
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	cobra.OnInitialize(startProfiling)

	// Flags globales
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "archivo de configuración (default: $HOME/.config/p2pollo/config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "output detallado")
	rootCmd.PersistentFlags().BoolVar(&profile, "profile", false, "habilitar profiling pprof")
	rootCmd.PersistentFlags().StringVar(&pprof, "pprof-port", "6060", "puerto para servidor pprof")
	playCmd.Flags().BoolVar(&playUsePipe, "pipe", false, "streaming real via stdin (experimental)")
}

// initConfig lee el archivo de configuración
func initConfig() {
	if cfgFile != "" {
		// Usar el archivo especificado
		viper.SetConfigFile(cfgFile)
	} else {
		// Buscar en ubicaciones por defecto
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		viper.AddConfigPath(home + "/.config/p2pollo")
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
	}

	viper.AutomaticEnv()

	// Si existe el archivo, leerlo
	if err := viper.ReadInConfig(); err == nil && verbose {
		fmt.Fprintln(os.Stderr, "Usando archivo de config:", viper.ConfigFileUsed())
	}
}
