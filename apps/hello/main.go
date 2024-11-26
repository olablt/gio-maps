package main

import (
	"gio-maps/maps"
	"image"
	"io"
	"log"
	"math"
	"os"
	"strings"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/widget"
)

type MapView struct {
	tileManager  *maps.TileManager
	center       maps.LatLng
	zoom         int
	list         *widget.List
	size         image.Point
	visibleTiles []maps.Tile
	drag         widget.Draggable
	lastDragPos  image.Point
}

func NewMapView() *MapView {
	return &MapView{
		// tileManager: maps.NewTileManager(maps.NewLocalTileProvider()), // Use local provider
		tileManager: maps.NewTileManager(maps.NewOSMTileProvider()), // Use OSM provider
		center:      maps.LatLng{Lat: 51.507222, Lng: -0.1275},      // London
		zoom:        4,
		list: &widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},
	}
}

func (mv *MapView) calculateVisibleTiles() {
	// Calculate center tile
	centerTile := maps.LatLngToTile(mv.center, mv.zoom)

	// Calculate how many tiles we need in each direction based on window size
	tilesX := (mv.size.X / 256) + 2 // Add buffer tiles
	tilesY := (mv.size.Y / 256) + 2

	startX := centerTile.X - tilesX/2
	startY := centerTile.Y - tilesY/2

	mv.visibleTiles = make([]maps.Tile, 0, tilesX*tilesY)

	for x := startX; x < startX+tilesX; x++ {
		for y := startY; y < startY+tilesY; y++ {
			mv.visibleTiles = append(mv.visibleTiles, maps.Tile{
				X:    x,
				Y:    y,
				Zoom: mv.zoom,
			})
		}
	}

	// Start loading tiles asynchronously
	for _, tile := range mv.visibleTiles {
		go mv.tileManager.GetTile(tile)
	}
}

func (mv *MapView) Layout(gtx layout.Context) layout.Dimensions {
	// Update size if changed
	if mv.size != gtx.Constraints.Max {
		mv.size = gtx.Constraints.Max
		mv.calculateVisibleTiles()
	}

	if mv.drag.Dragging() {
		log.Println("Dragging")
		// pos drag position relative to its initial click position
		pos := mv.drag.Pos()
		// convert screen movement to geographical coordinates
		// the conversion factor depends on the zoom level
		// at zoom level z, one pixel represents roughly 156543.03392 * cos(lat) / 2^z meters
		metersPerPixel := 156543.03392 * math.Cos(mv.center.Lat) / math.Pow(2, float64(mv.zoom))
		// convert pixel movement to degrees
		// 111319.9 is the number of meters per degree at the equator
		latChange := -float64(pos.Y) * metersPerPixel / 111319.9
		lngChange := -float64(pos.X) * metersPerPixel / (111319.9 * math.Cos(mv.center.Lat))
		mv.center.Lat += latChange
		mv.center.Lng += lngChange
		mv.calculateVisibleTiles()
	}

	// // Handle drag events
	// for _, e := range mv.drag.Events(gtx.Metric, gtx, gesture.Both) {
	// 	switch e.Type {
	// 	case pointer.Drag:
	// 		// Calculate the change in position
	// 		delta := e.Position.Sub(mv.lastDragPos)
	// 		mv.lastDragPos = e.Position
	// 		// Convert screen movement to geographical coordinates
	// 		// The conversion factor depends on the zoom level
	// 		// At zoom level z, one pixel represents roughly 156543.03392 * cos(lat) / 2^z meters
	// 		metersPerPixel := 156543.03392 * math.Cos(mv.center.Lat*math.Pi/180) / math.Pow(2, float64(mv.zoom))
	// 		// Convert pixel movement to degrees
	// 		// 111319.9 is the number of meters per degree at the equator
	// 		latChange := -delta.Y * metersPerPixel / 111319.9
	// 		lngChange := -delta.X * metersPerPixel / (111319.9 * math.Cos(mv.center.Lat*math.Pi/180))
	// 		mv.center.Lat += latChange
	// 		mv.center.Lng += lngChange
	// 		mv.calculateVisibleTiles()
	// 	case pointer.Press:
	// 		mv.lastDragPos = e.Position
	// 	}
	// }
	// // Add the drag input area
	// mv.drag.Add(gtx.Ops)

	ops := gtx.Ops

	// Draw all visible tiles
	for _, tile := range mv.visibleTiles {
		img, err := mv.tileManager.GetTile(tile)
		if err != nil {
			log.Printf("Error loading tile %v: %v", tile, err)
			continue
		}

		// Calculate position for this tile relative to center
		centerTile := maps.LatLngToTile(mv.center, mv.zoom)
		offsetX := (tile.X - centerTile.X) * 256
		offsetY := (tile.Y - centerTile.Y) * 256

		// Center the view in the window
		screenCenterX := mv.size.X / 2
		screenCenterY := mv.size.Y / 2
		finalX := screenCenterX + offsetX - 128 // 128 is half tile size
		finalY := screenCenterY + offsetY - 128

		// Create transform stack and apply offset
		transform := op.Offset(image.Point{X: finalX, Y: finalY}).Push(ops)

		// Draw the tile
		imageOp := paint.NewImageOp(img)
		imageOp.Add(ops)
		paint.PaintOp{}.Add(ops)

		transform.Pop()
	}

	w := func(gtx layout.Context) layout.Dimensions {
		// sz := image.Pt(10, 10) // drag area
		sz := gtx.Constraints.Max
		return layout.Dimensions{Size: sz}
	}
	mv.drag.Layout(gtx, w, w)
	// drag must respond with an Offer event when requested.
	// Use the drag method for this.
	if m, ok := mv.drag.Update(gtx); ok {
		mv.drag.Offer(gtx, m, io.NopCloser(strings.NewReader("hello world")))
	}
	// mv.drag.Layout(gtx, func(gtx layout.Context, index int) layout.Dimensions {
	// 	return layout.Dimensions{Size: image.Point{X: 256, Y: 256}}
	// })

	return layout.Dimensions{Size: mv.size}
}

func main() {
	go func() {
		w := new(app.Window)

		mv := NewMapView()

		var ops op.Ops
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
