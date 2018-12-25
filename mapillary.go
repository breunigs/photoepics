package main

import (
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/breunigs/photoepics/browser"
	"github.com/paulmach/orb/maptile"
)

const mapillaryBaseUrl = "https://a.mapillary.com/v3/"
const tileBuffer = 0.05 // i.e. 5% border around tile

func FindSequences(t maptile.Tile) {
	bbox := t.Bound(tileBuffer)
	bboxstr := fmt.Sprintf("%f,%f,%f,%f", bbox.Left(), bbox.Bottom(), bbox.Right(), bbox.Top())
	seqs := get("sequences", "per_page=1000&bbox="+bboxstr)
	log.Println(seqs)
}

func get(fun string, query string) string {
	url := mapillaryBaseUrl + fun
	url += "?client_id=" + mapillaryClientKey
	url += maybeFilterUsers()
	url += maybeFilterNewer()
	if query != "" {
		url += "&" + query
	}
	body, err := browser.Get(url)
	if err != nil {
		log.Fatalf("Failed to read from Mapillary: %+v", err)
	}
	return body
}

func maybeFilterUsers() string {
	if mapillaryFilterUsers == "" {
		return ""
	}

	matched, err := regexp.MatchString("^[a-zA-Z0-9_,-]+$", mapillaryFilterUsers)
	if err != nil {
		log.Fatalf("Failed to parse user filter: %+v", err)
	}
	if !matched {
		log.Fatalf("Failed to parse user filter. Only alphanumeric characters, underscores and dashes are allowed. Usernames should be separated by comma. Got: %s", mapillaryFilterUsers)
	}

	// spaced := strings.Replace(mapillaryFilterUsers, ",", ", ", -1)
	// log.Printf("photos by these users: %s", spaced)
	return "&usernames=" + mapillaryFilterUsers
}

func maybeFilterNewer() string {
	if mapillaryFilterNewer == "" {
		return ""
	}

	parsed, err := time.Parse("2006-01-02", mapillaryFilterNewer)
	if err != nil {
		log.Fatalf("Failed to parse date filter: %+v", err)
	}

	// log.Printf("photos newer than: %s", parsed.Format("2006-01-02"))
	return "&start_time=" + parsed.Format("2006-01-02")
}
