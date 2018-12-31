package main

import (
	"log"

	"github.com/breunigs/photoepics/browser"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "photoepics",
	Short: "convert GPX into Mapillary photo sequences",
	Long:  "Photoepics takes a GeoJSON file as input and tries to find matching sequences of photos from Mapillary.",
}

func main() {
	rootCmd.AddCommand(cmdPurge())
	rootCmd.AddCommand(cmdLoad())
	rootCmd.AddCommand(cmdQuery())
	rootCmd.Execute()
}

func doStuff() {
	log.Println(browser.Get("https://a.mapillary.com/v3/images?client_id="))

}
