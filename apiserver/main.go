package main

import (
	"expvar"
	"log"

	"github.com/pogointel/opm/db"
	"github.com/pogointel/opm/opm"
)

var (
	database    *db.OpenMapDb
	opmSettings opm.Settings
	keyMetrics  KeyMetrics
	apiMetrics  *APIMetrics
	blacklist   map[string]bool
)

func main() {
	log.SetFlags(log.Lmicroseconds | log.Lshortfile)
	// Settings
	var err error
	opmSettings = opm.LoadSettings("")
	// Db connections
	database, err = db.NewOpenMapDb(opmSettings.DbName, opmSettings.DbHost, opmSettings.DbUser, opmSettings.DbPassword)
	if err != nil {
		log.Fatal(err)
	}
	// Metrics
	apiMetrics = NewAPIMetrics()
	keyMetrics = make(map[string]APIKeyMetrics)
	expvar.Publish("keymetrics", keyMetrics)
	// Start webserver
	startHTTP()
}
