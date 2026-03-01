package com.p2pollo.data

import retrofit2.http.*

interface ApiService {

    // --- Películas ---
    @GET("api/popular")
    suspend fun getPopularMovies(): List<MovieCard>

    @GET("api/search")
    suspend fun searchMovies(@Query("q") query: String): List<MovieCard>

    @POST("api/movie/details")
    suspend fun getMovieDetails(@Body movie: MovieCard): MovieDetail

    // --- Series ---
    @GET("api/series/popular")
    suspend fun getPopularSeries(): List<SeriesCard>

    @GET("api/series/search")
    suspend fun searchSeries(@Query("q") query: String): List<SeriesCard>

    @POST("api/series/details")
    suspend fun getSeriesDetails(@Body series: SeriesCard): SeriesDetail

    @GET("api/series/episode/torrents")
    suspend fun getEpisodeTorrents(
        @Query("title") title: String,
        @Query("imdb") imdbId: String,
        @Query("season") season: Int,
        @Query("episode") episode: Int
    ): List<TorrentOption>

    // --- Streaming ---
    @POST("api/play")
    suspend fun play(@Body request: PlayRequest)

    @GET("api/stream-path")
    suspend fun getStreamPath(): StreamPathResponse

    @GET("api/progress")
    suspend fun getProgress(): ProgressResponse

    @POST("api/stop")
    suspend fun stopStream()
}
