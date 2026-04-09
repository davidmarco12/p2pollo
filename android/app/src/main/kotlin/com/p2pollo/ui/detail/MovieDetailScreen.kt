package com.p2pollo.ui.detail

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.focusable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.focus.onFocusChanged
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.lifecycle.viewmodel.compose.viewModel
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Text
import com.p2pollo.data.MovieCard
import com.p2pollo.data.TorrentOption
import com.squareup.moshi.Moshi
import com.squareup.moshi.kotlin.reflect.KotlinJsonAdapterFactory
import java.net.URLDecoder

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
        item {
            var focused by remember { mutableStateOf(false) }
            Box(
                modifier = Modifier
                    .onFocusChanged { focused = it.isFocused }
                    .focusable()
                    .clickable(onClick = onBack)
                    .clip(RoundedCornerShape(4.dp))
                    .background(if (focused) Color(0xFF2A2320) else Color.Transparent)
                    .padding(horizontal = 16.dp, vertical = 8.dp)
            ) {
                Text("← Volver", color = Color(0xFFFF6B35), fontSize = 14.sp)
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
                            contentScale = ContentScale.Crop
                        )
                        Column(
                            modifier = Modifier.weight(1f).padding(top = 8.dp)
                        ) {
                            Text(
                                text = d.title,
                                color = Color(0xFFF5F3F0),
                                fontSize = 28.sp
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
                        modifier = Modifier.padding(bottom = 16.dp)
                    )
                }

                items(d.torrents, key = { it.magnet }) { torrent ->
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

// Row sin Card de TV — Box simple con onFocusChanged
// TV Card tiene scale + shadow animation en focus, innecesario y pesado en Amlogic S905
@Composable
fun TorrentRow(
    torrent: TorrentOption,
    onPlay: () -> Unit
) {
    var focused by remember { mutableStateOf(false) }
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(4.dp))
            .onFocusChanged { focused = it.isFocused }
            .focusable()
            .clickable(onClick = onPlay)
            .background(if (focused) Color(0xFF2A2320) else Color(0xFF1A1715))
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
        Box(
            modifier = Modifier
                .clip(RoundedCornerShape(4.dp))
                .background(Color(0xFFFF6B35))
                .padding(horizontal = 16.dp, vertical = 8.dp)
        ) {
            Text("▶ Reproducir", color = Color.White, fontSize = 14.sp)
        }
    }
}
