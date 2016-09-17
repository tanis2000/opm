package main

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/femot/gophermon/encrypt"
	"github.com/femot/pgoapi-go/api"
)

var settings Settings
var ticks chan bool
var feed api.Feed
var crypto api.Crypto
var dispatcher *Dispatcher

func main() {
	log.SetFlags(log.Lmicroseconds)
	// Check command line flags
	if len(os.Args) == 3 {
		if os.Args[1] == "test" {
			n, err := strconv.Atoi(os.Args[2])
			if err != nil {
				log.Fatal(err)
			}
			runTestMode(n)
			return
		}
	}

	var err error
	// Load settings
	settings, err = loadSettings()
	if err != nil {
		log.Fatal(err)
	}
	crypto = &encrypt.Crypto{}
	feed = &api.VoidFeed{}
	api.ProxyHost = settings.ProxyHost
	// Init dispatcher
	dispatcher = NewDispatcher(time.Second)
	dispatcher.Start()

	// Create channels
	ticks = make(chan bool)
	// Start ticker
	go func(d time.Duration) {
		for {
			ticks <- true
			time.Sleep(d)
		}
	}(time.Duration(settings.ApiCallRate) * time.Millisecond)
	// Start webserver
	log.Println("Starting http server")
	listenAndServe()
}
