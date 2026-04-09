package com.p2pollo.data

import com.squareup.moshi.Json
import com.squareup.moshi.JsonClass

// --- Películas ---

@JsonClass(generateAdapter = true)
data class MovieCard(
    val id: String = "",
    val title: String = "",
    val year: Int = 0,
    val rating: Double = 0.0,
    @Json(name = "posterUrl") val poster: String = "",
    val genres: String = "",
    @Json(name = "imdbId") val imdbId: String = "",
    val slug: String = ""
)

@JsonClass(generateAdapter = true)
data class MovieDetail(
    val id: String = "",
    val title: String = "",
    val year: Int = 0,
    val rating: Double = 0.0,
    @Json(name = "posterUrl") val poster: String = "",
    val description: String = "",
    val genres: String = "",
    @Json(name = "imdbId") val imdbId: String = "",
    val torrents: List<TorrentOption> = emptyList()
)

@JsonClass(generateAdapter = true)
data class TorrentOption(
    val quality: String = "",
    val type: String = "",
    val size: String = "",
    @Json(name = "magnetLink") val magnet: String = "",
    val fileIndex: Int = 0
)

// --- Series ---

@JsonClass(generateAdapter = true)
data class SeriesCard(
    val id: String = "",
    val title: String = "",
    val year: Int = 0,
    val rating: Double = 0.0,
    @Json(name = "posterUrl") val poster: String = "",
    val genres: String = "",
    @Json(name = "imdbId") val imdbId: String = ""
)

@JsonClass(generateAdapter = true)
data class EpisodeInfo(
    val season: Int = 0,
    val episode: Int = 0,
    val title: String = "",
    @Json(name = "airDate") val airDate: String = ""
)

@JsonClass(generateAdapter = true)
data class SeasonGroup(
    val number: Int = 0,
    val episodes: List<EpisodeInfo> = emptyList()
)

@JsonClass(generateAdapter = true)
data class SeriesDetail(
    val id: String = "",
    val title: String = "",
    val year: Int = 0,
    val rating: Double = 0.0,
    @Json(name = "posterUrl") val poster: String = "",
    val description: String = "",
    val genres: String = "",
    @Json(name = "imdbId") val imdbId: String = "",
    val seasons: List<SeasonGroup> = emptyList()
)

// --- Streaming ---

@JsonClass(generateAdapter = true)
data class PlayRequest(
    @Json(name = "magnetLink") val magnetLink: String,
    @Json(name = "fileIndex") val fileIndex: Int = 0
)

@JsonClass(generateAdapter = true)
data class StreamPathResponse(
    val path: String = "",
    val ready: Boolean = false
)

@JsonClass(generateAdapter = true)
data class ProgressResponse(
    val preparing: Boolean = false,
    val percent: Double = 0.0,
    val peers: Int = 0,
    @Json(name = "speedMBps") val speedMBps: Double = 0.0,
    @Json(name = "totalSize") val totalSize: Long = 0
)
