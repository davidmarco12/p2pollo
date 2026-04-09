package server

import "strings"

// extractQuality extrae la calidad del nombre del torrent
func extractQuality(name string) string {
	name = strings.ToUpper(name)

	if strings.Contains(name, "2160P") || strings.Contains(name, "4K") || strings.Contains(name, "UHD") {
		return "2160p"
	}
	if strings.Contains(name, "1080P") {
		return "1080p"
	}
	if strings.Contains(name, "720P") {
		return "720p"
	}
	if strings.Contains(name, "480P") {
		return "480p"
	}

	return "Unknown"
}

// extractType extrae el tipo de release del nombre del torrent
func extractType(name string) string {
	name = strings.ToUpper(name)

	if strings.Contains(name, "BLURAY") || strings.Contains(name, "BLU-RAY") || strings.Contains(name, "BDRIP") {
		return "bluray"
	}
	if strings.Contains(name, "WEBRIP") || strings.Contains(name, "WEB-DL") || strings.Contains(name, "WEBDL") {
		return "web"
	}
	if strings.Contains(name, "DVDRIP") || strings.Contains(name, "DVD") {
		return "dvd"
	}
	if strings.Contains(name, "HDTV") {
		return "hdtv"
	}
	if strings.Contains(name, "CAM") || strings.Contains(name, "TS") || strings.Contains(name, "TELESYNC") {
		return "cam"
	}

	return "other"
}
