package catalog

import (
	"fmt"
	"net/url"
	"strings"
)

// Movie representa una pelicula con su metadata para mostrar en la grilla
type Movie struct {
	ID        string  `json:"id"`
	ImdbID    string  `json:"imdbId"`
	Title     string  `json:"title"`
	Year      int     `json:"year"`
	Rating    float64 `json:"rating"`
	PosterURL string  `json:"posterUrl"`
	Genres    string  `json:"genres"`
	Slug      string  `json:"slug"`
}

// MovieDetail contiene la informacion completa de una pelicula
// incluyendo los torrents disponibles
type MovieDetail struct {
	Movie
	Description string    `json:"description"`
	Runtime     int       `json:"runtime"`
	Torrents    []Torrent `json:"torrents"`
}

// Torrent representa una opcion de descarga para una pelicula
type Torrent struct {
	Hash       string   `json:"hash"`
	Quality    string   `json:"quality"`
	Type       string   `json:"type"`
	Size       string   `json:"size"`
	Seeds      int      `json:"seeds"`
	Peers      int      `json:"peers"`
	MagnetLink string   `json:"magnetLink"`
	FileName   string   `json:"fileName"`   // Nombre del archivo construido
	Subtitles  []string `json:"subtitles"`  // Idiomas de subtítulos disponibles
}

// CatalogProvider es la interfaz para proveedores de catalogo de peliculas
type CatalogProvider interface {
	// Popular retorna peliculas populares/recientes para la pagina de inicio
	Popular(page int) ([]Movie, error)

	// Search busca peliculas por titulo
	Search(query string, page int) ([]Movie, error)

	// Details obtiene el detalle completo de una pelicula incluyendo torrents
	Details(movie Movie) (*MovieDetail, error)
}

// BuildMagnetLink construye un magnet link a partir de un hash y titulo
func BuildMagnetLink(hash, title string, trackers []string) string {
	var b strings.Builder
	b.WriteString("magnet:?xt=urn:btih:")
	b.WriteString(hash)
	b.WriteString("&dn=")
	b.WriteString(url.QueryEscape(title))
	for _, tr := range trackers {
		b.WriteString("&tr=")
		b.WriteString(url.QueryEscape(tr))
	}
	return b.String()
}

// HealthScore calcula un puntaje de salud (0-100) basado en seeds/peers
func HealthScore(seeds, peers int) int {
	total := seeds + peers
	if total == 0 {
		return 0
	}
	return int(float64(seeds) / float64(total) * 100)
}

// DefaultTrackers son trackers comunes para construir magnet links
var DefaultTrackers = []string{
	"udp://open.demonii.com:1337/announce",
	"udp://tracker.openbittorrent.com:80",
	"udp://tracker.opentrackr.org:1337/announce",
	"udp://p4p.arenabg.com:1337",
	"udp://tracker.torrent.eu.org:451/announce",
	"udp://open.stealth.si:80/announce",
}

// FormatTracker construye la URL completa de un tracker para la query
func FormatTracker(tracker string) string {
	return fmt.Sprintf("&tr=%s", url.QueryEscape(tracker))
}
