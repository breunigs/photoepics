package main

import (
	"log"

	"github.com/breunigs/photoepics/dgraph"
	"github.com/breunigs/photoepics/edge"
	"github.com/breunigs/photoepics/mapillary"
	"github.com/spf13/cobra"
)

var purgeConfirmed bool

func cmdPurge() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "purge",
		Short: "Deletes EVERYTHING from DB",
		Run: func(cmd *cobra.Command, args []string) {
			db := dgraph.NewClient()

			log.Printf("Purging…")
			db.PurgeEverything()

			log.Printf("Photos: %d", mapillary.PhotoCount(db))
			log.Printf("Edges: %d", edge.Count(db))
		},
	}
	cmd.Flags().BoolVarP(&purgeConfirmed, "confirm", "", false, "Please confirm that you really want to delete everything in the database")
	cmd.MarkFlagRequired("confirm")

	return cmd
}
