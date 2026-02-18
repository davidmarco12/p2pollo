package main

import (
	"embed"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	// Guardar el cache de WebView2 en la carpeta del proyecto en lugar de AppData
	execPath, _ := os.Executable()
	webviewDataPath := filepath.Join(filepath.Dir(execPath), ".webview-data")

	err := wails.Run(&options.App{
		Title:  "p2pollo",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			WebviewUserDataPath: webviewDataPath,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
