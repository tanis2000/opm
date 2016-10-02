package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/femot/openmap-tools/opm"
)

var abuseCounter = make(map[string]int)

func submitHandler(w http.ResponseWriter, r *http.Request) {
	// Get key and format
	keyString := r.FormValue("key")
	format := r.FormValue("format")
	if keyString == "" || format == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Check API key
	key, err := database.GetApiKey(keyString)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, err)
		return
	}
	if !key.Enabled {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprintln(w, "Key disabled")
		return
	}
	// Metrics
	if _, ok := keyMetrics[key.Key]; !ok {
		keyMetrics[key.Key] = newAPIKeyMetrics(key)
	}
	// Process request
	var object opm.MapObject
	switch format {
	case "pgm":
		var pgmMessage PGMWebhookFormat
		err = json.NewDecoder(r.Body).Decode(&pgmMessage)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			keyMetrics[key.Key].InvalidCounter.Incr(1)
			return
		}
		if pgmMessage.Type != "pokemon" {
			keyMetrics[key.Key].InvalidCounter.Incr(1)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		object = pgmMessage.MapObject()
	default:
		w.WriteHeader(http.StatusBadRequest)
		keyMetrics[key.Key].InvalidCounter.Incr(1)
		return
	}
	keyMetrics[key.Key].PokemonCounter.Incr(1)
	// Add source information
	object.Source = keyString
	// Add to database
	log.Printf("Adding Pokemon %d from %s (%f,%f)\n", object.PokemonId, key.Name, object.Lat, object.Lng)
	database.AddMapObject(object)
	// Write response
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "<3")
}
