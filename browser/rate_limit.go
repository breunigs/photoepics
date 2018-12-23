package browser

import (
	"context"
	"log"
	"net/url"
	"sync"

	"golang.org/x/time/rate"
)

const burstRate = 5
const reqPerSec = 3

var hostLimiters = sync.Map{}

func EnsureRateLimit(uri string) {
	host := hostFromUrl(uri)
	err := getHostLimiter(host).Wait(context.Background())
	if err != nil {
		log.Fatal(err)
	}
}

func getHostLimiter(host string) *rate.Limiter {
	limiter, _ := hostLimiters.LoadOrStore(host, rate.NewLimiter(reqPerSec, burstRate))
	casted, ok := limiter.(*rate.Limiter)
	if !ok {
		log.Fatalf("Expected a *rate.Limiter, but was something else")
	}
	return casted
}

func hostFromUrl(uri string) string {
	u, err := url.Parse(uri)
	if err != nil {
		log.Fatal(err)
	}
	return u.Host
}
