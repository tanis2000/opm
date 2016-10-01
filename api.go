package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/femot/openmap-tools/opm"
)

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
	// Process request
	var object opm.MapObject
	switch format {
	case "pgm":
		var pgmMessage PGMWebhookFormat
		err = json.NewDecoder(r.Body).Decode(&pgmMessage)
		if err != nil || pgmMessage.Type != "pokemon" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		object = pgmMessage.MapObject()
	default:
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// Add source information
	object.Source = keyString
	// Add to database
	log.Printf("Adding Pokemon %d from %s\n", object.PokemonId, object.Source)
	database.AddMapObject(object)
	// Write response
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "<3")
}
