package main

import (
	"log"
	"time"

	"github.com/breunigs/photoepics/browser"
)

func main() {
	browser.Expire()

	go doStuff()
	// go doStuff()
	// go doStuff()
	// go doStuff()
	// go doStuff()
	// go doStuff()
	// go doStuff()
	// go doStuff()
	// go doStuff()
	// go doStuff()
	// go doStuff()
	// go doStuff()
	// go doStuff()
	// go doStuff()
	// go doStuff()

	time.Sleep(10 * time.Minute)
}

func doStuff() {
	log.Println(browser.Get("https://a.mapillary.com/v3/images?client_id="))

}
