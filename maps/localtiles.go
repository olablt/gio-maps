package maps

import (
    "image"
    "image/color"
    "image/draw"
    "fmt"
    "golang.org/x/image/font"
    "golang.org/x/image/font/basicfont"
    "golang.org/x/image/math/fixed"
)

type LocalTileProvider struct{}

func NewLocalTileProvider() *LocalTileProvider {
    return &LocalTileProvider{}
}

func (p *LocalTileProvider) GetTile(tile Tile) (image.Image, error) {
    // Create a new 256x256 RGBA image (standard tile size)
    img := image.NewRGBA(image.Rect(0, 0, 256, 256))

    // Fill with light gray background
    draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{240, 240, 240, 255}}, image.Point{}, draw.Src)

    // Create the text to draw
    text := fmt.Sprintf("%d/%d/%d", tile.Zoom, tile.X, tile.Y)

    // Set up the font drawer
    d := &font.Drawer{
        Dst:  img,
        Src:  image.NewUniform(color.RGBA{0, 0, 0, 255}),
        Face: basicfont.Face7x13,
        Dot:  fixed.Point26_6{},
    }

    // Calculate text width to center it
    textWidth := d.MeasureString(text).Round()
    
    // Position text in center of tile
    x := (256 - textWidth) / 2
    y := 256 / 2
    
    d.Dot = fixed.P(x, y)
    d.DrawString(text)

    return img, nil
}
