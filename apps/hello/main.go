package main

import (
	"log"
	"os"

	"github.com/olablt/gio-maps/mapview"
	"github.com/olablt/gio-maps/tiles"

	"gioui.org/app"
	"gioui.org/op"
)

const (
	initialLatitude  = 51.507222 // London
	initialLongitude = -0.1275
	tileSize         = 256
)

func NewMapView(refresh chan struct{}) *mapview.MapView {
	tm := tiles.NewTileManager(
		tiles.NewCombinedTileProvider(
			tiles.NewOSMTileProvider(),
			tiles.NewLocalTileProvider(),
		),
	)
	tm.SetOnLoadCallback(func() {
		// Non-blocking send to refresh channel
		select {
		case refresh <- struct{}{}:
		default:
		}
	})

	return &mapview.MapView{
		TileManager: tm,
		Center:      tiles.LatLng{Lat: initialLatitude, Lng: initialLongitude}, // London
		Zoom:        4,
		MinZoom:     0,
		MaxZoom:     19,
	}
}

func main() {
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	refresh := make(chan struct{}, 1)
	mv := NewMapView(refresh)
	go func() {
		w := new(app.Window)

		var ops op.Ops
		go func() {
			for range refresh {
				w.Invalidate()
			}
		}()
		for {
			switch e := w.Event().(type) {
			case app.DestroyEvent:
				os.Exit(0)
			case app.FrameEvent:
				gtx := app.NewContext(&ops, e)
				mv.Layout(gtx)
				e.Frame(gtx.Ops)
			}
		}
	}()
	app.Main()
}
