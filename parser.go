package main

import (
	"io/ioutil"
	"log"

	"github.com/paulmach/orb"
	geojson "github.com/paulmach/orb/geojson"
)

func trackFromFile(filePath string) (orb.LineString, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// TODO: support GPX
	return parseGeoJSON(data)
}

func parseGeoJSON(data []byte) (orb.LineString, error) {
	// TODO: what if the toplevel is not a FeatureCollection?
	fc, err := geojson.UnmarshalFeatureCollection(data)
	if err != nil {
		return nil, err
	}

	ls := orb.LineString{}
	for _, feat := range fc.Features {
		g := feat.Geometry
		switch g.GeoJSONType() {
		case "LineString":
			ls = append(ls, g.(orb.LineString)...)

		case "MultiLineString":
			for _, l := range g.(orb.MultiLineString) {
				ls = append(ls, l...)
			}

		default:
			log.Printf("Unknown GeoJSON type, ignoring: %s\n", g.GeoJSONType())
		}
	}

	return ls, nil
}
