package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"time"

	"github.com/femot/opm/opm"
)

var ErrBusy = errors.New("All our minions are busy")
var ErrTimeout = errors.New("Scan timed out")
var abuseCounter = make(map[string]int)

func createScanProxy() (http.Handler, error) {
	targetURL, err := url.Parse(apiSettings.ScannerAddr)
	if err != nil {
		return nil, err
	}
	return httputil.NewSingleHostReverseProxy(targetURL), nil
}

func submitHandler(w http.ResponseWriter, r *http.Request) {
	// Get key and format
	keyString := r.FormValue("key")
	format := r.FormValue("format")
	if keyString == "" || format == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Check API key
	key, err := database.GetAPIKey(keyString)
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
	if _, ok := keyMetrics[key.PublicKey]; !ok {
		keyMetrics[key.PublicKey] = newAPIKeyMetrics(key)
	}
	// Process request
	var object opm.MapObject
	switch format {
	case "pgm":
		var pgmMessage PGMWebhookFormat
		err = json.NewDecoder(r.Body).Decode(&pgmMessage)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			keyMetrics[key.PublicKey].InvalidCounter.Incr(1)
			return
		}
		if pgmMessage.Type != "pokemon" {
			keyMetrics[key.PublicKey].InvalidCounter.Incr(1)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		object = pgmMessage.MapObject()
	default:
		w.WriteHeader(http.StatusBadRequest)
		keyMetrics[key.PublicKey].InvalidCounter.Incr(1)
		return
	}
	// Add source information
	object.Source = keyString
	// Time validation
	if object.Expiry < time.Now().Unix() {
		log.Printf("%s tried to add expired Pokemon. Ignoring..", key.Name)
		keyMetrics[key.PublicKey].ExpiredCounter.Incr(1)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if object.Expiry > time.Now().Add(15*time.Minute).Unix() {
		log.Printf("%s tried to add Pokemon with a TTL of more than 15 minutes. Ignoring..", key.Name)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// Add to database
	keyMetrics[key.PublicKey].PokemonCounter.Incr(1)
	log.Printf("Adding Pokemon %d from %s (%f,%f)\n", object.PokemonID, key.Name, object.Lat, object.Lng)
	database.AddMapObject(object)
	// Write response
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "<3")
}

func cacheHandler(w http.ResponseWriter, r *http.Request) {
	var objects []opm.MapObject
	// Check method
	if r.Method != "POST" {
		writeCacheResponse(w, false, errors.New("Wrong method").Error(), objects)
		return
	}
	// Get Latitude and Longitude
	lat, err := strconv.ParseFloat(r.FormValue("lat"), 64)
	if err != nil {
		writeCacheResponse(w, false, "Wrong format", objects)
		return
	}
	lng, err := strconv.ParseFloat(r.FormValue("lng"), 64)
	if err != nil {
		writeCacheResponse(w, false, "Wrong format", objects)
		return
	}
	// Pokemon/Gym/Pokestop filter
	var filter []int
	if r.FormValue("p") != "" {
		filter = append(filter, opm.POKEMON)
	}
	if r.FormValue("s") != "" {
		filter = append(filter, opm.POKESTOP)
	}
	if r.FormValue("g") != "" {
		filter = append(filter, opm.GYM)
	}
	// If no filter is set -> show everything
	if len(filter) == 0 {
		filter = []int{opm.POKEMON, opm.POKESTOP, opm.GYM}
	}
	// Get objects from db
	objects, err = database.GetMapObjects(lat, lng, filter, apiSettings.CacheRadius)
	if err != nil {
		writeCacheResponse(w, false, "Failed to get MapObjects from DB", objects)
		log.Println(err)
		return
	}
	writeCacheResponse(w, true, "", objects)
}

func writeCacheResponse(w http.ResponseWriter, ok bool, e string, response []opm.MapObject) {
	if !ok {
		apiMetrics.CacheRequestFailsPerMinute.Incr(1)
	}
	writeAPIResopnse(w, ok, e, response)
}

func writeAPIResopnse(w http.ResponseWriter, ok bool, e string, response []opm.MapObject) {
	w.Header().Add("Content-Type", "application/json")

	if e != "" && e != ErrTimeout.Error() && e != ErrBusy.Error() && e != "Wrong format" && e != "Wrong method" && e != "Failed to get MapObjects from DB" {
		e = "Scan failed"
	}

	r := opm.APIResponse{Ok: ok, Error: e, MapObjects: response}
	err := json.NewEncoder(w).Encode(r)
	if err != nil {
		log.Println(err)
	}
}
