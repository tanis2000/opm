package main

import (
	"log"
	"time"

	"github.com/femot/gophermon/encrypt"
	"github.com/femot/openmap-tools/db"
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
	database, err = db.NewOpenMapDb(settings.DbName, settings.DbHost, settings.DbUser, settings.DbPassword)
	if err != nil {
		log.Fatal(err)
	}
	// Load trainers
	trainers := make([]*util.TrainerSession, 0)
	for {
		t, err := NewTrainerFromDb()
		if err != nil {
			log.Println(err)
			break
		}
		trainers = append(trainers, t)
		if len(trainers) >= settings.Accounts {
			break
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
