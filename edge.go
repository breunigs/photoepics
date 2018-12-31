package main

import (
	"fmt"
	"log"
	"math"
	"sync"

	"github.com/breunigs/photoepics/cheapruler"
	"github.com/breunigs/photoepics/dgraph"
	"github.com/breunigs/photoepics/mapillary"
	"github.com/paulmach/orb"
)

type edge struct {
	from, to string
	weight   float64
}

func (e edge) DgraphInsert() string {
	return fmt.Sprintf("<%s> <transitionable> <%s> (weight=%f) .\n", e.from, e.to, e.weight)
}

func calcWeightsAlong(db dgraph.Wrapper, lineStr orb.LineString, stepSize float64) <-chan dgraph.DgraphInsertable {
	weightChan := make(chan dgraph.DgraphInsertable, 50)
	var wg sync.WaitGroup
	var seen sync.Map
	go func() {
		defer func() {
			wg.Wait()
			close(weightChan)
		}()

		equidist := cheapruler.EveryN(lineStr, stepSize)
		prev := mapillary.PhotosNearQuery(db, <-equidist, stepSize*2)
		for point := range equidist {
			wg.Add(1)
			curr := mapillary.PhotosNearQuery(db, point, stepSize*2)
			go func(p, c []mapillary.Photo) {
				defer wg.Done()
				calcWeights(weightChan, &seen, p, c)
			}(prev, curr)
			prev = curr
		}
	}()

	return weightChan
}

func dupeKey(p1, p2 mapillary.Photo) string {
	if p1.Key > p2.Key {
		return p1.Key + p2.Key
	} else {
		return p2.Key + p1.Key
	}
}

func calcWeights(weightChan chan<- dgraph.DgraphInsertable, seen *sync.Map, ps1, ps2 []mapillary.Photo) {
	for _, p1 := range ps1 {
		for _, p2 := range ps2 {
			if p1.Key == p2.Key {
				continue
			}

			if _, loaded := seen.LoadOrStore(dupeKey(p1, p2), struct{}{}); loaded {
				continue
			}

			// arbitrary number to avoid negative weights
			weight := 15.0

			// consider distance from path
			weight += math.Pow(p1.DistFromPath, 1.5) + math.Pow(p2.DistFromPath, 1.5)

			// bonus if they are transitionable
			if p1.Transitionable(p2) {
				weight -= 10
			}

			// prefer pictures from the same sequence (less loading in MapillaryJS)
			if p1.Sequence == p2.Sequence {
				weight -= 1
			}

			// consider distance between the photos themselves, ideal distance at 2m apart.
			// 0m is +4, 2m +0, 4m +4, 8m +36
			dist := p1.Dist(p2)
			weight += math.Pow(dist-2, 2)

			// viewing in same direction? (+0 to +35)
			var angle float64
			if p1.CameraAngle > p2.CameraAngle {
				angle = p1.CameraAngle - p2.CameraAngle
			} else {
				angle = p2.CameraAngle - p1.CameraAngle
			}
			weight += (angle * angle) / 1000.0

			if weight > 100 {
				continue
			}

			// order
			// malusWrongOrder := 30.0
			bearing1 := p1.Bearing(p2)
			if bearing1 < 0 {
				bearing1 += 360
			}
			bearing2 := bearing1 - 180
			if bearing2 < 0 {
				bearing2 += 360
			}

			if p1.CameraAngle-90 < bearing1 && bearing1 < p1.CameraAngle+90 {
				log.Printf("%s -> %s @ %f\n", p1.Key, p2.Key, weight)
				weightChan <- edge{from: p1.Uid, to: p2.Uid, weight: weight}
			} else {
				// log.Printf("%s -> %s @ %f\n", p1.Key, p2.Key, weight+malusWrongOrder)
				// weightChan <- edge{from: p1.Uid, to: p2.Uid, weight: weight + malusWrongOrder}
			}

			if p2.CameraAngle-90 < bearing2 && bearing2 < p2.CameraAngle+90 {
				log.Printf("%s -> %s # %f\n", p2.Key, p1.Key, weight)
				weightChan <- edge{from: p2.Uid, to: p1.Uid, weight: weight}
			} else {
				// log.Printf("%s -> %s @ %f\n", p2.Key, p1.Key, weight+malusWrongOrder)
				// weightChan <- edge{from: p2.Uid, to: p1.Uid, weight: weight + malusWrongOrder}
			}

			// fmt.Printf("b1: %f, b2: %f, key1: %s, key2: %s, ang1: %f, ang2: %f\n", bearing1, bearing2, p1.Key, p2.Key, p1.CameraAngle, p2.CameraAngle)
			// panic(1)
		}
	}
}
