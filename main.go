package main

import (
	"encoding/json"
	"expvar"
	"log"
	"net/http"
	"time"

	"github.com/femot/openmap-tools/db"
)

var (
	dbName     string
	dbUser     string
	dbPassword string
	dbHost     string
)

var stats *Stats

type Stats struct {
	// Accounts
	AccountsInUse  int `json:"accounts_in_use"`
	AccountsBanned int `json:"accounts_banned"`
	AccountsTotal  int `json:"accounts_total"`
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
	dbHost = "localhost"
	dbName = "OpenPogoMap"
	// stuff
	stats = &Stats{}
	expvar.Publish("opm_stats", stats)
	go runStats()
	http.ListenAndServe(":8324", nil)
}

func runStats() {
	database, err := db.NewOpenMapDb(dbName, dbHost, dbUser, dbPassword)
	if err != nil {
		log.Fatal(err)
	}
	pollRate := 30 * time.Second
	for {
		// Accounts
		accountsTotal, accountsUse, accountsBanned, err := database.AccountStats()
		if err != nil {
			log.Println(err)
		}
		stats.AccountsTotal = accountsTotal
		stats.AccountsBanned = accountsBanned
		stats.AccountsInUse = accountsUse
		// Sleep
		time.Sleep(pollRate / 3)
		// Proxies
		proxiesAlive, proxiesUse, err := database.ProxyStats()
		if err != nil {
			log.Println(err)
		}
		stats.ProxiesInUse = proxiesUse
		stats.ProxiesAlive = proxiesAlive
		// Sleep
		time.Sleep(pollRate / 3)
		// MapObjects
		pokemonTotal, pokemonAlive, gyms, pokestops := database.MapObjectStats()
		stats.PokemonTotal = pokemonTotal
		stats.PokemonAlive = pokemonAlive
		stats.Gyms = gyms
		stats.Pokestops = pokestops
		// Sleep
		time.Sleep(pollRate / 3)
	}

}
