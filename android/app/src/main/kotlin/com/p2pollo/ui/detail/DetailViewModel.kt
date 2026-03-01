package com.p2pollo.ui.detail

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.p2pollo.data.*
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch

class DetailViewModel : ViewModel() {

    private val _movieDetail = MutableStateFlow<MovieDetail?>(null)
    val movieDetail: StateFlow<MovieDetail?> = _movieDetail

    private val _seriesDetail = MutableStateFlow<SeriesDetail?>(null)
    val seriesDetail: StateFlow<SeriesDetail?> = _seriesDetail

    private val _episodeTorrents = MutableStateFlow<List<TorrentOption>>(emptyList())
    val episodeTorrents: StateFlow<List<TorrentOption>> = _episodeTorrents

    private val _selectedEpisode = MutableStateFlow<EpisodeInfo?>(null)
    val selectedEpisode: StateFlow<EpisodeInfo?> = _selectedEpisode

    private val _isLoading = MutableStateFlow(false)
    val isLoading: StateFlow<Boolean> = _isLoading

    private val _error = MutableStateFlow<String?>(null)
    val error: StateFlow<String?> = _error

    fun loadMovieDetail(movie: MovieCard) {
        viewModelScope.launch {
            _isLoading.value = true
            _error.value = null
            Repository.getMovieDetails(movie)
                .onSuccess { _movieDetail.value = it }
                .onFailure { _error.value = it.message }
            _isLoading.value = false
        }
    }

    fun loadSeriesDetail(series: SeriesCard) {
        viewModelScope.launch {
            _isLoading.value = true
            _error.value = null
            Repository.getSeriesDetails(series)
                .onSuccess { _seriesDetail.value = it }
                .onFailure { _error.value = it.message }
            _isLoading.value = false
        }
    }

    fun loadEpisodeTorrents(
        title: String,
        imdbId: String,
        season: Int,
        episode: Int,
        episodeInfo: EpisodeInfo
    ) {
        // Toggle: si ya está seleccionado, deseleccionar
        if (_selectedEpisode.value == episodeInfo) {
            _selectedEpisode.value = null
            _episodeTorrents.value = emptyList()
            return
        }

        viewModelScope.launch {
            _isLoading.value = true
            _error.value = null
            _selectedEpisode.value = episodeInfo
            Repository.getEpisodeTorrents(title, imdbId, season, episode)
                .onSuccess { _episodeTorrents.value = it }
                .onFailure {
                    _error.value = it.message
                    _episodeTorrents.value = emptyList()
                }
            _isLoading.value = false
        }
    }
}
