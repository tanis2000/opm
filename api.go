package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/femot/gophermon"
	"github.com/femot/pgoapi-go/api"
	"github.com/pogodevorg/POGOProtos-go"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var MongoSess *mgo.Session

func listenAndServe() {
	MongoSess, _ = mgo.Dial("localhost")

	// Setup routes
	http.HandleFunc("/q", requestHandler)
	// Start listening
	log.Fatal(http.ListenAndServe(":8000", nil))
}

type encounter struct {
	EncounterId   string
	PokemonId     int
	Lat           float64
	Lng           float64
	DisappearTime int64
}

type pokestop struct {
	Id    string
	Lat   float64
	Lng   float64
	Lured bool
}

type gym struct {
	Id   string
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

type DbObject struct {
	Type      int
	PokemonId int
	Id        string
	Loc       bson.M
	Expiry    int64
	Lured     bool
	Team      int
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
	// Get trainer from queue
	trainer := dispatcher.GetSession()
	defer dispatcher.QueueSession(trainer)
	log.Printf("Using %s for request", trainer.account.Username)
	// Perform scan
	result, err := getMapResult(trainer, lat, lng)
	// Handle proxy death
	retrySuccess := false
	if err == api.ErrProxyDead {
		var p Proxy
		p, err = dispatcher.GetProxy()
		if err == nil {
			trainer.SetProxy(p)
			// Retry with new proxy
			result, err = getMapResult(trainer, lat, lng)
			retrySuccess = err == nil
		}
	}
	// TODO: Handle account problems

	//Save to db
	for _, pokemon := range result.Encounters {
		location := bson.M{"type": "Point", "coordinates": []float64{pokemon.Lat, pokemon.Lng}}
		obj := DbObject{Type: 1, Id: pokemon.EncounterId, PokemonId: pokemon.PokemonId, Loc: location, Expiry: pokemon.DisappearTime}
		MongoSess.DB("OpenPogoMap").C("Objects").Insert(obj)
	}
	for _, pokestop := range result.Pokestops {
		location := bson.M{"type": "Point", "coordinates": []float64{pokestop.Lat, pokestop.Lng}}
		obj := DbObject{Type: 2, Id: pokestop.Id, Loc: location, Lured: pokestop.Lured}
		MongoSess.DB("OpenPogoMap").C("Objects").Insert(obj)
	}
	for _, gym := range result.Gyms {
		location := bson.M{"type": "Point", "coordinates": []float64{gym.Lat, gym.Lng}}
		obj := DbObject{Type: 3, Id: gym.Id, Loc: location, Team: gym.Team}
		MongoSess.DB("OpenPogoMap").C("Objects").Insert(obj)
	}

	if err != nil {
		errString := err.Error()
		if strings.Contains(errString, "Your username or password is incorrect") {
			// TODO: something wrong here -> remove from db
		}
	}

	// Final error check
	if err != nil && !retrySuccess {
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

func getMapResult(trainer *TrainerSession, lat float64, lng float64) (*mapResult, error) {
	// Set location
	location := &api.Location{Lat: lat, Lon: lng}
	trainer.MoveTo(location)
	// Set accuracy and altitude
	gophermon.SetRandomAccuracy(location)
	// Login trainer
	err := trainer.Login()
	if err != nil {
		return &mapResult{}, err
	}
	// Query api
	<-ticks
	log.Println("Getting player map")
	mapObjects, err := trainer.GetPlayerMap()
	log.Printf("Got player map (Error: %s)\n", err.Error())
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
				encounter{EncounterId: strconv.FormatUint(p.EncounterId, 36), PokemonId: int(p.PokemonData.PokemonId), Lat: p.Latitude, Lng: p.Longitude, DisappearTime: bestBefore})
		}
		// Forts
		for _, f := range c.Forts {
			switch f.Type {
			case protos.FortType_CHECKPOINT:
				result.Pokestops = append(result.Pokestops,
					pokestop{Id: f.Id, Lat: f.Latitude, Lng: f.Longitude, Lured: f.ActiveFortModifier != nil})
			case protos.FortType_GYM:
				result.Gyms = append(result.Gyms,
					gym{Id: f.Id, Lat: f.Latitude, Lng: f.Longitude, Team: int(f.OwnedByTeam)})
			}
		}
	}
	return result
}
