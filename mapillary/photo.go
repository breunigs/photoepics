package mapillary

import (
	"fmt"
	"strings"
	"time"

	"github.com/paulmach/orb"
)

type Photo struct {
	Key          string
	Sequence     string
	Lon, Lat     float64
	CameraAngle  float32
	MergeCC      int64
	Captured     time.Time
	DistFromPath float64
}

func (p Photo) Point() orb.Point {
	return orb.Point{p.Lon, p.Lat}
}

func (p Photo) RFC3339() string {
	return p.Captured.Format(time.RFC3339)
}

func (p Photo) IRIKey() string {
	k := strings.Replace(p.Key, "-", "ü", -1)
	return strings.Replace(k, "_", "Ö", -1)
}

func (p Photo) DgraphInsert() string {
	k := p.IRIKey()
	return fmt.Sprintf(`
    _:`+k+` <Loc> "{'type':'Point','coordinates':[%f,%f]}"^^<geo:geojson> .
    _:`+k+` <Key> "%s" .
    _:`+k+` <Sequence> "%s" .
    _:`+k+` <CameraAngle> "%f" .
    _:`+k+` <MergeCC> "%d" .
    _:`+k+` <Captured> "%s" .
    _:`+k+` <DistFromPath> "%f" .
  `, p.Lon, p.Lat, p.Key, p.Sequence, p.CameraAngle, p.MergeCC, p.RFC3339(), p.DistFromPath)
}

func PhotoDgraphSchema() string {
	return `
    Key: string @index(exact) .
    Loc: geo @index(geo) .
    Sequence: string .
    CameraAngle: float .
    MergeCC: int .
    Captured: dateTime .
    DistFromPath: float .
  `
}
