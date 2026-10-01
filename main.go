package main

import (
	"context"
	"os"

	"github.com/netrasad/netrasad/app"
	"github.com/netrasad/netrasad/frontend"
	"github.com/netrasad/netrasad/internal/platform/windows/singleinstance"
	"github.com/netrasad/netrasad/internal/platform/windows/systray"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

func main() {
	if singleinstance.CheckAndLock() {
		os.Exit(0)
	}

	a := app.New()

	err := wails.Run(&options.App{
		Title:             "NetRasad",
		Width:             1024,
		Height:            700,
		HideWindowOnClose: true,
		AssetServer: &assetserver.Options{
			Assets: frontend.Assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 31, B: 38, A: 1},
		OnStartup: func(ctx context.Context) {
			a.SetEmitEvent(func(name string, data interface{}) {
				wailsruntime.EventsEmit(ctx, name, data)
			})
			a.Startup(ctx)

			_ = systray.GetSysTray().Start(
				func() {
					wailsruntime.WindowShow(ctx)
					wailsruntime.WindowUnminimise(ctx)
				},
				func() {
					a.SetTaskbarWidgetEnabled(!a.GetTaskbarWidgetEnabled())
				},
				func() {
					systray.GetSysTray().Stop()
					wailsruntime.Quit(ctx)
				},
			)
		},
		OnShutdown: func(ctx context.Context) {
			systray.GetSysTray().Stop()
			a.Shutdown(ctx)
		},
		Bind: []interface{}{
			a,
		},
	})
	if err != nil {
		panic(err)
	}
}
