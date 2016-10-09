package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/pogointel/opm/opm"
)

var securityCheck = func(r *http.Request) bool {
	return true
}

func startHTTP() {
	// Routes/Handlers
	mux := http.NewServeMux()
	scanHandler, err := createScanProxy()
	if err != nil {
		log.Fatal(err)
	}
	mux.Handle("/static", http.StripPrefix("/static", http.FileServer(http.Dir("/frontend"))))
	mux.HandleFunc("/q", httpDecorator(scanHandler.ServeHTTP))
	mux.HandleFunc("/c", httpDecorator(cacheHandler))
	mux.HandleFunc("/submit", httpDecorator(submitHandler))
	mux.Handle("/debug/vars", http.DefaultServeMux)
	// Create http server with timeouts
	s := http.Server{
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		Addr:         opmSettings.APIListenAddress,
		Handler:      mux,
	}
	// Run server
	log.Fatal(s.ListenAndServe())
}

func httpDecorator(inner func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// Log start
		start := time.Now()
		// Check if request is ok
		if !securityCheck(r) {
			apiMetrics.SecurityCheckFailsPerMinute.Incr(1)
			w.WriteHeader(http.StatusForbidden)
			return
		}
		// Metadata
		remoteAddr := r.RemoteAddr
		if r.Header.Get("CF-Connecting-IP") != "" {
			remoteAddr = r.Header.Get("CF-Connecting-IP")
		}
		// Check blacklist
		if blacklist[remoteAddr] {
			w.WriteHeader(http.StatusForbidden)
			apiMetrics.BlockedRequestsPerMinute.Incr(1)
			return
		}
		// ACAO
		if opmSettings.AllowOrigin == "*" {
			w.Header().Add("Access-Control-Allow-Origin", opmSettings.AllowOrigin)
		} else {
			origin := r.Header.Get("Origin")
			if origin != "" {
				if strings.HasSuffix(origin, opmSettings.AllowOrigin) {
					w.Header().Add("Access-Control-Allow-Origin", origin)
				}
			}
		}

		// Actually handle request
		inner(w, r)

		// Metrics
		dt := time.Since(start)
		// Log it
		log.Printf("%s", dt)

	}
}

func createScanProxy() (http.Handler, error) {
	targetURL, err := url.Parse(fmt.Sprintf("http://%s:%d", opmSettings.ScannerListenAddress, opmSettings.ScannerListenPort))
	if err != nil {
		return nil, err
	}
	return httputil.NewSingleHostReverseProxy(targetURL), nil
}

func submitHandler(w http.ResponseWriter, r *http.Request) {
	// Helper function for sending http.StatusBadRequest back
	badRequest := func() { w.WriteHeader(http.StatusBadRequest) }
	// Get key and format
	keyString := r.FormValue("key")
	format := r.FormValue("format")
	if keyString == "" || format == "" {
		badRequest()
		return
	}

	// Check API key
	key, err := database.GetAPIKey(keyString)
	if err != nil {
		badRequest()
		fmt.Fprintln(w, "Key not found")
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
	object, err := objectFromWebhook(format, r)
	if err != nil {
		keyMetrics[key.PublicKey].InvalidCounter.Incr(1)
		badRequest()
		return
	}
	// Add source information
	object.Source = key.PublicKey
	// Time validation
	err = validateMapObject(object, key)
	if err != nil {
		if err == opm.ErrPokemonExpired {
			keyMetrics[key.PublicKey].ExpiredCounter.Incr(1)
			badRequest()
			return
		}
		if err == opm.ErrPokemonFuture {
			keyMetrics[key.PublicKey].InvalidCounter.Incr(1)
			badRequest()
			return
		}
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
		writeCacheResponse(w, false, opm.ErrWrongMethod.Error(), objects)
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

func addBlacklist(w http.ResponseWriter, r *http.Request) {
	if r.FormValue("secret") != opmSettings.Secret {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	if r.FormValue("addr") != "" {
		blacklist[r.FormValue("addr")] = true
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, r.FormValue("addr"))
	}
}

func writeCacheResponse(w http.ResponseWriter, ok bool, e string, response []opm.MapObject) {
	if !ok {
		apiMetrics.CacheRequestFailsPerMinute.Incr(1)
	}
	writeAPIResopnse(w, ok, e, response)
}

func writeAPIResopnse(w http.ResponseWriter, ok bool, e string, response []opm.MapObject) {
	w.Header().Add("Content-Type", "application/json")

	if e != "" && e != opm.ErrScanTimeout.Error() && e != opm.ErrBusy.Error() && e != "Wrong format" && e != "Wrong method" && e != "Failed to get MapObjects from DB" {
		e = "Scan failed"
	}

	r := opm.APIResponse{Ok: ok, Error: e, MapObjects: response}
	err := json.NewEncoder(w).Encode(r)
	if err != nil {
		log.Println(err)
	}
}
