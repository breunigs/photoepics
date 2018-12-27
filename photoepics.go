package main

import (
	"log"

	"github.com/breunigs/photoepics/browser"
	"github.com/breunigs/photoepics/mapillary"
	"github.com/spf13/cobra"
)

var inputFilePath string
var startImageKey string
var endImageKey string

var mapConf mapillary.Config

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
