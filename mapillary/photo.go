package mapillary

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/breunigs/photoepics/cheapruler"
	"github.com/breunigs/photoepics/dgraph"
	"github.com/paulmach/orb"
)

type loc struct {
	Type   string    `json:"type,omitempty"`
	Coords []float64 `json:"coordinates,omitempty"`
}

type root struct {
	Photos []Photo `json:"photos"`
}

const photoReadQueryBody = `
  uid
  key
  loc
  sequence
  cameraAngle
  mergeCC
  captured
  distFromPath
`

type Photo struct {
	Uid          string    `json:"uid,omitempty"`
	Key          string    `json:"key,omitempty"`
	Sequence     string    `json:"sequence,omitempty"`
	Loc          loc       `json:"loc,omitempty"`
	CameraAngle  float64   `json:"cameraAngle,omitempty"`
	MergeCC      int64     `json:"mergeCC,omitempty"`
	Captured     time.Time `json:"captured,omitempty"`
	DistFromPath float64   `json:"distFromPath,omitempty"`
}

func (p *Photo) Point() orb.Point {
	return orb.Point{p.Loc.Coords[0], p.Loc.Coords[1]}
}

func (p *Photo) SetLocation(pt orb.Point) {
	p.Loc = loc{
		Type:   "Point",
		Coords: []float64{pt[0], pt[1]},
	}
}

func (p *Photo) Lon() float64 {
	return p.Loc.Coords[0]
}
func (p *Photo) Lat() float64 {
	return p.Loc.Coords[1]
}

func (p *Photo) RFC3339() string {
	return p.Captured.Format(time.RFC3339)
}

func (p *Photo) Transitionable(other Photo) bool {
	return p.MergeCC == other.MergeCC && p.Dist(other) < maxTransitionDistance
}

func (p *Photo) Bearing(other Photo) float64 {
	return cheapruler.Bearing(p.Loc.Coords, other.Loc.Coords)
}

func (p *Photo) Dist(other Photo) float64 {
	return cheapruler.Dist(p.Loc.Coords, other.Loc.Coords)
}

func (p *Photo) IRIKey() string {
	k := strings.Replace(p.Key, "-", "ü", -1)
	return strings.Replace(k, "_", "Ö", -1)
}

func (p *Photo) DgraphInsert() string {
	k := p.IRIKey()
	return fmt.Sprintf(`
    _:`+k+` <loc> "{'type':'Point','coordinates':[%f,%f]}"^^<geo:geojson> .
    _:`+k+` <key> "%s" .
    _:`+k+` <sequence> "%s" .
    _:`+k+` <cameraAngle> "%f" .
    _:`+k+` <mergeCC> "%d" .
    _:`+k+` <captured> "%s" .
    _:`+k+` <distFromPath> "%f" .
  `, p.Loc.Coords[0], p.Loc.Coords[1], p.Key, p.Sequence, p.CameraAngle, p.MergeCC, p.RFC3339(), p.DistFromPath)
}

func PhotoByKey(db dgraph.Wrapper, key string) Photo {
	query := `query PhotoByKey($key: string) {
    photos(func: eq(key, $key)) { ` + photoReadQueryBody + ` }
  }`
	params := map[string]string{
		"$key": key,
	}
	resp := db.Query(query, params)

	var r root
	if err := json.Unmarshal(resp, &r); err != nil {
		log.Fatal(err)
	}

	return r.Photos[0]
}

func PhotosNearQuery(db dgraph.Wrapper, pt orb.Point, radius float64) []Photo {
	query := fmt.Sprintf(`
    query PhotosNear($loc: string, $radius: float) {
      photos(func: near(loc, $loc, $radius) ) { ` + photoReadQueryBody + ` }
    }`)
	params := map[string]string{
		"$loc":    fmt.Sprintf("[%f, %f]", pt[0], pt[1]),
		"$radius": fmt.Sprintf("%f", radius),
	}

	resp := db.Query(query, params)

	var r root
	if err := json.Unmarshal(resp, &r); err != nil {
		log.Fatal(err)
	}

	return r.Photos
}

func PhotoDgraphSchema() string {
	return `
    key: string @index(exact) .
    loc: geo @index(geo) .
    sequence: string .
    cameraAngle: float .
    mergeCC: int .
    captured: dateTime .
    distFromPath: float .
  `
}
