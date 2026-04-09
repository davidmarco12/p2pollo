package torrent

import (
	"path/filepath"
	"strings"
)

// videoExtensions contiene las extensiones de archivos de video soportadas.
var videoExtensions = map[string]bool{
	".mkv": true, ".mp4": true, ".avi": true, ".webm": true,
	".mov": true, ".flv": true, ".wmv": true, ".m4v": true,
	".ts": true, ".mpg": true, ".mpeg": true,
}

// FindMediaFile busca el archivo multimedia más grande en la lista de archivos.
// Retorna el índice del archivo con extensión de video más grande, o 0 si no encuentra ninguno.
func FindMediaFile(files []FileInfo) int {
	bestIdx := 0
	bestSize := int64(0)
	for i, f := range files {
		ext := strings.ToLower(filepath.Ext(f.Path))
		if videoExtensions[ext] && f.Length > bestSize {
			bestIdx = i
			bestSize = f.Length
		}
	}
	return bestIdx
}

// IsVideoFile verifica si un archivo tiene extensión de video.
func IsVideoFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return videoExtensions[ext]
}

// subtitleExtensions contiene las extensiones de archivos de subtítulos soportadas.
var subtitleExtensions = map[string]bool{
	".srt": true, ".sub": true, ".ass": true, ".ssa": true, ".vtt": true,
}

// FindSubtitleFiles retorna los índices de archivos de subtítulos en la lista.
func FindSubtitleFiles(files []FileInfo) []int {
	var indices []int
	for i, f := range files {
		ext := strings.ToLower(filepath.Ext(f.Path))
		if subtitleExtensions[ext] {
			indices = append(indices, i)
		}
	}
	return indices
}
