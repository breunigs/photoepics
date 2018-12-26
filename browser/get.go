package browser

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/cenkalti/backoff"
)

const readTimeout = time.Second * 10
const maxReadSize = 10 * 1024 * 1024 // 10 MiB
const userAgent = "photoepics/0.1"

var activeUrls = sync.Map{}

var client = &http.Client{
	Timeout: readTimeout,
}

func Get(url string) (result string, outerErr error) {
	var newWg sync.WaitGroup
	newWg.Add(1)
	obj, loaded := activeUrls.LoadOrStore(url, &newWg)
	wg, _ := obj.(*sync.WaitGroup)
	if loaded {
		wg.Wait()
		return Get(url)
	}
	defer activeUrls.Delete(url)
	defer wg.Done()

	op := func() (innerErr error) {
		result, innerErr = getNoRetry(url)
		return
	}

	exp := backoff.NewExponentialBackOff()
	exp.InitialInterval = 5 * time.Second
	outerErr = backoff.Retry(op, exp)
	return
}

func getNoRetry(url string) (string, error) {
	fromCache := ReadFromCache(url)
	if fromCache != "" {
		return fromCache, nil
	}

	EnsureRateLimit(url)

	log.Printf("Reading %s\n", url)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)

	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		err = fmt.Errorf("Unexpected status: got %v for %s", res.Status, url)
		log.Println(err)
		return "", err
	}
	bodyBytes, err := ioutil.ReadAll(io.LimitReader(res.Body, maxReadSize))
	if err != nil {
		return "", err
	}

	err = WriteToCache(url, bodyBytes)
	if err != nil {
		log.Printf("Failed to write disk cache for %s\n", url)
	}
	return string(bodyBytes), nil
}
