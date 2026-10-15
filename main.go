//go:build !headless

// Vela Files — Linux desktop entry point (Wails v2 stable).
package main

import (
	"context"
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp(nil)

	err := wails.Run(&options.App{
		Title:            "Vela Files",
		Width:            1180,
		Height:           760,
		MinWidth:         760,
		MinHeight:        480,
		Frameless:        false,
		DisableResize:    false,
		BackgroundColour: &options.RGBA{R: 250, G: 250, B: 250, A: 1},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		DragAndDrop: &options.DragAndDrop{
			// Inbound drops only: the webview accepts OS file drops and hands
			// paths to the Go backend. Outbound drags are delegated to the
			// native GTK helper (see StartOutboundDrag) because WebKit2GTK
			// cannot originate OS-level drags.
			EnableFileDrop:     true,
			DisableWebViewDrop: false,
		},
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "vela-files-single-instance",
		},
		Linux: &linux.Options{
			ProgramName:         "Vela Files",
			WebviewGpuPolicy:    linux.WebviewGpuPolicyOnDemand,
			WindowIsTranslucent: false,
		},
		OnStartup: func(ctx context.Context) {
			app.emit = func(event string, data any) {
				runtime.EventsEmit(ctx, event, data)
			}
			registerFileDrop(ctx, app)
		},
		OnShutdown: func(ctx context.Context) {
			app.Close()
		},
		Bind: []any{app},
	})
	if err != nil {
		log.Fatalf("vela-files: %v", err)
	}
}

// registerFileDrop wires OS file drops into the cancellable copy/move engine.
func registerFileDrop(ctx context.Context, app *App) {
	runtime.OnFileDrop(ctx, func(x, y int, paths []string) {
		if len(paths) == 0 {
			return
		}
		app.emit("drop:received", map[string]any{
			"paths": paths,
			"x":     x,
			"y":     y,
		})
	})
}
