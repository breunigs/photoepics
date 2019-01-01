package main

import (
	"log"

	"github.com/breunigs/photoepics/cheapruler"
	"github.com/breunigs/photoepics/dgraph"
	"github.com/breunigs/photoepics/edge"
	"github.com/breunigs/photoepics/mapillary"
	"github.com/spf13/cobra"
)

func cmdLoad() *cobra.Command {
	var inputFilePath string
	var mapConf mapillary.Config
	var trackID int

	cmd := &cobra.Command{
		Use:   "load",
		Short: "Loads images along the given file. Also calculates desirability for the images it finds.",
		Run: func(cmd *cobra.Command, args []string) {
			runCmdLoad(mapConf, inputFilePath, trackID)
		},
	}
	cmd.Flags().StringVarP(&inputFilePath, "input", "i", "", "input file for which to generate a photo sequence")
	cmd.MarkFlagRequired("input")
	requireAPIKey(&mapConf, cmd)
	filterByUserName(&mapConf, cmd)
	filterByDate(&mapConf, cmd)
	cmd.Flags().IntVarP(&trackID, "track", "", -1, "If the input file has more than one track, use this to specify the index of the desired one. It will be ignored if there is only one track.")

	return cmd
}

func runCmdLoad(mapConf mapillary.Config, inputFilePath string, trackID int) {
	db := dgraph.NewClient()

	if mapillary.PhotoCount(db) > 0 {
		log.Fatalf("Tried to load data, but database is not empty. This will lead to wrong results since the entries depend on the given input file. Please purge the DB.")
	}
	downloadAlong(mapConf, db, inputFilePath, trackID)
}

func requireAPIKey(mapConf *mapillary.Config, cmd *cobra.Command) {
	cmd.Flags().StringVar(&mapConf.APIKey, "api-key", "", "Mapillary API Key")
	cmd.MarkFlagRequired("api-key")
}

func filterByUserName(mapConf *mapillary.Config, cmd *cobra.Command) {
	cmd.Flags().StringVarP(&mapConf.FilterUsers, "filter-users", "", "", "only use photos from these Mapillary users. Comma separated.")
}

func filterByDate(mapConf *mapillary.Config, cmd *cobra.Command) {
	cmd.Flags().StringVarP(&mapConf.FilterNewer, "filter-newer", "", "", "only use sequences newer than this date. Format YYYY-MM-DD.")
}

func downloadAlong(mapConf mapillary.Config, db dgraph.Wrapper, inputFilePath string, trackID int) {
	lineStr, err := trackFromFile(inputFilePath, trackID)
	if err != nil {
		log.Fatalf("Cannot extract GPS track from file: %+v", err)
	}
	cheapruler.Init(lineStr[0][1])

	insertChan := make(chan dgraph.DgraphInsertable, 50)
	go func() {
		defer close(insertChan)
		photoChan := mapillary.FindSequences(mapConf, lineStr)
		for x := range photoChan {
			insertChan <- x
		}
	}()

	db.CreateSchema(mapillary.PhotoDgraphSchema())
	db.InsertStream(insertChan)

	db.InsertStream(edge.CalcWeightsAlong(db, lineStr, 25))
}
