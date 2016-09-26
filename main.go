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
	go asdf()
	http.ListenAndServe(":8324", nil)
}

func asdf() {
	database, err := db.NewOpenMapDb(dbName, dbHost, dbUser, dbPassword)
	if err != nil {
		log.Fatal(err)
	}
	pollRate := time.Second
	for {
		accountsTotal, accountsUse, accountsBanned, err := database.AccountStats()
		if err != nil {
			log.Println(err)
		}
		proxiesAlive, proxiesUse, err := database.ProxyStats()
		if err != nil {
			log.Println(err)
		}
		pokemonTotal, pokemonAlive, gyms, pokestops := database.MapObjectStats()

		stats.AccountsTotal = accountsTotal
		stats.AccountsBanned = accountsBanned
		stats.AccountsInUse = accountsUse
		stats.ProxiesInUse = proxiesUse
		stats.ProxiesAlive = proxiesAlive
		stats.PokemonTotal = pokemonTotal
		stats.PokemonAlive = pokemonAlive
		stats.Gyms = gyms
		stats.Pokestops = pokestops

		time.Sleep(pollRate)
	}

}
