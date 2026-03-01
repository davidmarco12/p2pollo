package com.p2pollo.ui.home

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.lifecycle.viewmodel.compose.viewModel
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.OutlinedTextFieldDefaults
import androidx.tv.material3.*
import com.p2pollo.data.MovieCard
import com.p2pollo.data.SeriesCard
import com.squareup.moshi.Moshi
import com.squareup.moshi.kotlin.reflect.KotlinJsonAdapterFactory
import kotlinx.coroutines.launch

@OptIn(ExperimentalTvMaterial3Api::class)
@Composable
fun HomeScreen(
    viewModel: HomeViewModel = viewModel(),
    onMovieClick: (String) -> Unit,
    onSeriesClick: (String) -> Unit
) {
    val moshi = remember {
        Moshi.Builder().addLast(KotlinJsonAdapterFactory()).build()
    }

    var selectedTab by remember { mutableIntStateOf(0) }
    val tabs = listOf("Películas", "Series")

    val movies by viewModel.movies.collectAsState()
    val series by viewModel.series.collectAsState()
    val isLoading by viewModel.isLoading.collectAsState()
    val error by viewModel.error.collectAsState()
    val searchQuery by viewModel.searchQuery.collectAsState()

    val scope = rememberCoroutineScope()

    LaunchedEffect(Unit) {
        viewModel.loadPopularMovies()
        viewModel.loadPopularSeries()
    }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .background(Color(0xFF0A0908))
            .padding(horizontal = 48.dp)
    ) {
        // Header
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(top = 32.dp, bottom = 24.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.SpaceBetween
        ) {
            Text(
                text = "p2pollo",
                color = Color(0xFFFF6B35),
                fontSize = 28.sp,
                style = MaterialTheme.typography.headlineMedium
            )

            // Search
            SearchBar(
                query = searchQuery,
                onQueryChange = { q ->
                    viewModel.setSearchQuery(q)
                    scope.launch {
                        if (q.isBlank()) {
                            if (selectedTab == 0) viewModel.loadPopularMovies()
                            else viewModel.loadPopularSeries()
                        } else {
                            if (selectedTab == 0) viewModel.searchMovies(q)
                            else viewModel.searchSeries(q)
                        }
                    }
                }
            )
        }

        // Tabs
        TabRow(
            selectedTabIndex = selectedTab,
            modifier = Modifier.padding(bottom = 24.dp)
        ) {
            tabs.forEachIndexed { index, title ->
                Tab(
                    selected = selectedTab == index,
                    onFocus = {
                        selectedTab = index
                        scope.launch {
                            if (index == 0) viewModel.loadPopularMovies()
                            else viewModel.loadPopularSeries()
                        }
                    },
                    onClick = { selectedTab = index }
                ) {
                    Text(
                        text = title,
                        modifier = Modifier.padding(horizontal = 24.dp, vertical = 12.dp),
                        color = if (selectedTab == index) Color(0xFFFF6B35) else Color(0xFF8A837C)
                    )
                }
            }
        }

        when {
            isLoading -> {
                Box(modifier = Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                    CircularProgressIndicator(color = Color(0xFFFF6B35))
                }
            }
            error != null -> {
                Box(modifier = Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                    Text(
                        text = error ?: "Error desconocido",
                        color = Color.Red,
                        textAlign = TextAlign.Center
                    )
                }
            }
            selectedTab == 0 -> {
                MovieGrid(
                    movies = movies,
                    onMovieClick = { movie ->
                        val adapter = moshi.adapter(MovieCard::class.java)
                        val json = adapter.toJson(movie)
                        onMovieClick(java.net.URLEncoder.encode(json, "UTF-8"))
                    }
                )
            }
            else -> {
                SeriesGrid(
                    seriesList = series,
                    onSeriesClick = { s ->
                        val adapter = moshi.adapter(SeriesCard::class.java)
                        val json = adapter.toJson(s)
                        onSeriesClick(java.net.URLEncoder.encode(json, "UTF-8"))
                    }
                )
            }
        }
    }
}

@OptIn(ExperimentalTvMaterial3Api::class)
@Composable
fun MovieGrid(
    movies: List<MovieCard>,
    onMovieClick: (MovieCard) -> Unit
) {
    LazyVerticalGrid(
        columns = GridCells.Adaptive(minSize = 160.dp),
        horizontalArrangement = Arrangement.spacedBy(16.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp),
        contentPadding = PaddingValues(bottom = 32.dp)
    ) {
        items(movies) { movie ->
            MovieCardItem(movie = movie, onClick = { onMovieClick(movie) })
        }
    }
}

@OptIn(ExperimentalTvMaterial3Api::class)
@Composable
fun SeriesGrid(
    seriesList: List<SeriesCard>,
    onSeriesClick: (SeriesCard) -> Unit
) {
    LazyVerticalGrid(
        columns = GridCells.Adaptive(minSize = 160.dp),
        horizontalArrangement = Arrangement.spacedBy(16.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp),
        contentPadding = PaddingValues(bottom = 32.dp)
    ) {
        items(seriesList) { s ->
            SeriesCardItem(series = s, onClick = { onSeriesClick(s) })
        }
    }
}

@OptIn(ExperimentalTvMaterial3Api::class)
@Composable
fun MovieCardItem(
    movie: MovieCard,
    onClick: () -> Unit
) {
    Card(
        onClick = onClick,
        modifier = Modifier
            .width(160.dp)
            .height(240.dp)
    ) {
        Column(
            modifier = Modifier.fillMaxSize()
        ) {
            coil.compose.AsyncImage(
                model = movie.poster,
                contentDescription = movie.title,
                modifier = Modifier
                    .fillMaxWidth()
                    .height(180.dp),
                contentScale = androidx.compose.ui.layout.ContentScale.Crop
            )
            Column(
                modifier = Modifier
                    .fillMaxSize()
                    .background(Color(0xFF1A1715))
                    .padding(8.dp)
            ) {
                Text(
                    text = movie.title,
                    color = Color(0xFFF5F3F0),
                    fontSize = 12.sp,
                    maxLines = 1,
                    overflow = androidx.compose.ui.text.style.TextOverflow.Ellipsis
                )
                Text(
                    text = "${movie.year} • ★ ${movie.rating}",
                    color = Color(0xFF8A837C),
                    fontSize = 10.sp
                )
            }
        }
    }
}

@OptIn(ExperimentalTvMaterial3Api::class)
@Composable
fun SeriesCardItem(
    series: SeriesCard,
    onClick: () -> Unit
) {
    Card(
        onClick = onClick,
        modifier = Modifier
            .width(160.dp)
            .height(240.dp)
    ) {
        Column(modifier = Modifier.fillMaxSize()) {
            coil.compose.AsyncImage(
                model = series.poster,
                contentDescription = series.title,
                modifier = Modifier
                    .fillMaxWidth()
                    .height(180.dp),
                contentScale = androidx.compose.ui.layout.ContentScale.Crop
            )
            Column(
                modifier = Modifier
                    .fillMaxSize()
                    .background(Color(0xFF1A1715))
                    .padding(8.dp)
            ) {
                Text(
                    text = series.title,
                    color = Color(0xFFF5F3F0),
                    fontSize = 12.sp,
                    maxLines = 1,
                    overflow = androidx.compose.ui.text.style.TextOverflow.Ellipsis
                )
                Text(
                    text = "${series.year} • ★ ${series.rating}",
                    color = Color(0xFF8A837C),
                    fontSize = 10.sp
                )
            }
        }
    }
}

@OptIn(ExperimentalTvMaterial3Api::class)
@Composable
fun SearchBar(
    query: String,
    onQueryChange: (String) -> Unit
) {
    // TV-friendly search
    OutlinedTextField(
        value = query,
        onValueChange = onQueryChange,
        placeholder = { androidx.tv.material3.Text("Buscar...", color = Color(0xFF8A837C)) },
        colors = OutlinedTextFieldDefaults.colors(
            focusedBorderColor = Color(0xFFFF6B35),
            unfocusedBorderColor = Color(0xFF3A3330),
            focusedTextColor = Color(0xFFF5F3F0),
            unfocusedTextColor = Color(0xFFF5F3F0),
            cursorColor = Color(0xFFFF6B35)
        ),
        modifier = Modifier.width(300.dp)
    )
}
