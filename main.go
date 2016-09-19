package main

import (
	"log"
	"time"

	"github.com/femot/gophermon/encrypt"
	"github.com/femot/openmap-tools/db"
	"github.com/femot/openmap-tools/opm"
	"github.com/femot/openmap-tools/util"
	"github.com/femot/pgoapi-go/api"
)

var settings Settings
var ticks chan bool
var feed api.Feed
var crypto api.Crypto
var trainerQueue *util.TrainerQueue
var database *db.OpenMapDb

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
	// Init db
	database, err = db.NewOpenMapDb(settings.DbName, settings.DbHost)
	if err != nil {
		log.Fatal(err)
	}
	// Load trainers
	trainers := make([]*util.TrainerSession, settings.Accounts)
	for i := range trainers {
		// Get opm.Account from db
		a, err := database.GetAccount()
		if err != nil {
			log.Fatal("Not enough accounts")
		}
		// Initialize *util.TrainerSession
		trainers[i] = util.NewTrainerSession(a, &api.Location{}, feed, crypto)
		// Assign a proxy
		if p, err := database.GetProxy(); err == nil {
			trainers[i].SetProxy(p)
		} else {
			// Trainer will try to get new proxy, when a request is sent to him
			log.Println("No proxy available. Assigning Id 0")
			trainers[i].SetProxy(opm.Proxy{Id: "0"})
		}

	}
	// Init trainerQueue
	trainerQueue = util.NewTrainerQueue(trainers)
	// Start ticker
	ticks = make(chan bool)
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
