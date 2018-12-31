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

func Dist(p1, p2 []float64) float64 {
	return sharedCr.Distance(p1, p2)
}

func Bearing(p1, p2 []float64) float64 {
	return sharedCr.Bearing(p1, p2)
}

func LineDist(ls orb.LineString, pt orb.Point) float64 {
	if !crInitialized {
		log.Fatalf("Cheapruler not initialized!")
	}

	fls := toFloatLs(ls)

	pol := sharedCr.PointOnLine(fls, toFloat(pt))
	return sharedCr.Distance(toFloat(pt), pol.Point)
}

// emits a Point every interval <unit of sharedCr> along the line string
func EveryN(ls orb.LineString, interval float64) []orb.Point {
	if interval <= 0 {
		log.Fatalf("interval must be positive")
	}

	out := make([]orb.Point, 0)
	out = append(out, ls[0])

	step := 0.0
	currentPos := 0.0

	for i := 0; i < len(ls)-1; {
		p0 := toFloat(ls[i])
		p1 := toFloat(ls[i+1])
		d := sharedCr.Distance(p0, p1)

		if currentPos+d < interval*step {
			currentPos += d
			i++
			continue
		}

		out = append(out, interpolate(p0, p1, (interval*step-currentPos)/d))
		step++
	}

	out = append(out, ls[len(ls)-1])
	return out
}

func toFloat(pt orb.Point) []float64 {
	return []float64{pt[0], pt[1]}
}

func toFloatLs(ls orb.LineString) [][]float64 {
	fls := make([][]float64, len(ls))
	for i, x := range ls {
		fls[i] = toFloat(x)
	}
	return fls
}

func interpolate(a []float64, b []float64, t float64) orb.Point {
	dx := b[0] - a[0]
	dy := b[1] - a[1]
	return orb.Point{
		a[0] + dx*t,
		a[1] + dy*t,
	}
}
