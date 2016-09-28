package main

import (
	"log"
	"net/http"
	"time"

	"github.com/femot/openmap-tools/db"
)

var database *db.OpenMapDb
var settings Settings

func main() {
	log.SetFlags(log.Lmicroseconds | log.Lshortfile)
	// Settings
	var err error
	settings, err = loadSettings()
	if err != nil {
		log.Fatal(err)
	}
	// Db connections
	database, err = db.NewOpenMapDb(settings.DbName, settings.DbHost, settings.DbUser, settings.DbPassword)
	if err != nil {
		log.Fatal(err)
	}
	// Routes/Handlers
	mux := http.NewServeMux()
	mux.HandleFunc("/submit", handleFuncDecorator(submitHandler))
	// Create http server with timeouts
	s := http.Server{
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		Addr:         settings.ListenAddr,
		Handler:      mux,
	}
	// Run server
	log.Fatal(s.ListenAndServe())
}
