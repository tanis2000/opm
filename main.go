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

var database *db.OpenMapDb
var feed api.Feed
var crypto api.Crypto

func main() {
	// Log
	log.SetFlags(log.Lmicroseconds | log.Lshortfile)
	// Settings
	dbName := "OpenPogoMap"
	dbHost := "localhost"
	dbUser := ""
	dbPassword := ""
	// Databse connections
	database, err := db.NewOpenMapDb(dbName, dbHost, dbUser, dbPassword)
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
			checkAccount(a)
		}
		// Wait before next round
		time.Sleep(30 * time.Second)
	}
}

func checkAccount(account opm.Account) {
	// Create session
	s := util.NewTrainerSession(account, &api.Location{}, feed, crypto)
	// Login
	err := s.Login()
	count := 0
	for err != nil {
		log.Println(err)
		if count > 5 {
			log.Println("Cant login")
			break
		}
		time.Sleep(10 * time.Second)
		err = s.Login()
		count++
	}
	// Santa Monica Pier
	lat := 34.0075
	lng := -118.499795
	// Move there
	s.MoveTo(&api.Location{Lat: lat, Lon: lng})
	// Perform API call
	_, err = s.GetPlayerMap()
	if err != nil {
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
