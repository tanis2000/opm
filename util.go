package main

import (
	"encoding/json"
	"io/ioutil"

	"github.com/femot/openmap-tools/db"
)

type Settings struct {
	Accounts    int
	ListenAddr  string
	ProxyHost   string
	ScanDelay   int // Time between scans per account in seconds
	ApiCallRate int // Time between API calls in milliseconds
	Db          *db.OpenMapDb
}

func loadSettings() (Settings, error) {
	bytes, err := ioutil.ReadFile("config.json")
	if err != nil {
		return Settings{}, err
	}
	var settings Settings
	err = json.Unmarshal(bytes, &settings)
	if err != nil {
		return settings, err
	}
	settings.Db, err = db.NewOpenMapDb("OpenPogoMap", "localhost")
	return settings, err
}
