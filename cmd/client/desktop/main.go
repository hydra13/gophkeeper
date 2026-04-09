package main

import (
	"embed"
	"fmt"
	"os"

	"github.com/hydra13/gophkeeper/cmd/client/common"
	"github.com/hydra13/gophkeeper/cmd/client/desktop/backend"
	"github.com/hydra13/gophkeeper/pkg/buildinfo"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	core, cleanup, err := common.NewCore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	appInfo := backend.AppInfo{
		AppName:       "GophKeeper",
		Version:       buildinfo.Short(),
		ServerAddress: common.DefaultServerAddr(),
		CacheDir:      common.DefaultCacheDir(),
	}

	sessionService := backend.NewSessionService(core, appInfo)
	recordsService := backend.NewRecordsService(core)
	binaryService := backend.NewBinaryService(core)

	app := NewApp(cleanup, binaryService)

	if err := wails.Run(&options.App{
		Title:            "GophKeeper",
		Width:            1360,
		Height:           860,
		MinWidth:         1100,
		MinHeight:        720,
		DisableResize:    false,
		Frameless:        false,
		AssetServer:      &assetserver.Options{Assets: assets},
		BackgroundColour: &options.RGBA{R: 247, G: 248, B: 250, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			sessionService,
			recordsService,
			binaryService,
		},
	}); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
