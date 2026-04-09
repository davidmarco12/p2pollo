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
import com.p2pollo.data.EpisodeInfo
import com.p2pollo.data.SeriesCard
import com.p2pollo.data.TorrentOption
import com.squareup.moshi.Moshi
import com.squareup.moshi.kotlin.reflect.KotlinJsonAdapterFactory
import java.net.URLDecoder

@OptIn(ExperimentalTvMaterial3Api::class)
@Composable
fun SeriesDetailScreen(
    seriesJson: String,
    viewModel: DetailViewModel = viewModel(),
    onPlayMagnet: (String, Int) -> Unit,
    onBack: () -> Unit
) {
    val moshi = remember { Moshi.Builder().addLast(KotlinJsonAdapterFactory()).build() }
    val series = remember {
        val decoded = URLDecoder.decode(seriesJson, "UTF-8")
        moshi.adapter(SeriesCard::class.java).fromJson(decoded)
    } ?: return

    val detail by viewModel.seriesDetail.collectAsState()
    val isLoading by viewModel.isLoading.collectAsState()
    val error by viewModel.error.collectAsState()
    val episodeTorrents by viewModel.episodeTorrents.collectAsState()
    val selectedEpisode by viewModel.selectedEpisode.collectAsState()
    var selectedSeason by remember { mutableIntStateOf(1) }

    LaunchedEffect(series) {
        viewModel.loadSeriesDetail(series)
    }

    // Al cargar el detalle, seleccionar la primera temporada real
    val d = detail
    LaunchedEffect(d) {
        val first = d?.seasons?.firstOrNull()?.number
        if (first != null) selectedSeason = first
    }

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
                // Poster + info
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
                        Column(modifier = Modifier.weight(1f).padding(top = 8.dp)) {
                            Text(d.title, color = Color(0xFFF5F3F0), fontSize = 28.sp)
                            Spacer(Modifier.height(8.dp))
                            Text("${d.year} • ★ ${d.rating}", color = Color(0xFFFF6B35), fontSize = 16.sp)
                            Spacer(Modifier.height(8.dp))
                            Text(d.genres, color = Color(0xFF8A837C), fontSize = 14.sp)
                            Spacer(Modifier.height(16.dp))
                            Text(d.description, color = Color(0xFFBBB5AF), fontSize = 14.sp, lineHeight = 22.sp)
                        }
                    }
                }

                item { Spacer(Modifier.height(32.dp)) }

                // Selector de temporadas
                item {
                    Row(
                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                        modifier = Modifier.padding(bottom = 16.dp)
                    ) {
                        d.seasons.forEach { season ->
                            Button(
                                onClick = { selectedSeason = season.number },
                                colors = ButtonDefaults.colors(
                                    containerColor = if (selectedSeason == season.number) Color(0xFFFF6B35)
                                    else Color(0xFF2A2320)
                                )
                            ) {
                                Text("T${season.number}", color = Color.White)
                            }
                        }
                    }
                }

                // Episodios de la temporada seleccionada
                val episodesForSeason = d.seasons
                    .firstOrNull { it.number == selectedSeason }
                    ?.episodes ?: emptyList()

                if (episodesForSeason.isEmpty()) {
                    item {
                        Text(
                            text = "Sin episodios disponibles",
                            color = Color(0xFF8A837C),
                            fontSize = 14.sp,
                            modifier = Modifier.padding(vertical = 16.dp)
                        )
                    }
                } else {
                    items(episodesForSeason) { episode ->
                        EpisodeRow(
                            episode = episode,
                            isLoadingTorrents = isLoading && selectedEpisode == episode,
                            torrents = if (selectedEpisode == episode) episodeTorrents else emptyList(),
                            onEpisodeClick = {
                                viewModel.loadEpisodeTorrents(
                                    title = d.title,
                                    imdbId = d.imdbId,
                                    season = episode.season,
                                    episode = episode.episode,
                                    episodeInfo = episode
                                )
                            },
                            onPlayTorrent = { torrent ->
                                onPlayMagnet(torrent.magnet, torrent.fileIndex)
                            }
                        )
                        Spacer(modifier = Modifier.height(4.dp))
                    }
                }
            }
        }
    }
}

@OptIn(ExperimentalTvMaterial3Api::class)
@Composable
fun EpisodeRow(
    episode: EpisodeInfo,
    isLoadingTorrents: Boolean,
    torrents: List<TorrentOption>,
    onEpisodeClick: () -> Unit,
    onPlayTorrent: (TorrentOption) -> Unit
) {
    Column {
        Card(
            onClick = onEpisodeClick,
            modifier = Modifier.fillMaxWidth()
        ) {
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .background(Color(0xFF1A1715))
                    .padding(horizontal = 24.dp, vertical = 12.dp),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Column {
                    Text(
                        text = "E${episode.episode} — ${episode.title}",
                        color = Color(0xFFF5F3F0),
                        fontSize = 14.sp
                    )
                    if (episode.airDate.isNotBlank()) {
                        Text(
                            text = episode.airDate,
                            color = Color(0xFF8A837C),
                            fontSize = 11.sp
                        )
                    }
                }
                if (isLoadingTorrents) {
                    CircularProgressIndicator(
                        modifier = Modifier.size(20.dp),
                        color = Color(0xFFFF6B35)
                    )
                } else {
                    Text("Torrents", color = Color(0xFFFF6B35), fontSize = 12.sp)
                }
            }
        }

        // Torrents expandidos
        if (torrents.isNotEmpty()) {
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .background(Color(0xFF0F0D0C))
                    .padding(horizontal = 32.dp, vertical = 8.dp),
                verticalArrangement = Arrangement.spacedBy(4.dp)
            ) {
                torrents.forEach { torrent ->
                    TorrentRow(torrent = torrent, onPlay = { onPlayTorrent(torrent) })
                }
            }
        }
    }
}
