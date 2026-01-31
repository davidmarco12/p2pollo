package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	verbose bool
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

func init() {
	cobra.OnInitialize(initConfig)

	// Flags globales
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "archivo de configuración (default: $HOME/.config/p2pollo/config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "output detallado")
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
