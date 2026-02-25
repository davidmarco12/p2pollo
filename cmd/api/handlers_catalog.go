package main

import (
	"fmt"
	"net/http"
	"strconv"

	"p2pollo/internal/catalog"
	"github.com/gin-gonic/gin"
)

// handleGetPopular retorna películas populares desde JustWatch
func (s *Server) handleGetPopular(c *gin.Context) {
	page := 1
	if p := c.Query("page"); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n > 0 {
			page = n
		}
	}

	movies, err := s.justwatch.PopularMovies(page)
	if err != nil {
		s.log.Errorf("Error obteniendo películas populares de JustWatch: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	cards := make([]MovieCard, len(movies))
	for i, m := range movies {
		cards[i] = MovieCard{
			ID:        m.ID,
			ImdbID:    m.ImdbID,
			Title:     m.Title,
			Year:      m.Year,
			Rating:    m.Rating,
			PosterURL: m.PosterURL,
			Genres:    m.Genres,
		}
	}

	c.JSON(http.StatusOK, cards)
}

// handleSearchMovies busca películas por título
func (s *Server) handleSearchMovies(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query parameter 'q' is required"})
		return
	}

	movies, err := s.catalog.Search(query, 1)
	if err != nil {
		s.log.Errorf("Error buscando películas: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	cards := make([]MovieCard, len(movies))
	for i, m := range movies {
		cards[i] = MovieCard{
			ID:        m.ID,
			ImdbID:    m.ImdbID,
			Title:     m.Title,
			Year:      m.Year,
			Rating:    m.Rating,
			PosterURL: m.PosterURL,
			Genres:    m.Genres,
		}
	}

	c.JSON(http.StatusOK, cards)
}

// handleGetMovieDetails obtiene detalles de una película y busca torrents
func (s *Server) handleGetMovieDetails(c *gin.Context) {
	var req MovieDetailRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	movie := catalog.Movie{
		ID:        req.ID,
		ImdbID:    req.ImdbID,
		Title:     req.Title,
		Year:      req.Year,
		Rating:    req.Rating,
		PosterURL: req.PosterURL,
		Genres:    req.Genres,
	}

	// 1. Obtener descripción de YTS
	detail, err := s.catalog.Details(movie)
	if err != nil {
		s.log.Warnf("Error obteniendo detalles de YTS: %v", err)
		// Continuar con descripción vacía
		detail = &catalog.MovieDetail{
			Movie:       movie,
			Description: "",
			Runtime:     0,
		}
	}

	// 2. Buscar torrents en rargb y thepiratebay
	searchQuery := req.Title
	if req.Year > 0 {
		searchQuery = fmt.Sprintf("%s %d", req.Title, req.Year)
	}

	s.log.Infof("Buscando torrents para: %s", searchQuery)
	searchResults, err := s.scraper.Search(searchQuery)
	if err != nil {
		s.log.Errorf("Error buscando torrents: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 3. Convertir resultados a TorrentOption
	// Nota: No filtramos por extensión porque la mayoría de torrents no la incluyen en el nombre
	// rargb y ThePirateBay ya se especializan en películas/videos
	torrents := []TorrentOption{}
	for _, result := range searchResults {
		torrents = append(torrents, TorrentOption{
			Hash:       "", // rargb/thepiratebay no proveen hash directamente
			Quality:    extractQuality(result.Name),
			Type:       extractType(result.Name),
			Size:       result.Size,
			Seeds:      result.Seeds,
			Peers:      result.Leechers,
			MagnetLink: result.MagnetLink,
			Provider:   result.Source,
			FileName:   result.Name,      // Usar el nombre completo del torrent
			Subtitles:  result.Subtitles, // Subtítulos si están disponibles
		})
	}

	s.log.Infof("Encontrados %d torrents de %d resultados", len(torrents), len(searchResults))

	c.JSON(http.StatusOK, MovieDetailResponse{
		MovieCard: MovieCard{
			ID:        req.ID,
			ImdbID:    req.ImdbID,
			Title:     req.Title,
			Year:      req.Year,
			Rating:    req.Rating,
			PosterURL: req.PosterURL,
			Genres:    req.Genres,
		},
		Description: detail.Description,
		Runtime:     detail.Runtime,
		Torrents:    torrents,
	})
}
