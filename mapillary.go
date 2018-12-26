package main

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/breunigs/photoepics/browser"
	"github.com/mitchellh/mapstructure"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
	"github.com/paulmach/orb/maptile"
)

const mapillaryBaseUrl = "https://a.mapillary.com/v3/"
const tileBuffer = 0.05           // i.e. 5% border around tile
const imageDetailsChunkSize = 100 // how many image details to fetch from Mapillary's private API per request

type Photo struct {
	Key         string
	Sequence    string
	Lon, Lat    float64
	CameraAngle float32
	MergeCC     int64
	Captured    time.Time
}

type coordinateProperties struct {
	Image_keys []string
	Cas        []float32
}

type jsonGraphInt64 struct {
	Type  string `json:"$type"`
	Value int64  `json:"value"`
}

type imageByKey struct {
	CapturedAt jsonGraphInt64 `json:"captured_at"`
	MergeCC    jsonGraphInt64 `json:"merge_cc"`
}

type graphImageByKey struct {
	JsonGraph struct {
		ImageByKey map[string]imageByKey `json:"imageByKey"`
	} `json:"jsonGraph"`
}

func FindSequences(t maptile.Tile) {
	bbox := t.Bound(tileBuffer)
	bboxstr := fmt.Sprintf("%f,%f,%f,%f", bbox.Left(), bbox.Bottom(), bbox.Right(), bbox.Top())
	seqs := getApi("sequences", "per_page=1000&bbox="+bboxstr)

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
		makePhotos(seqkey, cp.Image_keys, ls, cp.Cas, &wg)
	}
	wg.Wait()
}

func makePhotos(seq string, imgKeys []string, ls orb.LineString, cas []float32, wg *sync.WaitGroup) {
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

			detailsChunk := getImageByKeys(imgKeyChunk)

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
				fmt.Printf("%+v\n", pic)
			}
		}(imgKeys[i:end], ls[i:end], cas[i:end])
	}
}

func getApi(fun string, query string) string {
	url := mapillaryBaseUrl + fun
	url += "?client_id=" + mapillaryClientKey
	url += maybeFilterUsers()
	url += maybeFilterNewer()
	if query != "" {
		url += "&" + query
	}
	body, err := browser.Get(url)
	if err != nil {
		log.Fatalf("Failed to read from Mapillary: %+v", err)
	}
	return body
}

func getImageByKeys(imageKey []string) map[string]imageByKey {
	imgKeys := strings.Join(imageKey, `","`)

	url := mapillaryBaseUrl + "model.json"
	url += "?client_id=" + mapillaryClientKey
	url += "&method=get"
	url += fmt.Sprintf(`&paths=[["imageByKey",["%s"],["captured_at","merge_cc"]]]`, imgKeys)
	body, err := browser.Get(url)
	if err != nil {
		log.Fatalf("Failed to read from Mapillary: %+v", err)
	}

	res := graphImageByKey{}
	err = json.Unmarshal([]byte(body), &res)
	if err != nil {
		log.Fatalf("Unexpected output for imageByKey: %+v", err)
	}
	return res.JsonGraph.ImageByKey
}

func maybeFilterUsers() string {
	if mapillaryFilterUsers == "" {
		return ""
	}

	matched, err := regexp.MatchString("^[a-zA-Z0-9_,-]+$", mapillaryFilterUsers)
	if err != nil {
		log.Fatalf("Failed to parse user filter: %+v", err)
	}
	if !matched {
		log.Fatalf("Failed to parse user filter. Only alphanumeric characters, underscores and dashes are allowed. Usernames should be separated by comma. Got: %s", mapillaryFilterUsers)
	}

	// spaced := strings.Replace(mapillaryFilterUsers, ",", ", ", -1)
	// log.Printf("photos by these users: %s", spaced)
	return "&usernames=" + mapillaryFilterUsers
}

func maybeFilterNewer() string {
	if mapillaryFilterNewer == "" {
		return ""
	}

	parsed, err := time.Parse("2006-01-02", mapillaryFilterNewer)
	if err != nil {
		log.Fatalf("Failed to parse date filter: %+v", err)
	}

	// log.Printf("photos newer than: %s", parsed.Format("2006-01-02"))
	return "&start_time=" + parsed.Format("2006-01-02")
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
