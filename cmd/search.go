package cmd

import (
	"fmt"

	"github.com/davidmarco12/p2pollo/internal/config"
	"github.com/davidmarco12/p2pollo/internal/search"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	searchMinSeeds int
	searchMaxSize  int64
	searchLimit    int
	searchSort     string
)

// searchCmd representa el comando search
var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Buscar torrents",
	Long: `Busca torrents en los trackers configurados.

Ejemplos:
  p2pollo search "big buck bunny"
  p2pollo search "movie 2024" --min-seeds 50
  p2pollo search "series" --sort seeds --limit 10`,
	Args: cobra.MinimumNArgs(1),
	Run:  runSearch,
}

func init() {
	rootCmd.AddCommand(searchCmd)

	searchCmd.Flags().IntVar(&searchMinSeeds, "min-seeds", 0, "mínimo de seeders")
	searchCmd.Flags().Int64Var(&searchMaxSize, "max-size", 0, "tamaño máximo en MB")
	searchCmd.Flags().IntVar(&searchLimit, "limit", 20, "número máximo de resultados")
	searchCmd.Flags().StringVar(&searchSort, "sort", "seeds", "ordenar por: seeds, size, date, relevance")
}

func runSearch(cmd *cobra.Command, args []string) {
	query := args[0]

	// Cargar configuración
	cfg, err := config.Load(cfgFile)
	if err != nil {
		cfg = config.DefaultConfig()
	}

	// Crear motor de búsqueda
	engine := search.New(cfg)

	// Preparar filtros
	filters := search.Filters{
		MinSeeds: searchMinSeeds,
		SortBy:   search.SortOrder(searchSort),
	}

	if searchMaxSize > 0 {
		filters.MaxSize = searchMaxSize * 1024 * 1024 // Convertir MB a bytes
	}

	// Buscar
	fmt.Printf("🔍 Buscando: %s\n\n", query)

	results, err := engine.SearchWithFilters(query, filters)
	if err != nil {
		color.Red("❌ Error: %v", err)
		return
	}

	if len(results) == 0 {
		color.Yellow("⚠️  No se encontraron resultados")
		return
	}

	// Limitar resultados
	if len(results) > searchLimit {
		results = results[:searchLimit]
	}

	// Mostrar resultados
	color.Green("✓ Encontrados %d resultados:\n", len(results))
	fmt.Println()

	for i, r := range results {
		// Número
		color.Cyan("[%d] ", i+1)

		// Nombre
		fmt.Printf("%s\n", r.Name)

		// Info
		fmt.Printf("    Tamaño: %s", search.FormatSize(r.Size))
		fmt.Printf(" | Seeds: ")

		if r.Seeds >= 50 {
			color.Green("%d", r.Seeds)
		} else if r.Seeds >= 10 {
			color.Yellow("%d", r.Seeds)
		} else {
			color.Red("%d", r.Seeds)
		}

		fmt.Printf(" | Leechers: %d", r.Leechers)
		fmt.Printf(" | Salud: %d%%", r.HealthScore())
		fmt.Println()

		// Magnet link (truncado)
		if verbose {
			fmt.Printf("    Magnet: %s\n", r.MagnetLink)
		}

		fmt.Println()
	}

	// Tip
	color.Cyan("💡 Tip: Usa --verbose para ver los magnet links completos")
	fmt.Println()
}
