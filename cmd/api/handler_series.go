package main

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"p2pollo/internal/catalog/tvmaze"

	"github.com/gin-gonic/gin"
)

// handleGetPopularSeries retorna series populares de TVmaze
func (s *Server) handleGetPopularSeries(c *gin.Context) {
	page := 0
	if p := c.Query("page"); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n > 0 {
			page = n - 1 // TVmaze es 0-indexed
		}
	}

	shows, err := s.tvmaze.PopularShows(page)
	if err != nil {
		s.log.Errorf("Error obteniendo series populares de TVmaze: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	result := showsToSeriesCards(shows)
	c.JSON(http.StatusOK, result)
}

// handleSearchSeries busca series por titulo en TVmaze
func (s *Server) handleSearchSeries(c *gin.Context) {
	query := strings.TrimSpace(c.Query("q"))
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "parametro q requerido"})
		return
	}

	shows, err := s.tvmaze.SearchShows(query)
	if err != nil {
		s.log.Errorf("Error buscando series '%s' en TVmaze: %v", query, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	result := showsToSeriesCards(shows)
	c.JSON(http.StatusOK, result)
}

// handleGetSeriesDetails retorna el detalle de una serie con sus temporadas/episodios
func (s *Server) handleGetSeriesDetails(c *gin.Context) {
	var req SeriesCard
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "request invalido"})
		return
	}

	var show *tvmaze.TVShow
	var err error

	// Buscar por IMDB ID primero, luego por nombre
	if req.ImdbID != "" {
		show, err = s.tvmaze.GetShowByIMDB(req.ImdbID)
	}
	if show == nil {
		results, searchErr := s.tvmaze.SearchShows(req.Title)
		if searchErr == nil && len(results) > 0 {
			show = &results[0]
			err = nil
		}
	}

	if err != nil || show == nil {
		s.log.Warnf("No se pudo encontrar '%s' en TVmaze", req.Title)
		c.JSON(http.StatusOK, SeriesDetailResponse{
			SeriesCard:  req,
			Description: "",
			Seasons:     []SeasonGroup{},
		})
		return
	}

	seasons, err := s.tvmaze.GetSeasons(show.ID)
	if err != nil {
		s.log.Errorf("Error obteniendo episodios de '%s': %v", req.Title, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var apiSeasons []SeasonGroup
	for _, season := range seasons {
		var episodes []EpisodeItem
		for _, ep := range season.Episodes {
			episodes = append(episodes, EpisodeItem{
				Season:  ep.Season,
				Episode: ep.Number,
				Title:   ep.Name,
				AirDate: ep.AirDate,
			})
		}
		apiSeasons = append(apiSeasons, SeasonGroup{Number: season.Number, Episodes: episodes})
	}

	// Actualizar IMDB ID si lo obtuvimos de TVmaze
	if req.ImdbID == "" && show.Externals.IMDB != "" {
		req.ImdbID = show.Externals.IMDB
	}

	c.JSON(http.StatusOK, SeriesDetailResponse{
		SeriesCard:  req,
		Description: stripHTMLTags(show.Summary),
		Seasons:     apiSeasons,
	})
}

// handleGetEpisodeTorrents busca torrents de EZTV + scrapers para un episodio
func (s *Server) handleGetEpisodeTorrents(c *gin.Context) {
	title := strings.TrimSpace(c.Query("title"))
	imdbID := strings.TrimSpace(c.Query("imdb"))

	season, err := strconv.Atoi(c.Query("season"))
	if err != nil || season < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "parametro season invalido"})
		return
	}

	episode, err := strconv.Atoi(c.Query("episode"))
	if err != nil || episode < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "parametro episode invalido"})
		return
	}

	var torrents []TorrentOption

	// Intentar EZTV primero (tiene datos por IMDB ID → más preciso)
	if imdbID != "" {
		s.log.Infof("Buscando en EZTV: %s S%02dE%02d (imdb: %s)", title, season, episode, imdbID)
		eztvTorrents, eztvErr := s.eztv.GetEpisodeTorrents(imdbID, season, episode)
		if eztvErr == nil {
			for _, t := range eztvTorrents {
				torrents = append(torrents, TorrentOption{
					Hash:       t.Hash,
					Quality:    detectQuality(t.Filename),
					Type:       "EZTV",
					Size:       formatBytes(t.Size),
					Seeds:      t.Seeds,
					Peers:      t.Peers,
					MagnetLink: t.MagnetURL,
					Provider:   "eztv",
					FileName:   t.Filename,
					Subtitles:  []string{},
				})
			}
		}
	}

	// Complementar con scrapers si EZTV no devolvió nada
	if len(torrents) == 0 && title != "" {
		query := fmt.Sprintf("%s S%02dE%02d", title, season, episode)
		s.log.Infof("Buscando en scrapers: %s", query)
		results, _ := s.scraper.Search(query)
		for _, r := range results {
			torrents = append(torrents, TorrentOption{
				Hash:       "",
				Quality:    detectQuality(r.Name),
				Type:       "",
				Size:       r.Size,
				Seeds:      r.Seeds,
				Peers:      r.Leechers,
				MagnetLink: r.MagnetLink,
				Provider:   r.Source,
				FileName:   r.Name,
				Subtitles:  r.Subtitles,
			})
		}
	}

	if torrents == nil {
		torrents = []TorrentOption{}
	}
	c.JSON(http.StatusOK, torrents)
}

// showsToSeriesCards convierte TVmaze shows a SeriesCards
func showsToSeriesCards(shows []tvmaze.TVShow) []SeriesCard {
	var result []SeriesCard
	for _, show := range shows {
		poster := tvmaze.ShowPoster(show)
		if poster == "" {
			continue
		}
		result = append(result, SeriesCard{
			ID:        strconv.Itoa(show.ID),
			ImdbID:    show.Externals.IMDB,
			Title:     show.Name,
			Year:      tvmaze.ShowYear(show),
			Rating:    show.Rating.Average,
			PosterURL: poster,
			Genres:    tvmaze.ShowGenres(show),
		})
	}
	if result == nil {
		return []SeriesCard{}
	}
	return result
}

var htmlTagRegex = regexp.MustCompile(`<[^>]+>`)

func stripHTMLTags(s string) string {
	return strings.TrimSpace(htmlTagRegex.ReplaceAllString(s, ""))
}

func detectQuality(name string) string {
	upper := strings.ToUpper(name)
	switch {
	case strings.Contains(upper, "2160P") || strings.Contains(upper, "4K"):
		return "2160p"
	case strings.Contains(upper, "1080P"):
		return "1080p"
	case strings.Contains(upper, "720P"):
		return "720p"
	case strings.Contains(upper, "480P"):
		return "480p"
	default:
		return "SD"
	}
}

func formatBytes(bytesStr string) string {
	if bytesStr == "" {
		return ""
	}
	n, err := strconv.ParseInt(bytesStr, 10, 64)
	if err != nil {
		return bytesStr
	}
	switch {
	case n >= 1<<30:
		return fmt.Sprintf("%.2f GB", float64(n)/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.0f MB", float64(n)/(1<<20))
	default:
		return fmt.Sprintf("%d KB", n/(1<<10))
	}
}
