package main

import (
	"image"
	"log"
	"math"
	"os"

	"github.com/olablt/gio-maps/maps"

	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/pointer"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget"
)

const (
	initialLatitude  = 51.507222 // London
	initialLongitude = -0.1275
	tileSize         = 256
)

type MapView struct {
	tileManager    *maps.TileManager
	center         maps.LatLng
	zoom           int
	minZoom        int
	maxZoom        int
	list           *widget.List
	size           image.Point
	visibleTiles   []maps.Tile
	metersPerPixel float64 // cached calculation
	//
	clickPos    f32.Point
	dragging    bool
	lastDragPos f32.Point
	released    bool
	refresh     chan struct{}
}

func (mv *MapView) Layout(gtx layout.Context) layout.Dimensions {
	tag := mv

	// process events
	dragDelta := f32.Point{}
	for {
		ev, ok := gtx.Event(pointer.Filter{
			Target:  tag,
			Kinds:   pointer.Scroll | pointer.Drag | pointer.Press | pointer.Release | pointer.Cancel,
			ScrollY: pointer.ScrollRange{Min: -10, Max: 10},
		})
		if !ok {
			break
		}

		if x, ok := ev.(pointer.Event); ok {
			// log
			// log.Println("pointer.Event", x)
			switch x.Kind {
			case pointer.Press:
				mv.clickPos = x.Position
				mv.dragging = true
			case pointer.Scroll:
				// Get mouse position relative to screen center
				screenCenterX := float64(mv.size.X >> 1)
				screenCenterY := float64(mv.size.Y >> 1)
				mouseOffsetX := float64(x.Position.X) - screenCenterX
				mouseOffsetY := float64(x.Position.Y) - screenCenterY

				// Convert screen coordinates to world coordinates at current zoom
				worldX, worldY := maps.CalculateWorldCoordinates(mv.center, mv.zoom)
				mouseWorldX := worldX + mouseOffsetX
				mouseWorldY := worldY + mouseOffsetY

				// Store old zoom level
				oldZoom := mv.zoom

				// Update zoom level
				if x.Scroll.Y < 0 {
					mv.setZoom(mv.zoom + 1)
				} else if x.Scroll.Y > 0 {
					mv.setZoom(mv.zoom - 1)
				}

				// If zoom changed, adjust center to keep mouse position fixed
				if oldZoom != mv.zoom {
					// Calculate the new world coordinates after zoom
					zoomFactor := math.Pow(2, float64(mv.zoom-oldZoom))
					newWorldX := mouseWorldX * zoomFactor
					newWorldY := mouseWorldY * zoomFactor

					// Calculate where the new center should be
					newWorldCenterX := newWorldX - mouseOffsetX
					newWorldCenterY := newWorldY - mouseOffsetY

					// Convert back to geographical coordinates
					mv.center = maps.WorldToLatLng(newWorldCenterX, newWorldCenterY, mv.zoom)

					mv.updateVisibleTiles()
				}
			case pointer.Drag:
				dragDelta = x.Position.Sub(mv.clickPos)
				log.Println("pointer.Drag", dragDelta)
			case pointer.Release:
				fallthrough
			case pointer.Cancel:
				mv.dragging = false
				mv.released = true
			}
		}
	}

	if mv.dragging {
		if mv.released {
			mv.lastDragPos = dragDelta
			mv.released = false
		}
		if dragDelta != mv.lastDragPos {
			// Calculate the delta from last position
			deltaX := dragDelta.X - mv.lastDragPos.X
			deltaY := dragDelta.Y - mv.lastDragPos.Y

			// Convert screen movement to geographical coordinates using cached metersPerPixel
			latChange := float64(deltaY) * mv.metersPerPixel / 111319.9
			lngChange := -float64(deltaX) * mv.metersPerPixel / (111319.9 * math.Cos(mv.center.Lat*math.Pi/180))

			mv.center.Lat += latChange
			mv.center.Lng += lngChange
			mv.updateVisibleTiles()
			mv.lastDragPos = dragDelta
		}
	}

	// Update size if changed
	if mv.size != gtx.Constraints.Max {
		mv.size = gtx.Constraints.Max
		mv.updateVisibleTiles()
	}

	// Confine the area of interest to a gtx Max
	defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
	// mv.drag.Add(gtx.Ops)
	// Declare `tag` as being one of the targets.
	event.Op(gtx.Ops, tag)

	// Draw all visible tiles
	for _, tile := range mv.visibleTiles {
		img, err := mv.tileManager.GetTile(tile)
		if err != nil {
			log.Printf("Error loading tile %v: %v", tile, err)
			continue
		}

		// Calculate center position in pixels at current zoom level
		centerWorldPx, centerWorldPy := maps.CalculateWorldCoordinates(mv.center, mv.zoom)

		// Calculate screen center
		screenCenterX := mv.size.X >> 1
		screenCenterY := mv.size.Y >> 1

		// Calculate tile position in pixels
		tileWorldPx := float64(tile.X * maps.TileSize)
		tileWorldPy := float64(tile.Y * maps.TileSize)

		// Calculate final screen position
		finalX := screenCenterX + int(tileWorldPx-centerWorldPx)
		finalY := screenCenterY + int(tileWorldPy-centerWorldPy)

		// Create transform stack and apply offset
		transform := op.Offset(image.Point{X: finalX, Y: finalY}).Push(gtx.Ops)

		// Draw the tile
		imageOp := paint.NewImageOp(img)
		imageOp.Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)

		transform.Pop()
	}

	return layout.Dimensions{Size: mv.size}
}

func NewMapView(refresh chan struct{}) *MapView {
	tm := maps.NewTileManager(
		maps.NewCombinedTileProvider(
			maps.NewOSMTileProvider(),
			maps.NewLocalTileProvider(),
		),
	)
	tm.SetOnLoadCallback(func() {
		// Non-blocking send to refresh channel
		select {
		case refresh <- struct{}{}:
		default:
		}
	})

	return &MapView{
		tileManager: tm,
		center:      maps.LatLng{Lat: initialLatitude, Lng: initialLongitude}, // London
		zoom:        4,
		minZoom:     0,
		maxZoom:     19,
		list: &widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},
	}
}

func (mv *MapView) setZoom(newZoom int) {
	mv.zoom = max(mv.minZoom, min(newZoom, mv.maxZoom))
	mv.updateVisibleTiles()
}

func (mv *MapView) updateVisibleTiles() {
	mv.metersPerPixel = maps.CalculateMetersPerPixel(mv.center.Lat, mv.zoom)
	mv.visibleTiles = maps.CalculateVisibleTiles(mv.center, mv.zoom, mv.size)

	// Start loading tiles asynchronously
	for _, tile := range mv.visibleTiles {
		go mv.tileManager.GetTile(tile)
	}
}

func main() {
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
