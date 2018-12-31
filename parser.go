package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/paulmach/orb"
	geojson "github.com/paulmach/orb/geojson"
	"github.com/tkrajina/gpxgo/gpx"
)

func trackFromFile(filePath string, trackId int) (orb.LineString, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	switch getFileEnding(filePath) {
	case "gpx":
		return parseGPX(data, trackId)
	case "geojson":
		return parseGeoJSON(data, trackId)
	default:
		return nil, errors.New("Unknown file extension")
	}
}

func getFileEnding(path string) string {
	path = strings.ToLower(path)
	parts := strings.Split(path, ".")
	return parts[len(parts)-1]
}

func parseGPX(data []byte, trackId int) (orb.LineString, error) {
	gpxFile, err := gpx.ParseBytes(data)
	if err != nil {
		return nil, err
	}

	tracks := make([]orb.LineString, 0, len(gpxFile.Tracks))
	trackDesc := make([]string, 0, len(gpxFile.Tracks))
	for _, track := range gpxFile.Tracks {
		ls := orb.LineString{}
		for _, segment := range track.Segments {
			for _, point := range segment.Points {
				ls = append(ls, orb.Point{point.Longitude, point.Latitude})
			}
		}
		trackDesc = append(trackDesc, track.Name)
		tracks = append(tracks, ls)
	}

	return chooseTrack(tracks, trackDesc, trackId)
}

func parseGeoJSON(data []byte, trackId int) (orb.LineString, error) {
	// TODO: what if the toplevel is not a FeatureCollection?
	fc, err := geojson.UnmarshalFeatureCollection(data)
	if err != nil {
		return nil, err
	}

	tracks := []orb.LineString{}
	trackDesc := []string{}
	for _, feat := range fc.Features {
		g := feat.Geometry
		switch g.GeoJSONType() {
		case "LineString":
			trackDesc = append(trackDesc, "LineString")
			tracks = append(tracks, g.(orb.LineString))

		case "MultiLineString":
			for i, l := range g.(orb.MultiLineString) {
				trackDesc = append(trackDesc, fmt.Sprintf("MultiLineString #%d", i))
				tracks = append(tracks, l)
			}

		default:
			log.Printf("Unknown GeoJSON type, ignoring: %s\n", g.GeoJSONType())
		}
	}

	return chooseTrack(tracks, trackDesc, trackId)
}

func chooseTrack(tracks []orb.LineString, trackDesc []string, trackId int) (orb.LineString, error) {
	if len(tracks) == 1 {
		return tracks[0], nil
	}

	if len(tracks) == 0 {
		return nil, errors.New("The given file does not contain any tracks")
	}

	if trackId >= len(tracks) {
		errMsg := fmt.Sprintf("The given file only contains %d tracks, cannot select track %d", len(tracks), trackId)
		return nil, errors.New(errMsg)
	}

	if trackId >= 0 {
		return tracks[trackId], nil
	}

	chooser := "\nThe file you specified contains multiple tracks. Please choose which should be used:\n"
	for idx, track := range tracks {
		desc := trackDesc[idx] + fmt.Sprintf(" (length: %d)", len(track))
		chooser += fmt.Sprintf("  %2d: %s\n", idx, desc)
	}
	chooser += fmt.Sprintf("\ne.g. %s --track 0", strings.Join(os.Args, " "))
	return nil, errors.New(chooser)
}
