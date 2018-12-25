package main

import (
	"log"

	"github.com/breunigs/photoepics/browser"
	"github.com/spf13/cobra"
)

// zoom level at which the bbox are aligned (using OSM tile boundaries)
const gridZoomLevel = 14

var inputFilePath string
var mapillaryClientKey string
var mapillaryFilterUsers string
var mapillaryFilterNewer string

var rootCmd = &cobra.Command{
	Use:   "photoepics",
	Short: "convert GPX into Mapillary photo sequences",
	Long:  "Photoepics takes a GeoJSON file as input and tries to find matching sequences of photos from Mapillary.",
}

func main() {
	rootCmd.AddCommand(cmdGen())
	rootCmd.Execute()
}

func doStuff() {
	log.Println(browser.Get("https://a.mapillary.com/v3/images?client_id="))

}
