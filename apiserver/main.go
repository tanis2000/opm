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
	apiSettings settings
	keyMetrics  KeyMetrics
	apiMetrics  APIMetrics
	blacklist   map[string]bool
)

func main() {
	log.SetFlags(log.Lmicroseconds | log.Lshortfile)
	// Settings
	var err error
	apiSettings, err = loadSettings()
	if err != nil {
		log.Println(err)
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
