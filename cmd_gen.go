package main

import (
	"fmt"
	"log"

	"github.com/breunigs/photoepics/cheapruler"
	"github.com/breunigs/photoepics/dgraph"
	"github.com/breunigs/photoepics/mapillary"
)

func GenPathAlong(inputFilePath, startImageKey, endImageKey string) {
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

	startPic := mapillary.PhotoByKey(db, startImageKey)
	endPic := mapillary.PhotoByKey(db, endImageKey)

	db.InsertStream(calcWeightsAlong(db, lineStr, 25))

	resp := db.Query(`
	       {
	         path as shortest(from: `+startPic.Uid+`, to: `+endPic.Uid+`, numpaths: 2) {
	           transitionable @facets(weight)
	         }
	         path(func: uid(path)) { uid, key }
	       }`,
		map[string]string{})
	fmt.Println(string(resp))
}
