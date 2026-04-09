package com.p2pollo

import android.content.Intent
import android.net.Uri
import android.os.Bundle
import android.view.KeyEvent
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.runtime.Composable
import androidx.navigation.NavType
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.rememberNavController
import androidx.navigation.navArgument
import com.p2pollo.ui.detail.MovieDetailScreen
import com.p2pollo.ui.detail.SeriesDetailScreen
import com.p2pollo.ui.home.HomeScreen
import com.p2pollo.ui.player.PlayerScreen
import com.p2pollo.ui.theme.P2polloTheme

class MainActivity : ComponentActivity() {

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        // Iniciar Foreground Service con el backend Go
        startForegroundService(Intent(this, BackendService::class.java))

        setContent {
            P2polloTheme {
                P2polloApp()
            }
        }
    }

    override fun onDestroy() {
        super.onDestroy()
        stopService(Intent(this, BackendService::class.java))
    }

    // Interceptar TODOS los eventos de teclas antes de que lleguen al árbol de vistas.
    // Necesario porque TextureView (View nativo) roba el foco de Compose y bloquea
    // onPreviewKeyEvent. PlayerScreen escucha este bus para mostrar los controles.
    override fun dispatchKeyEvent(event: KeyEvent): Boolean {
        PlayerKeyEventBus.events.tryEmit(event)
        return super.dispatchKeyEvent(event)
    }
}

@Composable
fun P2polloApp() {
    val navController = rememberNavController()

    NavHost(
        navController = navController,
        startDestination = "home"
    ) {
        composable("home") {
            HomeScreen(
                onMovieClick = { movieJson ->
                    navController.navigate("movie_detail/$movieJson")
                },
                onSeriesClick = { seriesJson ->
                    navController.navigate("series_detail/$seriesJson")
                }
            )
        }

        composable(
            route = "movie_detail/{movieJson}",
            arguments = listOf(navArgument("movieJson") { type = NavType.StringType })
        ) { backStackEntry ->
            val movieJson = backStackEntry.arguments?.getString("movieJson") ?: ""
            MovieDetailScreen(
                movieJson = movieJson,
                onPlayMagnet = { magnet, fileIndex ->
                    val encoded = Uri.encode(magnet)
                    navController.navigate("player/$encoded/$fileIndex")
                },
                onBack = { navController.popBackStack() }
            )
        }

        composable(
            route = "series_detail/{seriesJson}",
            arguments = listOf(navArgument("seriesJson") { type = NavType.StringType })
        ) { backStackEntry ->
            val seriesJson = backStackEntry.arguments?.getString("seriesJson") ?: ""
            SeriesDetailScreen(
                seriesJson = seriesJson,
                onPlayMagnet = { magnet, fileIndex ->
                    val encoded = Uri.encode(magnet)
                    navController.navigate("player/$encoded/$fileIndex")
                },
                onBack = { navController.popBackStack() }
            )
        }

        composable(
            route = "player/{magnet}/{fileIndex}",
            arguments = listOf(
                navArgument("magnet") { type = NavType.StringType },
                navArgument("fileIndex") { type = NavType.IntType }
            )
        ) { backStackEntry ->
            val rawMagnet = backStackEntry.arguments?.getString("magnet") ?: ""
            val magnet = Uri.decode(rawMagnet)
            val fileIndex = backStackEntry.arguments?.getInt("fileIndex") ?: 0
            PlayerScreen(
                magnetUri = magnet,
                fileIndex = fileIndex,
                onBack = {
                    navController.popBackStack()
                }
            )
        }
    }
}
