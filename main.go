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
var trainerQueue chan Session

func main() {
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
	// Load trainers
	trainers := LoadTrainers(settings.Accounts, &api.VoidFeed{}, &encrypt.Crypto{})
	// Create channels
	ticks = make(chan bool)
	trainerQueue = make(chan Session, len(trainers))
	// Start ticker
	go func(d time.Duration) {
		for {
			ticks <- true
			time.Sleep(d)
		}
	}(time.Duration(settings.ApiCallRate) * time.Millisecond)
	// Fill trainerQueue
	for _, t := range trainers {
		trainerQueue <- t
	}
	// Start webserver
	log.Println("Starting http server")
	listenAndServe()
}
