package main

import (
	"expvar"
	"log"

	"github.com/pogointel/opm/db"
	"github.com/pogointel/opm/opm"
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
	// Start webserver
	startHTTP()
}
