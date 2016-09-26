package main

import (
	"log"
	"time"

	"expvar"

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
var status Status
var metrics *ScannerMetrics
var blacklist map[string]bool

func main() {
	log.SetFlags(log.Lshortfile | log.Ltime)
	var err error
	// Load settings
	settings, err = loadSettings()
	if err != nil {
		log.Fatal(err)
	}
	status = make(Status)
	crypto = &encrypt.Crypto{}
	feed = &api.VoidFeed{}
	api.ProxyHost = settings.ProxyHost
	blacklist = make(map[string]bool)
	// Metrics/expvar
	metrics = NewScannerMetrics()
	expvar.Publish("scan_reponse_times_ms", metrics.ScanResponseTimesMs)
	expvar.Publish("cache_reponse_times_ns", metrics.CacheResponseTimesNs)
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
		status[t.Account.Username] = opm.StatusEntry{AccountName: t.Account.Username, ProxyId: t.Proxy.Id}
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
