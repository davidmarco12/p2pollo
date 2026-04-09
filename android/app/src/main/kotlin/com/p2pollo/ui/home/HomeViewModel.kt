package com.p2pollo.ui.home

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.p2pollo.data.MovieCard
import com.p2pollo.data.Repository
import com.p2pollo.data.SeriesCard
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch

class HomeViewModel : ViewModel() {

    private val _movies = MutableStateFlow<List<MovieCard>>(emptyList())
    val movies: StateFlow<List<MovieCard>> = _movies

    private val _series = MutableStateFlow<List<SeriesCard>>(emptyList())
    val series: StateFlow<List<SeriesCard>> = _series

    private val _isLoading = MutableStateFlow(false)
    val isLoading: StateFlow<Boolean> = _isLoading

    private val _error = MutableStateFlow<String?>(null)
    val error: StateFlow<String?> = _error

    private val _searchQuery = MutableStateFlow("")
    val searchQuery: StateFlow<String> = _searchQuery

    fun setSearchQuery(q: String) {
        _searchQuery.value = q
    }

    fun loadPopularMovies() {
        viewModelScope.launch {
            _isLoading.value = true
            _error.value = null
            Repository.getPopularMovies()
                .onSuccess { _movies.value = it }
                .onFailure { _error.value = it.message }
            _isLoading.value = false
        }
    }

    fun loadPopularSeries() {
        viewModelScope.launch {
            _isLoading.value = true
            _error.value = null
            Repository.getPopularSeries()
                .onSuccess { _series.value = it }
                .onFailure { _error.value = it.message }
            _isLoading.value = false
        }
    }

    fun searchMovies(query: String) {
        viewModelScope.launch {
            _isLoading.value = true
            _error.value = null
            Repository.searchMovies(query)
                .onSuccess { _movies.value = it }
                .onFailure { _error.value = it.message }
            _isLoading.value = false
        }
    }

    fun searchSeries(query: String) {
        viewModelScope.launch {
            _isLoading.value = true
            _error.value = null
            Repository.searchSeries(query)
                .onSuccess { _series.value = it }
                .onFailure { _error.value = it.message }
            _isLoading.value = false
        }
    }
}
