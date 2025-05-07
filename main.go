package main

import (
	"embed"

	"github.com/atoscerebro/bms-analysis/internal/config"
	"github.com/atoscerebro/bms-analysis/internal/handler"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:internal/client/dist
var assets embed.FS

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	// Create an instance of the app structure
	h := handler.NewHandler(cfg)

	// Create application with options
	err = wails.Run(&options.App{
		Title:  "bms-analysis",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        h.Startup,
		Bind: []interface{}{
			h,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
