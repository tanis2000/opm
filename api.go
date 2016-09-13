package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/femot/gophermon"
	"github.com/femot/pgoapi-go/api"
	"github.com/pogodevorg/POGOProtos-go"
)

func listenAndServe() {
	// Setup routes
	http.HandleFunc("/q", requestHandler)
	// Start listening
	log.Fatal(http.ListenAndServe(settings.ListenAddr, nil))
}

type encounter struct {
	PokemonId     int
	Lat           float64
	Lng           float64
	DisappearTime int64
}

type pokestop struct {
	Lat   float64
	Lng   float64
	Lured bool
}

type gym struct {
	Lat  float64
	Lng  float64
	Team int
}

type mapResult struct {
	Encounters []encounter
	Pokestops  []pokestop
	Gyms       []gym
}

type ApiResponse struct {
	Ok       bool
	Error    string
	Response *mapResult
}

func requestHandler(w http.ResponseWriter, r *http.Request) {
	// Check method
	if r.Method != "POST" {
		writeApiResponse(w, false, errors.New("Wrong method").Error(), &mapResult{})
		return
	}
	// Get Latitude and Longitude
	lat, err := strconv.ParseFloat(r.FormValue("lat"), 64)
	if err != nil {
		writeApiResponse(w, false, err.Error(), &mapResult{})
		return
	}
	lng, err := strconv.ParseFloat(r.FormValue("lng"), 64)
	if err != nil {
		writeApiResponse(w, false, err.Error(), &mapResult{})
		return
	}
	// Perform scan
	result, err := getMapResult(lat, lng)
	if err != nil {
		writeApiResponse(w, false, err.Error(), &mapResult{})
		return
	}

	writeApiResponse(w, true, "", result)
}

func writeApiResponse(w http.ResponseWriter, ok bool, error string, response *mapResult) {
	r := ApiResponse{Ok: ok, Error: error, Response: response}
	err := json.NewEncoder(w).Encode(r)
	if err != nil {
		log.Println(err)
	}
}

func getMapResult(lat float64, lng float64) (*mapResult, error) {
	// Get trainer from queue
	trainer := dispatcher.GetSession()
	defer dispatcher.QueueSession(trainer)
	// Set location
	location := &api.Location{Lat: lat, Lon: lng}
	trainer.MoveTo(location)
	// Set accuracy and altitude
	gophermon.SetRandomAccuracy(location)
	err := gophermon.SetCorrectAltitudes([]*api.Location{location}, settings.GmapsKey)
	if err != nil {
		return &mapResult{}, err
	}
	// Login trainer
	err = trainer.Login()
	if err != nil {
		return &mapResult{}, err
	}
	// Query api
	<-ticks
	mapObjects, err := trainer.GetPlayerMap()
	// Handle proxy death
	if err == api.ErrProxyDead {
		// Get new proxy
		p, e := dispatcher.RequestProxy()
		if e != nil {
			return &mapResult{}, e
		}
		trainer.SetProxy(p)
		// Retry with new proxy
		mapObjects, err = trainer.GetPlayerMap()
	}
	if err != nil && err != api.ErrNewRPCURL {
		return &mapResult{}, err
	}
	// Parse and return result
	return parseMapObjects(mapObjects), nil
}

func parseMapObjects(r *protos.GetMapObjectsResponse) *mapResult {
	result := new(mapResult)
	// Cells
	for _, c := range r.MapCells {
		// Pokemon
		for _, p := range c.WildPokemons {
			tth := p.TimeTillHiddenMs
			bestBefore := time.Now().Add(time.Duration(tth) * time.Millisecond).Unix()
			result.Encounters = append(result.Encounters,
				encounter{PokemonId: int(p.PokemonData.PokemonId), Lat: p.Latitude, Lng: p.Longitude, DisappearTime: bestBefore})
		}
		// Forts
		for _, f := range c.Forts {
			switch f.Type {
			case protos.FortType_CHECKPOINT:
				result.Pokestops = append(result.Pokestops,
					pokestop{Lat: f.Latitude, Lng: f.Longitude, Lured: f.ActiveFortModifier != nil})
			case protos.FortType_GYM:
				result.Gyms = append(result.Gyms,
					gym{Lat: f.Latitude, Lng: f.Longitude, Team: int(f.OwnedByTeam)})
			}
		}
	}
	return result
}
