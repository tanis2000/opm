package main

import (
	"log"

	"time"

	"github.com/femot/gophermon/encrypt"
	"github.com/femot/pgoapi-go/api"
)

var settings Settings
var ticks chan bool
var trainerQueue chan *TrainerSession

func main() {
	var err error
	// Load settings
	settings, err = loadSettings()
	if err != nil {
		log.Fatal("Could not load settings")
	}
	// Load trainers
	trainers := LoadTrainers(settings.Accounts, &api.VoidFeed{}, &encrypt.Crypto{}, &api.Location{})
	// Create channels
	ticks = make(chan bool)
	trainerQueue = make(chan *TrainerSession, len(trainers))
	// Start ticker
	go func(d time.Duration) {
		for {
			ticks <- true
			time.Sleep(d)
		}
	}(200 * time.Millisecond)
	// Fill trainerQueue
	for _, t := range trainers {
		trainerQueue <- t
	}
	// Start webserver
	log.Println("Starting http server")
	listenAndServe()
}
