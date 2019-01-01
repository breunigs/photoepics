package edge

import (
	"fmt"
	"log"
	"math"
	"runtime"
	"sync"
	"time"

	"github.com/breunigs/photoepics/cheapruler"
	"github.com/breunigs/photoepics/dgraph"
	"github.com/breunigs/photoepics/mapillary"
	"github.com/paulmach/orb"
	pb "gopkg.in/cheggaaa/pb.v1"
)

const month = 30 * 24 * time.Hour

type edge struct {
	from, to string
	weight   float64
}

func (e edge) DgraphInsert() string {
	return fmt.Sprintf("<%s> <transitionable> <%s> (weight=%f) .\n", e.from, e.to, e.weight)
}

func CalcWeightsAlong(db dgraph.Wrapper, lineStr orb.LineString, stepSize float64) <-chan dgraph.DgraphInsertable {
	weightChan := make(chan dgraph.DgraphInsertable, 50)
	var wg sync.WaitGroup
	var seen sync.Map
	go func() {
		defer func() {
			wg.Wait()
			close(weightChan)
		}()

		equidist := cheapruler.EveryN(lineStr, stepSize)

		log.Println("Calculating weights for close images…")
		bar := pb.StartNew(len(equidist) - 1)
		picPairChan := findNearbyImages(db, equidist, stepSize*2)

		for picPair := range picPairChan {
			calcWeights(weightChan, &seen, picPair[0], picPair[1])
			bar.Increment()
		}

		bar.Finish()
	}()

	return weightChan
}

func Count(db dgraph.Wrapper) int64 {
	return db.Count("transitionable")
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

			// arbitrary number to avoid negative weights
			weight := 15.0

			// consider distance from path, but ignore differences below 2m
			avgDist := (p1.DistFromPath + p2.DistFromPath) / 2
			weight += math.Max(0, math.Pow(avgDist, 1.5)-2)

			// bonus if they are transitionable
			if p1.Transitionable(p2) {
				weight -= 10
			}

			// prefer pictures from the same sequence (less loading in MapillaryJS)
			if p1.Sequence == p2.Sequence {
				weight -= 1
			}

			// consider distance between the photos themselves, ideal distance at 2m apart.
			dist := p1.Dist(p2)
			// 1m is +4, 3m +0, 5m +4, 9m +36
			// weight += math.Pow(dist-3, 2)

			// 0m is +4, 2m +0, 4m +4, 8m +36
			weight += math.Pow(dist-2, 2)
			// discourage close pics more strongly
			if dist < 1 {
				weight += 2
			}

			// viewing in same direction? (+0 to +35)
			var angle float64
			if p1.CameraAngle > p2.CameraAngle {
				angle = p1.CameraAngle - p2.CameraAngle
			} else {
				angle = p2.CameraAngle - p1.CameraAngle
			}
			weight += (angle * angle) / 1000.0

			if weight > 250 {
				continue
			}

			if _, loaded := seen.LoadOrStore(dupeKey(p1, p2), struct{}{}); loaded {
				continue
			}

			// prefer newer pictures. Since this is unbounded, do it after pruning
			age1 := time.Now().Sub(p1.Captured)
			age2 := time.Now().Sub(p2.Captured)
			weight += 3*float64(age1/month) + 3*float64(age2/month)

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

			if p1.AngleWithin(bearing1, 45) {
				weightChan <- edge{from: p1.Uid, to: p2.Uid, weight: weight}
			} else if p1.AngleWithin(bearing1, 90) {
				weightChan <- edge{from: p1.Uid, to: p2.Uid, weight: weight + 5}
			}

			if p2.AngleWithin(bearing2, 45) {
				weightChan <- edge{from: p2.Uid, to: p1.Uid, weight: weight}
			} else if p2.AngleWithin(bearing2, 90) {
				weightChan <- edge{from: p2.Uid, to: p1.Uid, weight: weight + 5}
			}
		}
	}
}

func findNearbyImages(db dgraph.Wrapper, pts []orb.Point, radius float64) <-chan [2][]mapillary.Photo {
	cache := make([][]mapillary.Photo, len(pts))
	var mu sync.Mutex

	jobs := make(chan int, len(pts))
	done := make(chan int, len(pts))
	for w := 0; w < runtime.NumCPU()-1; w++ {
		go func(jobs <-chan int, done chan<- int) {
			for j := range jobs {
				nearby := mapillary.PhotosNearQuery(db, pts[j], radius)
				mu.Lock()
				cache[j] = nearby
				mu.Unlock()
				done <- j
			}
		}(jobs, done)
	}

	for i := 0; i < len(pts); i++ {
		jobs <- i
	}
	close(jobs)

	groupChan := make(chan [2][]mapillary.Photo, 1)
	go func() {
		startFrom := 0
		status := make([]bool, len(pts))
		for c := 0; c < len(pts); c++ {
			status[<-done] = true
			// everytime a new query is done, try to emit the next pair
			for i := startFrom; i < len(pts)-1; i++ {
				if !status[i] || !status[i+1] {
					break
				}

				mu.Lock()
				groupChan <- [2][]mapillary.Photo{cache[i], cache[i+1]}
				cache[i] = nil
				mu.Unlock()
				startFrom = i + 1
			}
		}
		close(done)
		close(groupChan)
	}()

	return groupChan
}
