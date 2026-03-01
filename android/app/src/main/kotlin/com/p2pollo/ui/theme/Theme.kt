package com.p2pollo.ui.theme

import androidx.compose.runtime.Composable
import androidx.tv.material3.ExperimentalTvMaterial3Api
import androidx.tv.material3.MaterialTheme
import androidx.tv.material3.darkColorScheme
import androidx.compose.ui.graphics.Color

private val P2polloDark = darkColorScheme(
    primary = Color(0xFFFF6B35),
    onPrimary = Color.White,
    secondary = Color(0xFF8A837C),
    background = Color(0xFF0A0908),
    surface = Color(0xFF1A1715),
    onBackground = Color(0xFFF5F3F0),
    onSurface = Color(0xFFF5F3F0)
)

@OptIn(ExperimentalTvMaterial3Api::class)
@Composable
fun P2polloTheme(content: @Composable () -> Unit) {
    MaterialTheme(
        colorScheme = P2polloDark,
        content = content
    )
}
