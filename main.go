package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strings"
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
	addAccounts := flag.Bool("addaccounts", false, "Add accounts to the db")
	accountsFile := flag.String("accountsfile", "accounts.txt", "Add accounts from provided file to database")
	cleanProxies := flag.Bool("cleanproxies", false, "Marks all proxies as unused")
	cleanAccounts := flag.Bool("cleanaccounts", false, "Marks all accounts as unused")
	ufs := flag.Bool("ufs", false, "Update database from status")
	statusPage := flag.String("statuspage", "http://localhost:8000/s", "Status page to use with -ufs and -status flags")
	secret := flag.String("secret", "meow", "Secret for the status page")
	status := flag.Bool("status", false, "Show status")
	removeDeadProxies := flag.Bool("removedeadproxies", false, "Remove all dead proxies from the database")
	addPokemon := flag.Bool("addpokemon", false, "Adds a pokemon to the database. Use with -id, -lat and -lng")
	pokeId := flag.Int("id", 151, "Pokemon Id to add to the database (-addpokemon)")
	lat := flag.Float64("lat", 34.008096, "Latitude for pokemon (-addpokemon)")
	lng := flag.Float64("lng", -118.497933, "Latitude for pokemon (-addpokemon)")
	// API keys
	key := flag.String("key", "", "API key. Use with -enablekey, -disablekey, ...")
	addKey := flag.String("addkey", "", "Adds an Api key to the database")
	enableKey := flag.Bool("enablekey", false, "Enables an Api key")
	disableKey := flag.Bool("disablekey", false, "Disables an Api key")
	verifyKey := flag.Bool("verifykey", false, "Verifies an Api key")
	unverifyKey := flag.Bool("unverifykey", false, "Unverifies an Api key")
	setName := flag.String("setname", "", "Sets the name for an API key")
	setURL := flag.String("seturl", "", "Sets the URL for an API key")
	keyStats := flag.Bool("keystats", false, "Shows stats for API keys")
	// Parse flags
	flag.Parse()
	// Do something
	database, err := db.NewOpenMapDb(*dbName, *dbHost, *dbUser, *dbPass)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Api key stuff
	// stats
	if *keyStats {
		stats := database.ApiKeyStats()
		var lines []string
		for k, v := range stats {
			if v > 0 {
				lines = append(lines, fmt.Sprintf("%-12s %13d", k, v))
			}
		}
		for _, l := range lines {
			fmt.Println(l)
		}
	}
	// Add
	if *addKey != "" {
		err := database.AddApiKey(opm.ApiKey{Key: *addKey})
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Printf("Key %s added\n", *addKey)
		}
	}
	// Enable
	if *enableKey && *key != "" {
		k, err := database.GetApiKey(*key)
		if err != nil {
			fmt.Println(err)
		} else {
			if !k.Enabled {
				k.Enabled = true
				database.UpdateApiKey(k)
			} else {
				fmt.Println("Key already enabled")
			}
		}
	}

	// Disable
	if *disableKey && *key != "" {
		k, err := database.GetApiKey(*key)
		if err != nil {
			fmt.Println(err)
		} else {
			if k.Enabled {
				k.Enabled = false
				database.UpdateApiKey(k)
			} else {
				fmt.Println("Key already disabled")
			}
		}
	}
	// Verify
	if *verifyKey && *key != "" {
		k, err := database.GetApiKey(*key)
		if err != nil {
			fmt.Println(err)
		} else {
			if !k.Verified {
				k.Verified = true
				database.UpdateApiKey(k)
			} else {
				fmt.Println("Key already verified")
			}
		}
	}
	// Unverify
	if *unverifyKey && *key != "" {
		k, err := database.GetApiKey(*key)
		if err != nil {
			fmt.Println(err)
		} else {
			if k.Verified {
				k.Verified = false
				database.UpdateApiKey(k)
			} else {
				fmt.Println("Key not verified")
			}
		}
	}
	// Set name for API key
	if *setName != "" && *key != "" {
		k, err := database.GetApiKey(*key)
		if err != nil {
			fmt.Println(err)
		} else {
			k.Name = *setName
			database.UpdateApiKey(k)
		}
	}
	// Set URL for API key
	if *setURL != "" && *key != "" {
		k, err := database.GetApiKey(*key)
		if err != nil {
			fmt.Println(err)
		} else {
			k.URL = *setURL
			database.UpdateApiKey(k)
		}
	}

	// Status
	if *status {
		// Scanner status
		req, _ := http.NewRequest("GET", fmt.Sprintf("%s?secret=%s", *statusPage, *secret), nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Println(err)
		}
		var s []opm.StatusEntry
		err = json.NewDecoder(resp.Body).Decode(&s)
		if err != nil {
			log.Println(err)
		} else {
			fmt.Printf("Scanner currently using %d accounts/proxies\n", len(s))
		}
		// Proxy status
		pAlive, pUsed, err := database.ProxyStats()
		if err != nil {
			log.Println(err)
		} else {
			fmt.Printf("Proxies:\n\tTotal:\t%d\n\tIn use:\t%d (%.2f%%)\n", pAlive, pUsed, float64(pUsed)/float64(pAlive)*100)
		}
		// Account status
		aTotal, aUsed, aBanned, err := database.AccountStats()
		if err != nil {
			log.Println(err)
		} else {
			fmt.Printf("Accounts:\n\tTotal:\t%d\n\tIn use:\t%d (%.2f%%)\n\tBanned:\t%d (%.2f%%)\n", aTotal, aUsed, float64(aUsed)/float64(aTotal)*100, aBanned, float64(aBanned)/float64(aTotal)*100)
		}
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
	// Add accounts
	if *addAccounts {
		// Read file
		bytes, err := ioutil.ReadFile(*accountsFile)
		if err != nil {
			log.Fatal(err)
		}
		var lines []string
		if strings.Contains(string(bytes), "\r\n") {
			lines = strings.Split(string(bytes), "\r\n")
		} else {
			lines = strings.Split(string(bytes), "\n")
		}
		// Get accounts from file
		accounts := make([]opm.Account, 0)
		for _, l := range lines {
			split := strings.Split(l, ":")
			if len(split) == 2 && split[0] != "false" {
				accounts = append(accounts, opm.Account{Username: split[0], Password: split[1], Provider: "ptc", Used: false, Banned: false})
				database.AddAccount(opm.Account{Username: split[0], Password: split[1], Provider: "ptc"})
			}
		}
		fmt.Printf("Added %d accounts\n", len(accounts))
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
		req, _ := http.NewRequest("GET", fmt.Sprintf("%s?secret=%s", *statusPage, *secret), nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Println(err)
			return
		}
		var status []opm.StatusEntry
		err = json.NewDecoder(resp.Body).Decode(&status)
		if err != nil {
			log.Println(err)
			return
		}

		count, err := database.Cleanup(status)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Printf("Updated %d database entries\n", count)
	}

}
