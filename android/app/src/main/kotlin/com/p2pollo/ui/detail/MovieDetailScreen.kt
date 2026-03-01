package com.p2pollo.ui.detail

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.lifecycle.viewmodel.compose.viewModel
import androidx.compose.material3.CircularProgressIndicator
import androidx.tv.material3.*
import com.p2pollo.data.MovieCard
import com.p2pollo.data.TorrentOption
import com.squareup.moshi.Moshi
import com.squareup.moshi.kotlin.reflect.KotlinJsonAdapterFactory
import java.net.URLDecoder

@OptIn(ExperimentalTvMaterial3Api::class)
@Composable
fun MovieDetailScreen(
    movieJson: String,
    viewModel: DetailViewModel = viewModel(),
    onPlayMagnet: (String, Int) -> Unit,
    onBack: () -> Unit
) {
    val moshi = remember { Moshi.Builder().addLast(KotlinJsonAdapterFactory()).build() }
    val movie = remember {
        val decoded = URLDecoder.decode(movieJson, "UTF-8")
        moshi.adapter(MovieCard::class.java).fromJson(decoded)
    } ?: return

    val detail by viewModel.movieDetail.collectAsState()
    val isLoading by viewModel.isLoading.collectAsState()
    val error by viewModel.error.collectAsState()

    LaunchedEffect(movie) {
        viewModel.loadMovieDetail(movie)
    }

    val d = detail

    LazyColumn(
        modifier = Modifier
            .fillMaxSize()
            .background(Color(0xFF0A0908)),
        contentPadding = PaddingValues(horizontal = 48.dp, vertical = 48.dp)
    ) {
        // Botón volver
        item {
            Row(
                modifier = Modifier.fillMaxWidth(),
                verticalAlignment = Alignment.CenterVertically
            ) {
                Button(
                    onClick = onBack,
                    colors = ButtonDefaults.colors(containerColor = Color.Transparent)
                ) {
                    Text("← Volver", color = Color(0xFFFF6B35))
                }
            }
        }

        item { Spacer(modifier = Modifier.height(24.dp)) }

        when {
            isLoading && d == null -> item {
                Box(
                    modifier = Modifier.fillMaxWidth().height(400.dp),
                    contentAlignment = Alignment.Center
                ) {
                    CircularProgressIndicator(color = Color(0xFFFF6B35))
                }
            }

            error != null && d == null -> item {
                Text(text = error ?: "Error", color = Color.Red)
            }

            d != null -> {
                item {
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.spacedBy(32.dp)
                    ) {
                        coil.compose.AsyncImage(
                            model = d.poster,
                            contentDescription = d.title,
                            modifier = Modifier.width(200.dp).height(300.dp),
                            contentScale = androidx.compose.ui.layout.ContentScale.Crop
                        )
                        Column(
                            modifier = Modifier.weight(1f).padding(top = 8.dp)
                        ) {
                            Text(
                                text = d.title,
                                color = Color(0xFFF5F3F0),
                                fontSize = 28.sp,
                                style = MaterialTheme.typography.headlineMedium
                            )
                            Spacer(modifier = Modifier.height(8.dp))
                            Text(
                                text = "${d.year} • ★ ${d.rating}",
                                color = Color(0xFFFF6B35),
                                fontSize = 16.sp
                            )
                            Spacer(modifier = Modifier.height(8.dp))
                            Text(
                                text = d.genres,
                                color = Color(0xFF8A837C),
                                fontSize = 14.sp
                            )
                            Spacer(modifier = Modifier.height(16.dp))
                            Text(
                                text = d.description,
                                color = Color(0xFFBBB5AF),
                                fontSize = 14.sp,
                                lineHeight = 22.sp
                            )
                        }
                    }
                }

                item {
                    Spacer(modifier = Modifier.height(32.dp))
                    Text(
                        text = "Torrents disponibles",
                        color = Color(0xFFF5F3F0),
                        fontSize = 20.sp,
                        style = MaterialTheme.typography.titleMedium,
                        modifier = Modifier.padding(bottom = 16.dp)
                    )
                }

                items(d.torrents) { torrent ->
                    TorrentRow(
                        torrent = torrent,
                        onPlay = { onPlayMagnet(torrent.magnet, torrent.fileIndex) }
                    )
                    Spacer(modifier = Modifier.height(8.dp))
                }
            }
        }
    }
}

@OptIn(ExperimentalTvMaterial3Api::class)
@Composable
fun TorrentRow(
    torrent: TorrentOption,
    onPlay: () -> Unit
) {
    Card(
        onClick = onPlay,
        modifier = Modifier.fillMaxWidth()
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .background(Color(0xFF1A1715))
                .padding(horizontal = 24.dp, vertical = 16.dp),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically
        ) {
            Column {
                Text(
                    text = "${torrent.quality} • ${torrent.type}",
                    color = Color(0xFFF5F3F0),
                    fontSize = 16.sp
                )
                Text(
                    text = torrent.size,
                    color = Color(0xFF8A837C),
                    fontSize = 12.sp
                )
            }
            Button(
                onClick = onPlay,
                colors = ButtonDefaults.colors(containerColor = Color(0xFFFF6B35))
            ) {
                Text("▶ Reproducir", color = Color.White)
            }
        }
    }
}
