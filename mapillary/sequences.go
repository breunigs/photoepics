package mapillary

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
	"github.com/paulmach/orb/maptile"
	"github.com/paulmach/orb/planar"
)

type coordinateProperties struct {
	Image_keys []string
	Cas        []float32
}

type sequenceRetriever struct {
	out     chan Photo
	lineStr orb.LineString
	conf    Config
}

func FindSequences(mapConf Config, lineStr orb.LineString) <-chan Photo {
	sr := sequenceRetriever{
		out:     make(chan Photo, 10),
		lineStr: lineStr,
		conf:    mapConf,
	}

	wg := sr.RetrieveTiles()
	go func() {
		wg.Wait()
		close(sr.out)
	}()

	return sr.out
}

func (s sequenceRetriever) RetrieveTiles() *sync.WaitGroup {
	var wg sync.WaitGroup
	tiles := s.Tiles()
	log.Printf("Reading data for %d tiles", len(tiles))
	for _, tile := range tiles {
		wg.Add(1)
		go func(tile maptile.Tile) {
			defer wg.Done()
			s.retrieveTile(tile)
			log.Printf("done reading tile: %dx%d", tile.X, tile.Y)
		}(tile)
	}
	return &wg
}

func (s sequenceRetriever) Tiles() []maptile.Tile {
	tilesMap := make(map[maptile.Tile]struct{})
	for _, pt := range s.lineStr {
		tile := maptile.At(pt, gridZoomLevel)
		tilesMap[tile] = struct{}{}
	}
	tiles := make([]maptile.Tile, 0, len(tilesMap))
	for tile, _ := range tilesMap {
		tiles = append(tiles, tile)
	}
	return tiles
}

func (s sequenceRetriever) retrieveTile(t maptile.Tile) {
	bbox := t.Bound(tileBuffer)
	bboxstr := fmt.Sprintf("%f,%f,%f,%f", bbox.Left(), bbox.Bottom(), bbox.Right(), bbox.Top())
	seqs := getApi(s.conf, "sequences", "per_page=1000&bbox="+bboxstr)

	fc, err := geojson.UnmarshalFeatureCollection([]byte(seqs))
	if err != nil {
		log.Printf("Failed to parse feature collection: %v", err)
		return
	}

	var wg sync.WaitGroup
	for _, feat := range fc.Features {
		g := feat.Geometry
		if g.GeoJSONType() != "LineString" {
			log.Printf("Unexpected Geometry type in Mapillary sequence: %s", g.GeoJSONType())
			continue
		}

		var cp coordinateProperties
		err := mapstructure.Decode(feat.Properties["coordinateProperties"], &cp)
		if err != nil {
			log.Printf("Failed to parse coordinateProperties from Mapillary sequence: %v", err)
			return
		}

		seqkey := fmt.Sprintf("%s", feat.Properties["key"])
		ls := g.(orb.LineString)
		s.makePhotos(seqkey, cp.Image_keys, ls, cp.Cas, &wg)
	}
	wg.Wait()
}

func (s sequenceRetriever) makePhotos(seq string, imgKeys []string, ls orb.LineString, cas []float32, wg *sync.WaitGroup) {
	// Mapillary data is not always consistent
	maxLen := min(len(imgKeys), len(ls), len(cas))

	for i := 0; i < maxLen; i += imageDetailsChunkSize {
		end := i + imageDetailsChunkSize
		if end > maxLen {
			end = maxLen
		}

		wg.Add(1)
		go func(imgKeyChunk []string, lsChunk []orb.Point, casChunk []float32) {
			defer wg.Done()

			detailsChunk := getImageByKeys(s.conf, imgKeyChunk)

			for j := 0; j < len(imgKeyChunk); j++ {
				details := detailsChunk[imgKeyChunk[j]]
				pic := Photo{
					Key:         imgKeyChunk[j],
					CameraAngle: casChunk[j],
					Lon:         lsChunk[j][0],
					Lat:         lsChunk[j][1],
					Captured:    time.Unix(details.CapturedAt.Value/1000, 0),
					MergeCC:     details.MergeCC.Value,
					Sequence:    seq,
				}
				pic.DistFromPath = planar.DistanceFrom(s.lineStr, pic.Point())
				s.out <- pic
			}
		}(imgKeys[i:end], ls[i:end], cas[i:end])
	}
}

func min(x, y, z int) int {
	if x < y && x < z {
		return x
	}
	if y < z {
		return y
	}
	return z
}
