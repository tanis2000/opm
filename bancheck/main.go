package main

import (
	"fmt"
	"log"
	"time"

	"github.com/femot/gophermon/encrypt"
	"github.com/femot/pgoapi-go/api"
	"github.com/pogointel/opm/db"
	"github.com/pogointel/opm/opm"
	"github.com/pogointel/opm/util"
)

var database *db.OpenMapDb
var feed api.Feed
var crypto api.Crypto
var opmSettings opm.Settings

func main() {
	// Log
	log.SetFlags(log.Lmicroseconds | log.Lshortfile)
	// Settings
	var err error
	opmSettings, err = opm.LoadSettings("")
	if err != nil {
		log.Printf("Error loading settings (%s). Using default settings.\n", err)
	}
	api.ProxyHost = fmt.Sprintf("%s:%d", opmSettings.ProxyListenAddress, opmSettings.ProxyListenPort)
	// Databse connections
	database, err = db.NewOpenMapDb(opmSettings.DbName, opmSettings.DbHost, opmSettings.DbUser, opmSettings.DbPassword)
	if err != nil {
		log.Fatal(err)
	}
	// Init vars
	feed = &api.VoidFeed{}
	crypto = &encrypt.Crypto{}
	// Main loop
	for {
		// Get accounts
		accounts, err := database.GetBannedAccounts()
		if err != nil {
			log.Println(err)
			time.Sleep(time.Minute)
		}
		// Check accounts
		for _, a := range accounts {
			log.Printf("Checking <%s> now\n", a.Username)
			checkAccount(a)
			time.Sleep(30 * time.Second)
		}
		// Wait before next round
		time.Sleep(30 * time.Second)
	}
}

func checkAccount(account opm.Account) {
	// Create session
	trainer := util.NewTrainerSession(account, &api.Location{}, feed, crypto)
	// Get a proxy
	proxy, err := database.GetProxy()
	if err != nil {
		log.Println(err)
		return
	}
	defer database.ReturnProxy(proxy)
	trainer.SetProxy(proxy)
	// Login
	err = trainer.Login()
	count := 0
	for err != nil && err != api.ErrNewRPCURL {
		log.Println(err)
		if count > 5 {
			log.Println("Cant login")
			break
		}
		time.Sleep(10 * time.Second)
		err = trainer.Login()
		count++
	}
	// Santa Monica Pier
	lat := 34.0075
	lng := -118.499795
	// Move there
	trainer.MoveTo(&api.Location{Lat: lat, Lon: lng})
	// Perform API call
	_, err = trainer.GetPlayerMap()
	if err != nil && err != api.ErrNewRPCURL {
		if err == api.ErrBadRequest {
			log.Printf("Account <%s> banned for sure! (StatusCode 3)", account.Username)
		} else {
			log.Printf("Account <%s> produced error: %s", account.Username, err.Error())
		}
	} else {
		log.Printf("Account <%s> probably not banned, or just temp ban. Marking as not banned", account.Username)
		account.Banned = false
		database.UpdateAccount(account)
	}
}
