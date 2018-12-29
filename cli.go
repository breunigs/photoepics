package main

import (
	"log"

	"github.com/breunigs/photoepics/cheapruler"
	"github.com/breunigs/photoepics/dgraph"
	"github.com/breunigs/photoepics/mapillary"
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
			cheapruler.Init(lineStr[0][1])

			insertChan := make(chan dgraph.DgraphInsertable, 50)
			go func() {
				defer close(insertChan)
				photoChan := mapillary.FindSequences(mapConf, lineStr)
				for x := range photoChan {
					insertChan <- x
				}
			}()

			db := dgraph.NewClient()
			db.CreateSchema(mapillary.PhotoDgraphSchema())

			db.InsertStream(insertChan)
		},
	}
	cmd.Flags().StringVarP(&inputFilePath, "input", "i", "", "input file for which to generate a photo sequence")
	cmd.MarkFlagRequired("input")
	requireApiKey(cmd)
	filterByUserName(cmd)
	filterByDate(cmd)

	cmd.Flags().StringVar(&startImageKey, "start-image", "", "The image to start from")
	cmd.MarkFlagRequired("start-image")
	cmd.Flags().StringVar(&endImageKey, "end-image", "", "The image to stop at")
	cmd.MarkFlagRequired("end-image")

	return cmd
}

func requireApiKey(cmd *cobra.Command) {
	cmd.Flags().StringVar(&mapConf.ApiKey, "api-key", "", "Mapillary API Key")
	cmd.MarkFlagRequired("api-key")
}

func filterByUserName(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&mapConf.FilterUsers, "filter-users", "", "", "only use photos from these Mapillary users. Comma separated.")
}

func filterByDate(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&mapConf.FilterNewer, "filter-newer", "", "", "only use sequences newer than this date. Format YYYY-MM-DD.")
}
