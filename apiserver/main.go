package main

import (
	"expvar"
	"log"
	"net/http"
	"time"

	"github.com/pogointel/opm/db"
)

var database *db.OpenMapDb
var apiSettings settings
var keyMetrics KeyMetrics
var apiMetrics APIMetrics

func main() {
	log.SetFlags(log.Lmicroseconds | log.Lshortfile)
	// Settings
	var err error
	apiSettings, err = loadSettings()
	if err != nil {
		log.Fatal(err)
	}
	// Db connections
	database, err = db.NewOpenMapDb(apiSettings.DbName, apiSettings.DbHost, apiSettings.DbUser, apiSettings.DbPassword)
	if err != nil {
		log.Fatal(err)
	}
	// Expvar
	keyMetrics = make(map[string]APIKeyMetrics)
	expvar.Publish("metrics", keyMetrics)
	// Routes/Handlers
	mux := http.NewServeMux()
	scanHandler, err := createScanProxy()
	if err != nil {
		log.Fatal(err)
	}
	mux.Handle("/q", scanHandler)
	mux.HandleFunc("/c", handleFuncDecorator(cacheHandler))
	mux.HandleFunc("/submit", handleFuncDecorator(submitHandler))
	mux.Handle("/debug/vars", http.DefaultServeMux)
	// Create http server with timeouts
	s := http.Server{
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		Addr:         apiSettings.ListenAddr,
		Handler:      mux,
	}
	// Run server
	log.Fatal(s.ListenAndServe())
}
