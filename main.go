package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"

	"opensca-ui/internal/config"
	"opensca-ui/internal/history"
	"opensca-ui/internal/recent"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	cfg, err := config.Open()
	if err != nil {
		// 极少见：连 %APPDATA%/opensca-ui 都创建不了（比如权限）。
		// 不 panic，前端会显示"未配置"，但用户仍能浏览 UI。
		log.Printf("config.Open failed: %v (settings will not be persisted)", err)
		cfg = &config.Store{}
	}

	rec, err := recent.Open()
	if err != nil {
		log.Printf("recent.Open failed: %v (recent projects will not be persisted)", err)
	}

	hist, err := history.Open()
	if err != nil {
		log.Printf("history.Open failed: %v (task history will not be persisted)", err)
	}

	app := NewApp(cfg, rec, hist)

	err = wails.Run(&options.App{
		Title:     "OpenSCA UI",
		Width:     1280,
		Height:    800,
		MinWidth:  960,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
		// Windows 平台：让 webview 启用拖拽（用户可拖目录到 DropZone）
		Windows: &windows.Options{
			WebviewIsTransparent:              false,
			WindowIsTranslucent:               false,
			DisableWindowIcon:                 false,
			DisableFramelessWindowDecorations: false,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}