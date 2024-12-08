package tiles

import (
	"image"
	"math"
)

const (
	TileSize           = 256
	earthCircumference = 40075016.686 // meters at equator
)

// Tile represents a map tile coordinates
type Tile struct {
	X, Y, Zoom int
}

// LatLng represents a geographical point
type LatLng struct {
	Lat, Lng float64
}

// LatLngToTile converts geographical coordinates to tile coordinates
func LatLngToTile(ll LatLng, zoom int) Tile {
	lat_rad := ll.Lat * math.Pi / 180
	n := math.Pow(2, float64(zoom))
	x := int((ll.Lng + 180.0) / 360.0 * n)
	y := int((1.0 - math.Log(math.Tan(lat_rad)+(1/math.Cos(lat_rad)))/math.Pi) / 2.0 * n)
	return Tile{X: x, Y: y, Zoom: zoom}
}

// TileToLatLng converts tile coordinates to geographical coordinates (returns center of tile)
func TileToLatLng(tile Tile) LatLng {
	n := math.Pow(2, float64(tile.Zoom))
	lon_deg := float64(tile.X)/n*360.0 - 180.0
	lat_rad := math.Atan(math.Sinh(math.Pi * (1 - 2*float64(tile.Y)/n)))
	lat_deg := lat_rad * 180.0 / math.Pi
	return LatLng{Lat: lat_deg, Lng: lon_deg}
}

// CalculateWorldCoordinates converts geographical coordinates to world pixel coordinates at given zoom level
func CalculateWorldCoordinates(ll LatLng, zoom int) (float64, float64) {
	n := math.Pow(2, float64(zoom))
	worldX := float64(TileSize) * n * (ll.Lng + 180) / 360
	worldY := float64(TileSize) * n * (1 - math.Log(math.Tan(ll.Lat*math.Pi/180)+1/math.Cos(ll.Lat*math.Pi/180))/math.Pi) / 2
	return worldX, worldY
}

// WorldToLatLng converts world pixel coordinates back to geographical coordinates
func WorldToLatLng(worldX, worldY float64, zoom int) LatLng {
	n := math.Pow(2, float64(zoom))
	lng := (worldX/(float64(TileSize)*n))*360 - 180
	latRad := math.Pi * (1 - 2*worldY/(float64(TileSize)*n))
	lat := 180 / math.Pi * math.Atan(math.Sinh(latRad))
	return LatLng{Lat: lat, Lng: lng}
}

// CalculateMetersPerPixel calculates the meters per pixel at a given latitude and zoom level
func CalculateMetersPerPixel(latitude float64, zoom int) float64 {
	return earthCircumference * math.Cos(latitude*math.Pi/180) / (math.Pow(2, float64(zoom)) * TileSize)
}

// ConstrainTile ensures tile coordinates are within valid bounds for the zoom level
func ConstrainTile(tile Tile) Tile {
	maxTile := int(math.Pow(2, float64(tile.Zoom))) - 1
	tile.X = max(0, min(tile.X, maxTile))
	tile.Y = max(0, min(tile.Y, maxTile))
	return tile
}

// CalculateVisibleTiles calculates which tiles are visible given a center point and screen size
func CalculateVisibleTiles(center LatLng, zoom int, screenSize image.Point) []Tile {
	centerTile := LatLngToTile(center, zoom)
	tilesX := (screenSize.X / TileSize) + 2 // Add buffer tiles
	tilesY := (screenSize.Y / TileSize) + 2

	startX := centerTile.X - tilesX/2
	startY := centerTile.Y - tilesY/2

	visibleTiles := make([]Tile, 0, tilesX*tilesY)
	for x := startX; x < startX+tilesX; x++ {
		for y := startY; y < startY+tilesY; y++ {
			tile := ConstrainTile(Tile{
				X:    x,
				Y:    y,
				Zoom: zoom,
			})
			visibleTiles = append(visibleTiles, tile)
		}
	}
	return visibleTiles
}
