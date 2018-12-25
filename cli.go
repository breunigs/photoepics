package main

import (
	"log"
	"sync"

	mapset "github.com/deckarep/golang-set"
	"github.com/paulmach/orb/maptile"
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
			tiles := mapset.NewSet()
			for _, pt := range lineStr {
				tile := maptile.At(pt, gridZoomLevel)

				tiles.Add(tile)
			}

			log.Printf("Reading data for %d bounding boxes", tiles.Cardinality())
			var wg sync.WaitGroup
			for tile := range tiles.Iterator().C {
				wg.Add(1)
				go func(tile maptile.Tile) {
					defer wg.Done()
					FindSequences(tile)
				}(tile.(maptile.Tile))
			}
			wg.Wait()
			// find suitable bounding boxes on grid
			// log.Printf("%v    %v", tiles, err)
		},
	}
	cmd.Flags().StringVarP(&inputFilePath, "input", "i", "", "input file for which to generate a photo sequence")
	cmd.MarkFlagRequired("input")
	requireApiKey(cmd)
	filterByUserName(cmd)
	filterByDate(cmd)
	return cmd
}

func requireApiKey(cmd *cobra.Command) {
	cmd.Flags().StringVar(&mapillaryClientKey, "api-key", "", "Mapillary API Key")
	cmd.MarkFlagRequired("api-key")
}

func filterByUserName(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&mapillaryFilterUsers, "filter-users", "", "", "only use photos from these Mapillary users. Comma separated.")
}

func filterByDate(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&mapillaryFilterNewer, "filter-newer", "", "", "only use sequences newer than this date. Format YYYY-MM-DD.")
}
