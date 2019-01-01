package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/breunigs/photoepics/dgraph"
	"github.com/breunigs/photoepics/edge"
	"github.com/breunigs/photoepics/mapillary"
	"github.com/spf13/cobra"
)

const emptyImageKey = "           ?          "

func cmdQuery() *cobra.Command {
	var startImageKey string
	var endImageKey string

	cmd := &cobra.Command{
		Use:   "query",
		Short: "Attempts to find path between two images.",
		Run: func(cmd *cobra.Command, args []string) {
			runCmdQuery(startImageKey, endImageKey)
		},
	}

	cmd.Flags().StringVar(&startImageKey, "start-image", "", "The image to start from")
	cmd.MarkFlagRequired("start-image")
	cmd.Flags().StringVar(&endImageKey, "end-image", "", "The image to stop at")
	cmd.MarkFlagRequired("end-image")

	return cmd
}

type shortestPath struct {
	Path []mapillary.Photo
}

func runCmdQuery(startImageKey, endImageKey string) {
	db := dgraph.NewClient()

	startPic := mapillary.PhotoByKey(db, startImageKey)
	endPic := mapillary.PhotoByKey(db, endImageKey)

	if mapillary.PhotoCount(db) == 0 || edge.Count(db) == 0 {
		log.Fatalf("Hmm, there are no photos or edges in the database. Did you run the load command?")
	}

	resp := db.Query(`
         {
           path as shortest(from: `+startPic.Uid+`, to: `+endPic.Uid+`, numpaths: 1) {
             transitionable @facets(weight)
           }
           path(func: uid(path)) { `+mapillary.PhotoReadQueryBody+` }
         }`,
		map[string]string{})

	var r shortestPath
	if err := json.Unmarshal(resp, &r); err != nil {
		log.Fatal(err)
	}

	// full list
	// fmt.Println("DB UID     SEQUENCE KEY             IMAGE KEY")
	// for _, pic := range r.Path {
	// 	fmt.Printf("(%s) %s:  %s\n", pic.Uid, pic.Sequence, pic.Key)
	// }

	// abbreviated
	prevSeq := ""

	seqStart := emptyImageKey
	seqEnd := emptyImageKey

	for _, pic := range r.Path {
		if prevSeq == pic.Sequence {
			seqEnd = pic.Key
			continue
		}

		if prevSeq != "" {
			if seqEnd == emptyImageKey {
				seqStart, seqEnd = seqEnd, seqStart
			}
			fmt.Println(`{ "seq": "` + prevSeq + `", "from": "` + seqStart + `", "to": "` + seqEnd + `" },`)
		}

		prevSeq = pic.Sequence
		seqStart = pic.Key
		seqEnd = emptyImageKey
	}
	fmt.Println(`{ "seq": "` + prevSeq + `", "from": "` + seqStart + `", "to": "` + emptyImageKey + `" },`)
}
