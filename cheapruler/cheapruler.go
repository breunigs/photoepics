package cheapruler

import (
	"log"

	cheapruler "github.com/JamesMilnerUK/cheap-ruler-go"
	"github.com/paulmach/orb"
)

var sharedCr cheapruler.CheapRuler
var crInitialized = false

func Init(lat float64) {
	if crInitialized {
		log.Fatalf("Cheapruler was already initialized!")
	}
	crInitialized = true

	cr, err := cheapruler.NewCheapruler(lat, "meters")
	if err != nil {
		log.Fatalf("Failed to initialize Cheapruler: %s", err)
	}
	sharedCr = cr
}

func LineDist(ls orb.LineString, pt orb.Point) float64 {
	if !crInitialized {
		log.Fatalf("Cheapruler not initialized!")
	}

	fls := make([][]float64, len(ls))
	for i, x := range ls {
		fls[i] = toFloat(x)
	}

	pol := sharedCr.PointOnLine(fls, toFloat(pt))
	return sharedCr.Distance(toFloat(pt), pol.Point)
}

func toFloat(pt orb.Point) []float64 {
	return []float64{pt[0], pt[1]}
}
