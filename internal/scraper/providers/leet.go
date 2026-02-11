package providers

import (
	"github.com/davidmarco12/p2pollo/internal/scraper"
)

// Leet implementa el proveedor de busqueda para el sitio 1337x.
// Realiza scraping del HTML de la pagina de resultados para
// extraer los torrents disponibles.
type Leet struct {
	// baseURL es la URL base del sitio 1337x
	baseURL string
}

// NewLeet crea una nueva instancia del proveedor 1337x
func NewLeet() *Leet {
	return &Leet{
		baseURL: "https://1337x.to",
	}
}

// Search busca torrents en 1337x para la consulta dada.
// TODO: Implementar scraping HTTP de 1337x
//   - Hacer GET a {baseURL}/search/{query}/1/
//   - Parsear la tabla HTML de resultados
//   - Extraer nombre, tamano, seeds, leechers de cada fila
//   - Navegar a la pagina de detalle de cada torrent para obtener el magnet link
//   - Manejar paginacion si hay mas de una pagina de resultados
func (l *Leet) Search(query string) ([]scraper.Result, error) {
	// Stub: devuelve resultados vacios hasta que se implemente el scraping
	return []scraper.Result{}, nil
}
