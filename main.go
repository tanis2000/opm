package main

import (
	"flag"
	"fmt"

	"github.com/femot/openmap-tools/db"
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
}
