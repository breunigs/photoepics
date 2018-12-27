package dgraph

import (
	"context"
	"log"
	"strings"
	"sync"

	"github.com/dgraph-io/dgo"
	"github.com/dgraph-io/dgo/protos/api"
	"google.golang.org/grpc"
)

type DgraphInsertable interface {
	DgraphInsert() string
}

type wrapper struct {
	client *dgo.Dgraph
}

func NewClient() wrapper {
	d, err := grpc.Dial("localhost:9080", grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}

	client := dgo.NewDgraphClient(
		api.NewDgraphClient(d),
	)

	return wrapper{
		client: client,
	}
}

func (w wrapper) CreateSchema(schema string) {
	err := w.client.Alter(context.Background(), &api.Operation{
		Schema: schema,
	})
	if err != nil {
		log.Fatalf("cannot create schema: %+v\n\nSchema was:\n%s", err, schema)
	}
}

func (w wrapper) insertStr(entry string) {
	mu := &api.Mutation{
		CommitNow: true,
	}

	mu.SetNquads = []byte(entry)
	_, err := w.client.NewTxn().Mutate(context.Background(), mu)
	if err != nil {
		log.Fatalf("Failed to insert entry into DB: %+v\n\nOriginal query was:\n%s", err, entry)
	}
}

func (w wrapper) Insert(entry DgraphInsertable) {
	w.insertStr(entry.DgraphInsert())
}

func (w wrapper) InsertBatch(entries []DgraphInsertable) {
	var b strings.Builder
	for _, entry := range entries {
		b.WriteString(entry.DgraphInsert())
	}
	w.insertStr(b.String())
}

func (w wrapper) InsertStream(stream <-chan DgraphInsertable) {
	var b strings.Builder

	i := 0

	var wg sync.WaitGroup
	for entry := range stream {
		i++
		b.WriteString(entry.DgraphInsert())
		if i == 50 {
			i = 0
			wg.Add(1)
			go func(query string) {
				defer wg.Done()
				w.insertStr(query)
			}(b.String())
			b.Reset()
		}
	}
	if i > 0 {
		w.insertStr(b.String())
	}
	wg.Wait()
}
