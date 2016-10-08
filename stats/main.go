package main

import (
	"encoding/json"
	"expvar"
	"log"
	"net/http"
	"time"

	"github.com/femot/opm/opm"
	"github.com/pogointel/opm/db"
)

var opmSettings opm.Settings
var stats *Stats
var database *db.OpenMapDb

type Stats struct {
	// Accounts
	AccountsInUse      int `json:"accounts_in_use"`
	AccountsBanned     int `json:"accounts_banned"`
	AccountsChallenged int `json:"accounts_challenged"`
	AccountsTotal      int `json:"accounts_total"`
	// Proxies
	ProxiesAlive int `json:"proxies_alive"`
	ProxiesInUse int `json:"proxies_in_use"`
	// MapObjects
	PokemonTotal int `json:"pokemon_total"`
	PokemonAlive int `json:"pokemon_alive"`
	Gyms         int `json:"gyms"`
	Pokestops    int `json:"pokestops"`
}

func (s *Stats) String() string {
	b, _ := json.Marshal(s)
	return string(b)
}

func main() {
	// db
	var err error
	opmSettings, err := opm.LoadSettings("")
	if err != nil {
		log.Printf("Error loading settings (%s). Using default settings.\n", err)
	}
	// stuff
	stats = &Stats{}
	expvar.Publish("opm_stats", stats)
	database, err = db.NewOpenMapDb(opmSettings.DbName, opmSettings.DbHost, opmSettings.DbUser, opmSettings.DbPassword)
	if err != nil {
		log.Fatal(err)
	}
	go runStats()
	go runObjects()
	http.ListenAndServe(":8324", nil)
}

func runObjects() {
	for {
		// MapObjects
		pokemonTotal, pokemonAlive, gyms, pokestops := database.MapObjectStats()
		stats.PokemonTotal = pokemonTotal
		stats.PokemonAlive = pokemonAlive
		stats.Gyms = gyms
		stats.Pokestops = pokestops
		// Sleep
		time.Sleep(3 * time.Minute)
	}
}

func runStats() {
	for {
		// Accounts
		accountsTotal, accountsUse, accountsBanned, accountsChallenged, err := database.AccountStats()
		if err != nil {
			log.Println(err)
		}
		stats.AccountsTotal = accountsTotal
		stats.AccountsBanned = accountsBanned
		stats.AccountsInUse = accountsUse
		stats.AccountsChallenged = accountsChallenged
		// Proxies
		proxiesAlive, proxiesUse, err := database.ProxyStats()
		if err != nil {
			log.Println(err)
		}
		stats.ProxiesInUse = proxiesUse
		stats.ProxiesAlive = proxiesAlive
		// Sleep
		time.Sleep(15 * time.Second)
	}
}
