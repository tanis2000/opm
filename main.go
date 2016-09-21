package main

import (
	"flag"
	"fmt"

	"net/http"

	"encoding/json"
	"log"

	"math/rand"

	"time"

	"github.com/femot/openmap-tools/db"
	"github.com/femot/openmap-tools/opm"
)

func main() {
	// Flags
	// DB
	dbHost := flag.String("dbhost", "localhost", "Host of the database")
	dbUser := flag.String("dbuser", "", "Username for the database")
	dbPass := flag.String("dbpass", "", "Password for the database")
	dbName := flag.String("dbname", "OpenPogoMap", "Name of the database")
	// Commands
	removePokemon := flag.Int64("removepokemon", -1, "Delete Pokemon which expire before the provided unix timestamp")
	dropProxies := flag.Bool("dropproxies", false, "Delete all proxies from the database")
	cleanProxies := flag.Bool("cleanproxies", false, "Marks all proxies as unused")
	cleanAccounts := flag.Bool("cleanaccounts", false, "Marks all accounts as unused")
	ufs := flag.Bool("ufs", false, "Update database from status")
	statusPage := flag.String("statuspage", "", "Status page to use with -ufs flag")
	removeDeadProxies := flag.Bool("removedeadproxies", false, "Remove all dead proxies from the database")
	addPokemon := flag.Bool("addpokemon", false, "Adds a pokemon to the database. Use with -id, -lat and -lng")
	pokeId := flag.Int("id", 151, "Pokemon Id to add to the database (-addpokemon)")
	lat := flag.Float64("lat", 34.008096, "Latitude for pokemon (-addpokemon)")
	lng := flag.Float64("lng", -118.497933, "Latitude for pokemon (-addpokemon)")
	// Parse flags
	flag.Parse()
	// Do something
	database, err := db.NewOpenMapDb(*dbName, *dbHost, *dbUser, *dbPass)
	if err != nil {
		fmt.Println(err)
		return
	}
	// Remove old Pokemon
	if *removePokemon != -1 {
		count, err := database.RemoveOldPokemon(*removePokemon)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Printf("Removed %d Pokemon from database\n", count)
	}
	// Clean up the Proxies
	if *dropProxies {
		err = database.DropProxies()
		if err != nil {
			fmt.Println(err)
		}
	}
	// Mark accounts as unused
	if *cleanAccounts {
		count, err := database.MarkAccountsAsUnused()
		if err != nil {
			fmt.Println(err)
		}
		fmt.Printf("Updated %d accounts\n", count)
	}
	// Mark proxies as unused
	if *cleanProxies {
		count, err := database.MarkProxiesAsUnused()
		if err != nil {
			fmt.Println(err)
		}
		fmt.Printf("Updated %d proxies\n", count)
	}
	// Remove dead proxies
	if *removeDeadProxies {
		count, err := database.RemoveDeadProxies()
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Printf("Removed %d proxies\n", count)
		}
	}
	// Add pokemon
	if *addPokemon {
		rand.Seed(time.Now().UnixNano())
		letterRunes := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")
		b := make([]rune, 16)
		for i := range b {
			b[i] = letterRunes[rand.Intn(len(letterRunes))]
		}
		randId := string(b)

		obj := opm.MapObject{
			Type:      opm.POKEMON,
			Lat:       *lat,
			Lng:       *lng,
			Id:        randId,
			PokemonId: *pokeId,
			Expiry:    time.Now().Add(15 * time.Minute).Unix(),
		}
		database.AddMapObject(obj)
	}

	// UFS
	if *ufs {
		req, _ := http.NewRequest("GET", *statusPage, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Println(err)
			return
		}
		var status []map[string]string
		err = json.NewDecoder(resp.Body).Decode(&status)
		if err != nil {
			log.Println(err)
			return
		}
		list := make([][]string, len(status))
		for i, v := range status {
			list[i] = []string{v["AccountName"], v["ProxyId"]}
		}
		count, err := database.Cleanup(list)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Printf("Updated %d database entries\n", count)
	}

}
