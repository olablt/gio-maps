package mapview

import (
	"image"
	"log"
	"math"

	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	maps "github.com/olablt/gio-maps/tiles"
)

type MapView struct {
	TileManager *maps.TileManager
	Center      maps.LatLng
	Zoom        int
	MinZoom     int
	MaxZoom     int

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
				worldX, worldY := maps.CalculateWorldCoordinates(mv.Center, mv.Zoom)
				mouseWorldX := worldX + mouseOffsetX
				mouseWorldY := worldY + mouseOffsetY

				// Store old zoom level
				oldZoom := mv.Zoom

				// Update zoom level
				if x.Scroll.Y < 0 {
					mv.setZoom(mv.Zoom + 1)
				} else if x.Scroll.Y > 0 {
					mv.setZoom(mv.Zoom - 1)
				}

				// If zoom changed, adjust center to keep mouse position fixed
				if oldZoom != mv.Zoom {
					// Calculate the new world coordinates after zoom
					zoomFactor := math.Pow(2, float64(mv.Zoom-oldZoom))
					newWorldX := mouseWorldX * zoomFactor
					newWorldY := mouseWorldY * zoomFactor

					// Calculate where the new center should be
					newWorldCenterX := newWorldX - mouseOffsetX
					newWorldCenterY := newWorldY - mouseOffsetY

					// Convert back to geographical coordinates
					mv.Center = maps.WorldToLatLng(newWorldCenterX, newWorldCenterY, mv.Zoom)

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
			lngChange := -float64(deltaX) * mv.metersPerPixel / (111319.9 * math.Cos(mv.Center.Lat*math.Pi/180))

			mv.Center.Lat += latChange
			mv.Center.Lng += lngChange
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
		imageOp, err := mv.TileManager.GetTile(tile)
		if err != nil {
			log.Printf("Error loading tile %v: %v", tile, err)
			continue
		}

		// Calculate Center position in pixels at current zoom level
		centerWorldPx, centerWorldPy := maps.CalculateWorldCoordinates(mv.Center, mv.Zoom)

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
		imageOp.Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)

		transform.Pop()
	}

	return layout.Dimensions{Size: mv.size}
}

func (mv *MapView) setZoom(newZoom int) {
	mv.Zoom = max(mv.MinZoom, min(newZoom, mv.MaxZoom))
	mv.updateVisibleTiles()
}

func (mv *MapView) updateVisibleTiles() {
	mv.metersPerPixel = maps.CalculateMetersPerPixel(mv.Center.Lat, mv.Zoom)
	mv.visibleTiles = maps.CalculateVisibleTiles(mv.Center, mv.Zoom, mv.size)

	// Start loading tiles asynchronously
	for _, tile := range mv.visibleTiles {
		go mv.TileManager.GetTile(tile)
	}
}
