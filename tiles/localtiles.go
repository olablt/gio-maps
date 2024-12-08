package tiles

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"

	"gioui.org/op/paint"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

type LocalTileProvider struct{}

func NewLocalTileProvider() *LocalTileProvider {
	return &LocalTileProvider{}
}

func (p *LocalTileProvider) GetTile(tile Tile) (*paint.ImageOp, error) {
	// Create a new 256x256 RGBA image (standard tile size)
	img := image.NewRGBA(image.Rect(0, 0, 256, 256))

	// Fill with light blue background
	bgColor := color.RGBA{200, 220, 255, 255}
	draw.Draw(img, img.Bounds(), &image.Uniform{bgColor}, image.Point{}, draw.Src)

	// Draw the tile text
	drawText(img, tile)

	// Draw a border around the tile
	borderColor := color.RGBA{100, 100, 100, 255}
	borders := []image.Rectangle{
		image.Rect(0, 0, 256, 1),     // Top
		image.Rect(0, 255, 256, 256), // Bottom
		image.Rect(0, 0, 1, 256),     // Left
		image.Rect(255, 0, 256, 256), // Right
	}
	for _, rect := range borders {
		draw.Draw(img, rect, &image.Uniform{borderColor}, image.Point{}, draw.Src)
	}

	imgOp := paint.NewImageOp(img)
	return &imgOp, nil
}

func drawText(img *image.RGBA, tile Tile) {
	// Create the text to draw
	text := fmt.Sprintf("%d/%d/%d", tile.Zoom, tile.X, tile.Y)

	// Use a font drawer to measure text dimensions
	face := basicfont.Face7x13
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(color.White),
		Face: face,
	}

	// Measure text dimensions
	textWidth := d.MeasureString(text).Round()
	textHeight := face.Metrics().Height.Round()

	// Calculate background rectangle for the text
	padding := 10
	textBgRect := image.Rect(
		(256-textWidth)/2-padding,
		120-textHeight/2-padding,
		(256+textWidth)/2+padding,
		120+textHeight/2+padding,
	)
	// Draw text background
	textBgColor := color.RGBA{255, 255, 255, 220}
	draw.Draw(img, textBgRect, &image.Uniform{textBgColor}, image.Point{}, draw.Over)

	// Set up the position for the text
	d.Dot = fixed.Point26_6{
		X: fixed.I((256 - textWidth) / 2),
		Y: fixed.I(120 + textHeight/2),
	}

	// Draw the text
	d.DrawString(text)
}
