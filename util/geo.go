package util

import (
	"math/rand"
	"time"

	"github.com/kellydunn/golang-geo"
)

// LatLngOffset returns a new pair of coordinates at a given distance in a random direction
func LatLngOffset(lat, lng, distance float64) (float64, float64) {
	rand.Seed(time.Now().Unix())
	newPoint := geo.NewPoint(lat, lng).PointAtDistanceAndBearing(distance, float64(rand.Intn(360)))
	return newPoint.Lat(), newPoint.Lng()
}
