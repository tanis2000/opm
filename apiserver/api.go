package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/pogointel/opm/opm"
)

var abuseCounter = make(map[string]int)

func objectFromWebhook(format string, r *http.Request) (opm.MapObject, error) {
	var object opm.MapObject
	switch format {
	case "pgm":
		// PokemonGo-Map format
		var pgmMessage PGMWebhookFormat
		err := json.NewDecoder(r.Body).Decode(&pgmMessage)
		if err != nil || pgmMessage.Type != "pokemon" {
			return object, opm.ErrInvalidWebhook
		}
		object = pgmMessage.MapObject()
	default:
		return object, opm.ErrInvalidWebhook
	}
	return object, nil
}

func validateMapObject(object opm.MapObject, key opm.APIKey) error {
	if object.Expiry < time.Now().Unix() {
		return opm.ErrPokemonExpired
	}
	if object.Expiry > time.Now().Add(15*time.Minute).Unix() {
		return opm.ErrPokemonFuture
	}
	return nil
}
