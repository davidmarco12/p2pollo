package com.p2pollo.ui.home

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.focus.onFocusChanged
import androidx.compose.foundation.focusable
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.lifecycle.viewmodel.compose.viewModel
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.OutlinedTextFieldDefaults
import androidx.compose.material3.Text
import com.p2pollo.data.MovieCard
import com.p2pollo.data.SeriesCard
import com.squareup.moshi.Moshi
import com.squareup.moshi.kotlin.reflect.KotlinJsonAdapterFactory
import kotlinx.coroutines.launch

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
                fontSize = 28.sp
            )

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

        // Tabs — Box simple sin animaciones (TabRow de TV tiene indicador animado)
        Row(modifier = Modifier.padding(bottom = 24.dp)) {
            tabs.forEachIndexed { index, title ->
                var focused by remember { mutableStateOf(false) }
                val active = selectedTab == index
                Box(
                    modifier = Modifier
                        .onFocusChanged { focused = it.isFocused }
                        .focusable()
                        .clickable {
                            selectedTab = index
                            scope.launch {
                                if (index == 0) viewModel.loadPopularMovies()
                                else viewModel.loadPopularSeries()
                            }
                        }
                        .clip(RoundedCornerShape(4.dp))
                        .background(
                            when {
                                active -> Color(0xFF2A1F1A)
                                focused -> Color(0xFF1F1C1A)
                                else -> Color.Transparent
                            }
                        )
                        .padding(horizontal = 24.dp, vertical = 12.dp)
                ) {
                    Text(
                        text = title,
                        color = if (active) Color(0xFFFF6B35) else Color(0xFF8A837C),
                        fontSize = 14.sp
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

@Composable
fun MovieGrid(
    movies: List<MovieCard>,
    onMovieClick: (MovieCard) -> Unit
) {
    LazyVerticalGrid(
        columns = GridCells.Fixed(5),
        horizontalArrangement = Arrangement.spacedBy(16.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp),
        contentPadding = PaddingValues(bottom = 32.dp)
    ) {
        items(movies, key = { it.title + it.year }) { movie ->
            MovieCardItem(movie = movie, onClick = { onMovieClick(movie) })
        }
    }
}

@Composable
fun SeriesGrid(
    seriesList: List<SeriesCard>,
    onSeriesClick: (SeriesCard) -> Unit
) {
    LazyVerticalGrid(
        columns = GridCells.Fixed(5),
        horizontalArrangement = Arrangement.spacedBy(16.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp),
        contentPadding = PaddingValues(bottom = 32.dp)
    ) {
        items(seriesList, key = { it.title + it.year }) { s ->
            SeriesCardItem(series = s, onClick = { onSeriesClick(s) })
        }
    }
}

// Card sin animaciones de TV — Box simple con borde en focus.
// TV Material3 Card tiene scale + shadow animation en focus, muy pesado en Mali (Amlogic S905).
// Aquí solo cambia el color de fondo del info panel: sin layout changes, sin animaciones.
@Composable
fun MovieCardItem(
    movie: MovieCard,
    onClick: () -> Unit
) {
    var focused by remember { mutableStateOf(false) }
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .height(240.dp)
            .onFocusChanged { focused = it.isFocused }
            .focusable()
            .clickable(onClick = onClick)
            .clip(RoundedCornerShape(4.dp))
            .background(if (focused) Color(0xFF2A1F1A) else Color(0xFF1A1715))
    ) {
        coil.compose.AsyncImage(
            model = movie.poster,
            contentDescription = movie.title,
            modifier = Modifier
                .fillMaxWidth()
                .height(180.dp),
            contentScale = ContentScale.Crop
        )
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(8.dp)
        ) {
            Text(
                text = movie.title,
                color = if (focused) Color(0xFFFF6B35) else Color(0xFFF5F3F0),
                fontSize = 12.sp,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis
            )
            Text(
                text = "${movie.year} • ★ ${movie.rating}",
                color = Color(0xFF8A837C),
                fontSize = 10.sp
            )
        }
    }
}

@Composable
fun SeriesCardItem(
    series: SeriesCard,
    onClick: () -> Unit
) {
    var focused by remember { mutableStateOf(false) }
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .height(240.dp)
            .onFocusChanged { focused = it.isFocused }
            .focusable()
            .clickable(onClick = onClick)
            .clip(RoundedCornerShape(4.dp))
            .background(if (focused) Color(0xFF2A1F1A) else Color(0xFF1A1715))
    ) {
        coil.compose.AsyncImage(
            model = series.poster,
            contentDescription = series.title,
            modifier = Modifier
                .fillMaxWidth()
                .height(180.dp),
            contentScale = ContentScale.Crop
        )
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(8.dp)
        ) {
            Text(
                text = series.title,
                color = if (focused) Color(0xFFFF6B35) else Color(0xFFF5F3F0),
                fontSize = 12.sp,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis
            )
            Text(
                text = "${series.year} • ★ ${series.rating}",
                color = Color(0xFF8A837C),
                fontSize = 10.sp
            )
        }
    }
}

@Composable
fun SearchBar(
    query: String,
    onQueryChange: (String) -> Unit
) {
    OutlinedTextField(
        value = query,
        onValueChange = onQueryChange,
        placeholder = { Text("Buscar...", color = Color(0xFF8A837C)) },
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
