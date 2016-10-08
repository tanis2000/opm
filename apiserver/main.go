package main

import (
	"expvar"
	"log"
	"net/http"
	"time"

	"github.com/femot/opm/opm"
	"github.com/pogointel/opm/db"
)

var database *db.OpenMapDb
var apiSettings settings
var opmSettings opm.Settings
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
	opmSettings, err = opm.LoadSettings("")
	// Db connections
	database, err = db.NewOpenMapDb(opmSettings.DbName, opmSettings.DbHost, opmSettings.DbUser, opmSettings.DbPassword)
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
		Addr:         opmSettings.APIListenAddress,
		Handler:      mux,
	}
	// Run server
	log.Fatal(s.ListenAndServe())
}
