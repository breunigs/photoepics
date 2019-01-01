package mapillary

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/breunigs/photoepics/browser"
)

type jsonGraphInt64 struct {
	Type  string `json:"$type"`
	Value int64  `json:"value"`
}

type imageByKey struct {
	CapturedAt jsonGraphInt64 `json:"captured_at"`
	MergeCC    jsonGraphInt64 `json:"merge_cc"`
}

type graphImageByKey struct {
	JsonGraph struct {
		ImageByKey map[string]imageByKey `json:"imageByKey"`
	} `json:"jsonGraph"`
}

func getApi(conf Config, fun string, query string) string {
	url := mapillaryBaseUrl + fun
	url += "?client_id=" + conf.APIKey
	url += maybeFilterUsers(conf)
	url += maybeFilterNewer(conf)
	if query != "" {
		url += "&" + query
	}
	body, err := browser.Get(url)
	if err != nil {
		log.Fatalf("Failed to read from Mapillary: %+v", err)
	}
	return body
}

func getImageByKeys(conf Config, imageKey []string) map[string]imageByKey {
	imgKeys := strings.Join(imageKey, `","`)

	url := mapillaryBaseUrl + "model.json"
	url += "?client_id=" + conf.APIKey
	url += "&method=get"
	url += fmt.Sprintf(`&paths=[["imageByKey",["%s"],["captured_at","merge_cc"]]]`, imgKeys)
	body, err := browser.Get(url)
	if err != nil {
		log.Fatalf("Failed to read from Mapillary: %+v", err)
	}

	res := graphImageByKey{}
	err = json.Unmarshal([]byte(body), &res)
	if err != nil {
		log.Fatalf("Unexpected output for imageByKey: %+v", err)
	}
	return res.JsonGraph.ImageByKey
}

func maybeFilterUsers(conf Config) string {
	if conf.FilterUsers == "" {
		return ""
	}

	matched, err := regexp.MatchString("^[a-zA-Z0-9_,-]+$", conf.FilterUsers)
	if err != nil {
		log.Fatalf("Failed to parse user filter: %+v", err)
	}
	if !matched {
		log.Fatalf("Failed to parse user filter. Only alphanumeric characters, underscores and dashes are allowed. Usernames should be separated by comma. Got: %s", conf.FilterUsers)
	}

	// spaced := strings.Replace(conf.FilterUsers, ",", ", ", -1)
	// log.Printf("photos by these users: %s", spaced)
	return "&usernames=" + conf.FilterUsers
}

func maybeFilterNewer(conf Config) string {
	if conf.FilterNewer == "" {
		return ""
	}

	parsed, err := time.Parse("2006-01-02", conf.FilterNewer)
	if err != nil {
		log.Fatalf("Failed to parse date filter: %+v", err)
	}

	// log.Printf("photos newer than: %s", parsed.Format("2006-01-02"))
	return "&start_time=" + parsed.Format("2006-01-02")
}
