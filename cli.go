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
	"github.com/spf13/cobra"
)

func cmdGen() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gen",
		Short: "Generate photo sequence for the given input file",
		Run: func(cmd *cobra.Command, args []string) {
			lineStr, err := trackFromFile(inputFilePath)
			if err != nil {
				log.Fatalf("Cannot extract GPS track from file: %+v", err)
			}
			cheapruler.Init(lineStr[0][1])

			insertChan := make(chan dgraph.DgraphInsertable, 50)
			go func() {
				defer close(insertChan)
				photoChan := mapillary.FindSequences(mapConf, lineStr)
				for x := range photoChan {
					insertChan <- x
				}
			}()

			db := dgraph.NewClient()
			db.CreateSchema(mapillary.PhotoDgraphSchema())
			db.InsertStream(insertChan)

			startPic := mapillary.PhotoByKey(db, startImageKey)
			endPic := mapillary.PhotoByKey(db, endImageKey)

			db.InsertStream(calcWeightsAlong(db, lineStr, 50))

			resp := db.Query(`
				{
					path as shortest(from: `+startPic.Uid+`, to: `+endPic.Uid+`, numpaths: 10) {
						transitionable @facets(weight)
					}
					path(func: uid(path)) { uid, key }
				}`,
				map[string]string{})
			fmt.Println(string(resp))
		},
	}
	cmd.Flags().StringVarP(&inputFilePath, "input", "i", "", "input file for which to generate a photo sequence")
	cmd.MarkFlagRequired("input")
	requireApiKey(cmd)
	filterByUserName(cmd)
	filterByDate(cmd)

	cmd.Flags().StringVar(&startImageKey, "start-image", "", "The image to start from")
	cmd.MarkFlagRequired("start-image")
	cmd.Flags().StringVar(&endImageKey, "end-image", "", "The image to stop at")
	cmd.MarkFlagRequired("end-image")

	return cmd
}

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
				calcWeights(weightChan, p, c)
			}(prev, curr)
		}
	}()

	return weightChan
}

func calcWeights(weightChan chan<- dgraph.DgraphInsertable, ps1, ps2 []mapillary.Photo) {
	for _, p1 := range ps1 {
		for _, p2 := range ps2 {
			if p1.Uid <= p2.Uid {
				continue
			}

			// consider distance from path
			weight := math.Sqrt(p1.DistFromPath + p2.DistFromPath)

			// bonus if they are transitionable
			if p1.Transitionable(p2) {
				weight -= 5
			}

			// consider distance between the photos themselves, ideal distance at 2m apart.
			// +1.5 to 0 at dist=2, to +8.5 at dist=100
			sqdist := math.Sqrt(p1.Dist(p2))
			if sqdist <= math.Sqrt(2) {
				weight += math.Sqrt(2) - sqdist
			} else {
				weight += sqdist - math.Sqrt(2)
			}

			// viewing in same direction? (+0 to +35)
			var angle float64
			if p1.CameraAngle > p2.CameraAngle {
				angle = p1.CameraAngle - p2.CameraAngle
			} else {
				angle = p2.CameraAngle - p1.CameraAngle
			}
			weight += (angle * angle) / 1000.0

			// order
			malusWrongOrder := 30.0
			bearing1 := p1.Bearing(p2)
			if bearing1 < 0 {
				bearing1 += 360
			}
			bearing2 := bearing1 - 180
			if bearing2 < 0 {
				bearing2 += 360
			}

			if p1.CameraAngle-90 < bearing1 && bearing1 < p1.CameraAngle+90 {
				// fmt.Println("I think p1 before p2")
				weightChan <- edge{from: p1.Uid, to: p2.Uid, weight: weight}
			} else {
				weightChan <- edge{from: p1.Uid, to: p2.Uid, weight: weight + malusWrongOrder}
			}

			if p2.CameraAngle-90 < bearing2 && bearing2 < p2.CameraAngle+90 {
				// fmt.Println("I think p2 before p1")
				weightChan <- edge{from: p2.Uid, to: p1.Uid, weight: weight}
			} else {
				weightChan <- edge{from: p2.Uid, to: p1.Uid, weight: weight + malusWrongOrder}
			}

			// fmt.Printf("b1: %f, b2: %f, key1: %s, key2: %s, ang1: %f, ang2: %f\n", bearing1, bearing2, p1.Key, p2.Key, p1.CameraAngle, p2.CameraAngle)
			// panic(1)
		}
	}
}

func requireApiKey(cmd *cobra.Command) {
	cmd.Flags().StringVar(&mapConf.ApiKey, "api-key", "", "Mapillary API Key")
	cmd.MarkFlagRequired("api-key")
}

func filterByUserName(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&mapConf.FilterUsers, "filter-users", "", "", "only use photos from these Mapillary users. Comma separated.")
}

func filterByDate(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&mapConf.FilterNewer, "filter-newer", "", "", "only use sequences newer than this date. Format YYYY-MM-DD.")
}
