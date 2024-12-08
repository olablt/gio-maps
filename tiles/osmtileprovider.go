package tiles

import (
	"fmt"
	"image"
	_ "image/png"
	"log"
	"net/http"

	"gioui.org/op/paint"
)

type OSMTileProvider struct {
	client *http.Client
}

func NewOSMTileProvider() *OSMTileProvider {
	return &OSMTileProvider{
		client: &http.Client{},
	}
}

func (p *OSMTileProvider) GetTile(tile Tile) (*paint.ImageOp, error) {
	url := p.GetTileURL(tile)
	log.Printf("Requesting OSM tile: %s", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("Error creating request for tile %v: %v", tile, err)
		return nil, err
	}

	// Add browser-like headers
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:109.0) Gecko/20100101 Firefox/115.0")
	req.Header.Set("Accept", "image/webp,*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Referer", "https://www.openstreetmap.org/")

	resp, err := p.client.Do(req)
	if err != nil {
		log.Printf("Error fetching tile %v: %v", tile, err)
		return nil, err
	}
	defer resp.Body.Close()

	log.Printf("OSM tile response status: %s", resp.Status)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	img, _, err := image.Decode(resp.Body)
	if err != nil {
		log.Printf("Error decoding tile image %v: %v", tile, err)
		return nil, err
	}

	log.Printf("Successfully loaded OSM tile: %v", tile)
	imgOp := paint.NewImageOp(img)
	return &imgOp, nil
}

// GetTileURL returns the URL for downloading the map tile
func (p *OSMTileProvider) GetTileURL(tile Tile) string {
	return fmt.Sprintf("https://tile.openstreetmap.org/%d/%d/%d.png",
		tile.Zoom, tile.X, tile.Y)
}
