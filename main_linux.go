//go:build linux

package main

import (
	"bytes"
	"context"
	_ "embed"
	"image/png"

	"github.com/netrasad/netrasad/internal/platform/linux/tray"
	"github.com/netrasad/netrasad/internal/platform/windows/singleinstance"
	"github.com/netrasad/netrasad/internal/platform/windows/systray"

	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed build/appicon.png
var appIcon []byte

func configurePlatform(opts *options.App) {
	opts.Linux = &linux.Options{
		Icon: windowIcon(),
		// What Wails picks when no Linux options are given (wails#2977).
		WebviewGpuPolicy: linux.WebviewGpuPolicyNever,
	}
	systray.GetSysTray().SetIcon(appIcon)

	// Not every desktop shows tray icons (GNOME needs an extension), so the
	// window only hides on close while the icon can bring it back; otherwise
	// closing it quits instead of leaving an invisible process behind.
	opts.HideWindowOnClose = false
	opts.OnBeforeClose = func(ctx context.Context) bool {
		if !systray.GetSysTray().Visible() {
			return false
		}
		wailsruntime.WindowHide(ctx)
		return true
	}

	startup := opts.OnStartup
	opts.OnStartup = func(ctx context.Context) {
		singleinstance.OnActivate(func() {
			wailsruntime.WindowShow(ctx)
			wailsruntime.WindowUnminimise(ctx)
		})
		startup(ctx)
	}
}

// windowIcon returns the app icon at 256×256: GTK silently drops X11 window
// icons of more than 262144 pixels, which rules out the 512×512 original.
func windowIcon() []byte {
	img, err := png.Decode(bytes.NewReader(appIcon))
	if err != nil {
		return nil
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, tray.Thumbnail(img, 256)); err != nil {
		return nil
	}
	return buf.Bytes()
}
