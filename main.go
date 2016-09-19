package main

import (
	"log"
	"time"

	"github.com/femot/gophermon/encrypt"
	"github.com/femot/openmap-tools/opm"
	"github.com/femot/openmap-tools/util"
	"github.com/femot/pgoapi-go/api"
)

var settings Settings
var ticks chan bool
var feed api.Feed
var crypto api.Crypto
var trainerQueue *util.TrainerQueue

func main() {
	log.SetFlags(log.Lmicroseconds)
	var err error
	// Load settings
	settings, err = loadSettings()
	if err != nil {
		log.Fatal(err)
	}
	crypto = &encrypt.Crypto{}
	feed = &api.VoidFeed{}
	api.ProxyHost = settings.ProxyHost
	// Load trainers
	trainers := make([]*util.TrainerSession, settings.Accounts)
	for i := range trainers {
		a, err := settings.Db.GetAccount()
		if err != nil {
			log.Fatal("Not enough accounts")
		}
		trainers[i] = util.NewTrainerSession(a, &api.Location{}, feed, crypto)
		if p, err := settings.Db.GetProxy(); err == nil {
			trainers[i].SetProxy(p)
		} else {
			log.Println("No proxy available. Assigning Id 0")
			trainers[i].SetProxy(opm.Proxy{Id: "0"})
		}

	}
	trainerQueue = util.NewTrainerQueue(trainers)

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
