package com.p2pollo.data

import com.squareup.moshi.Moshi
import com.squareup.moshi.kotlin.reflect.KotlinJsonAdapterFactory
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.flow
import okhttp3.OkHttpClient
import retrofit2.Retrofit
import retrofit2.converter.moshi.MoshiConverterFactory
import java.util.concurrent.TimeUnit

object Repository {

    private val moshi = Moshi.Builder()
        .addLast(KotlinJsonAdapterFactory())
        .build()

    private val okHttp = OkHttpClient.Builder()
        .connectTimeout(10, TimeUnit.SECONDS)
        .readTimeout(30, TimeUnit.SECONDS)
        .build()

    private val retrofit = Retrofit.Builder()
        .baseUrl("http://127.0.0.1:9876/")
        .client(okHttp)
        .addConverterFactory(MoshiConverterFactory.create(moshi))
        .build()

    val api: ApiService = retrofit.create(ApiService::class.java)

    // --- Películas ---

    suspend fun getPopularMovies(): Result<List<MovieCard>> = runCatching {
        api.getPopularMovies()
    }

    suspend fun searchMovies(query: String): Result<List<MovieCard>> = runCatching {
        api.searchMovies(query)
    }

    suspend fun getMovieDetails(movie: MovieCard): Result<MovieDetail> = runCatching {
        api.getMovieDetails(movie)
    }

    // --- Series ---

    suspend fun getPopularSeries(): Result<List<SeriesCard>> = runCatching {
        api.getPopularSeries()
    }

    suspend fun searchSeries(query: String): Result<List<SeriesCard>> = runCatching {
        api.searchSeries(query)
    }

    suspend fun getSeriesDetails(series: SeriesCard): Result<SeriesDetail> = runCatching {
        api.getSeriesDetails(series)
    }

    suspend fun getEpisodeTorrents(
        title: String,
        imdbId: String,
        season: Int,
        episode: Int
    ): Result<List<TorrentOption>> = runCatching {
        api.getEpisodeTorrents(title, imdbId, season, episode)
    }

    // --- Streaming ---

    suspend fun play(magnet: String, fileIndex: Int = 0): Result<Unit> = runCatching {
        api.play(PlayRequest(magnetLink = magnet, fileIndex = fileIndex))
    }

    suspend fun stopStream(): Result<Unit> = runCatching {
        api.stopStream()
    }

    suspend fun getStreamPath(): Result<StreamPathResponse> = runCatching {
        api.getStreamPath()
    }

    suspend fun getProgress(): Result<ProgressResponse> = runCatching {
        api.getProgress()
    }

    /** Poll stream path each second until ready, emitting each response. */
    fun streamPathFlow(): Flow<StreamPathResponse> = flow {
        while (true) {
            val result = getStreamPath()
            if (result.isSuccess) {
                val resp = result.getOrThrow()
                emit(resp)
                if (resp.ready) break
            }
            kotlinx.coroutines.delay(1_000)
        }
    }
}
