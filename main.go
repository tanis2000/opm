package main

import (
	"context"
	"expvar"
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
var loginTicks chan bool
var feed api.Feed
var crypto api.Crypto
var trainerQueue *util.TrainerQueue
var database *db.OpenMapDb
var status Status
var metrics *ScannerMetrics
var blacklist map[string]bool

func main() {
	log.SetFlags(log.Lmicroseconds | log.Lshortfile)
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
	// Metrics
	metrics = NewScannerMetrics()
	expvar.Publish("scanner_metrics", metrics)
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
	// Queue up all the trainer logins
	go func(trainers []*util.TrainerSession) {
		count := 0
		for _, t := range trainers {
			if !t.IsLoggedIn() {
				<-loginTicks
				log.Printf("Logging in %s", t.Account.Username)
				t.Context, _ = context.WithTimeout(context.Background(), 10*time.Second)
				err := t.Login()
				if err == api.ErrProxyDead {
					p, err := database.GetProxy()
					if err != nil {
						t.SetProxy(p)
					}
				}
				if err != nil {
					log.Println(err)
				}
			} else {
				count++
				if count >= len(trainers) {
					break
				}
			}
		}
		log.Println("All treners logged in")
	}(trainers)
	// Init trainerQueue
	trainerQueue = util.NewTrainerQueue(trainers)
	// Start ticker
	loginTicks = make(chan bool)
	go func(d time.Duration) {
		for {
			loginTicks <- true
			time.Sleep(d)
		}
	}(1 * time.Second)

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
