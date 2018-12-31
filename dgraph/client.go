package dgraph

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/dgraph-io/dgo"
	"github.com/dgraph-io/dgo/protos/api"
	"google.golang.org/grpc"
)

type DgraphInsertable interface {
	DgraphInsert() string
}

type Wrapper struct {
	client *dgo.Dgraph
}

func NewClient() Wrapper {
	d, err := grpc.Dial("localhost:9080", grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}

	client := dgo.NewDgraphClient(
		api.NewDgraphClient(d),
	)

	return Wrapper{
		client: client,
	}
}

type countRoot struct {
	Count []struct {
		Total int64 `json:"total"`
	} `json:"count"`
}

func (w Wrapper) Count(predicate string) int64 {
	query := `{
    count(func: has(<` + predicate + `>)) {
      total: count(uid)
    }
  }`
	resp := w.Query(query, map[string]string{})

	var r countRoot
	if err := json.Unmarshal(resp, &r); err != nil {
		log.Fatal(err)
	}
	return r.Count[0].Total
}

func (w Wrapper) Query(query string, params map[string]string) []byte {
	resp, err := w.client.NewTxn().QueryWithVars(context.Background(), query, params)
	if err != nil {
		log.Fatalf("Failed to run query: %+v\nOriginal Query was:\n%s\nwith params: %+v\n", err, query, params)
	}
	return resp.GetJson()
}

func (w Wrapper) CreateSchema(schema string) {
	err := w.client.Alter(context.Background(), &api.Operation{
		Schema: schema,
	})
	if err != nil {
		log.Fatalf("cannot create schema: %+v\n\nSchema was:\n%s", err, schema)
	}
}

func (w Wrapper) PurgeEverything() {
	err := w.client.Alter(context.Background(), &api.Operation{
		DropAll: true,
	})
	if err != nil {
		log.Fatalf("Failed to purge everything: %s", err)
	}
}

func (w Wrapper) insertStr(entry string) {
	maxRetries := 5
	for i := 1; i <= maxRetries; i++ {
		mu := &api.Mutation{
			CommitNow: true,
			SetNquads: []byte(entry),
		}
		_, err := w.client.NewTxn().Mutate(context.Background(), mu)
		if err == nil {
			return
		}

		if i != maxRetries && strings.Index(err.Error(), "Transaction has been aborted") >= 0 {
			time.Sleep(1 * time.Second)
			continue
		}

		log.Fatalf("Failed to insert entry into DB: %+v\n\nOriginal query was:\n%s", err, entry)
	}
}

func (w Wrapper) Insert(entry DgraphInsertable) {
	w.insertStr(entry.DgraphInsert())
}

func (w Wrapper) InsertBatch(entries []DgraphInsertable) {
	var b strings.Builder
	for _, entry := range entries {
		b.WriteString(entry.DgraphInsert())
	}
	w.insertStr(b.String())
}

func (w Wrapper) InsertStream(stream <-chan DgraphInsertable) {
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
