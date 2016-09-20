package main

import (
	"encoding/json"
	"io/ioutil"

	"github.com/femot/openmap-tools/util"
	"github.com/femot/pgoapi-go/api"
)

type Settings struct {
	Accounts    int    // Number of accounts to load from db
	ListenAddr  string // Listen address for http
	ProxyHost   string // Address of the openmap-proxy
	CacheRadius int    // Radius in meters for getting cached MapObjects
	ScanDelay   int    // Time between scans per account in seconds
	ApiCallRate int    // Time between API calls in milliseconds
	DbName      string // Name of the db
	DbHost      string // Host of the db
}

func loadSettings() (Settings, error) {
	// Read from file
	bytes, err := ioutil.ReadFile("config.json")
	if err != nil {
		return Settings{}, err
	}
	// Unmarshal json
	var settings Settings
	err = json.Unmarshal(bytes, &settings)
	if err != nil {
		return settings, err
	}
	return settings, err
}

func NewTrainerFromDb() (*util.TrainerSession, error) {
	p, err := database.GetProxy()
	if err != nil {
		return &util.TrainerSession{}, ErrBusy
	}
	a, err := database.GetAccount()
	if err != nil {
		database.ReturnProxy(p)
		return &util.TrainerSession{}, ErrBusy
	}
	trainer := util.NewTrainerSession(a, &api.Location{}, feed, crypto)
	trainer.SetProxy(p)
	return trainer, nil
}
