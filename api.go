package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/femot/openmap-tools/opm"
	"github.com/femot/openmap-tools/util"
	"github.com/femot/pgoapi-go/api"
	"github.com/pogodevorg/POGOProtos-go"
)

func listenAndServe() {
	// Setup routes
	http.HandleFunc("/q", requestHandler)
	http.HandleFunc("/c", cacheHandler)
	// Start listening
	log.Fatal(http.ListenAndServe(":8000", nil))
}

type ApiResponse struct {
	Ok       bool
	Error    string
	Response []opm.MapObject
}

func cacheHandler(w http.ResponseWriter, r *http.Request) {
	var objects []opm.MapObject
	// Check method
	if r.Method != "POST" {
		writeApiResponse(w, false, errors.New("Wrong method").Error(), objects)
		return
	}
	// Get Latitude and Longitude
	lat, err := strconv.ParseFloat(r.FormValue("lat"), 64)
	if err != nil {
		writeApiResponse(w, false, err.Error(), objects)
		return
	}
	lng, err := strconv.ParseFloat(r.FormValue("lng"), 64)
	if err != nil {
		writeApiResponse(w, false, err.Error(), objects)
	}
	// Get objects from db
	objects, err = settings.Db.GetMapObjects(lat, lng, 400)
	if err != nil {
		writeApiResponse(w, false, "Query failed", objects)
		log.Println(err)
		return
	}
	writeApiResponse(w, true, "", objects)
}

func requestHandler(w http.ResponseWriter, r *http.Request) {
	// Check method
	if r.Method != "POST" {
		writeApiResponse(w, false, errors.New("Wrong method").Error(), nil)
		return
	}
	// Get Latitude and Longitude
	lat, err := strconv.ParseFloat(r.FormValue("lat"), 64)
	if err != nil {
		writeApiResponse(w, false, err.Error(), nil)
		return
	}
	lng, err := strconv.ParseFloat(r.FormValue("lng"), 64)
	if err != nil {
		writeApiResponse(w, false, err.Error(), nil)
	}
	// Get trainer from queue
	trainer := trainerQueue.Get()
	defer trainerQueue.Queue(trainer, time.Duration(settings.ScanDelay)*time.Second)
	log.Printf("Using %s for request (%f,%f)", trainer.Account.Username, lat, lng)
	// Perform scan
	result, err := getMapResult(trainer, lat, lng)
	// Error handling
	retrySuccess := false
	// Handle proxy death
	if err != nil && err.Error() == api.ErrProxyDead.Error() {
		var p opm.Proxy
		p, err = settings.Db.GetProxy()
		if err == nil {
			trainer.SetProxy(p)
			// Retry with new proxy
			result, err = getMapResult(trainer, lat, lng)
			retrySuccess = err == nil
		}
	}
	// Account problems
	if err != nil {
		errString := err.Error()
		if strings.Contains(errString, "Your username or password is incorrect") || err == api.ErrAccountBanned {
			trainer.Account.Banned = true
		}
	}
	// Just retry when this error comes
	if err == api.ErrInvalidPlatformRequest {
		result, err = getMapResult(trainer, lat, lng)
	}
	// Final error check
	if err != nil && !retrySuccess {
		writeApiResponse(w, false, err.Error(), nil)
		return
	}
	//Save to db
	for _, o := range result {
		settings.Db.AddMapObject(o)
	}
	writeApiResponse(w, true, "", result)
}

func writeApiResponse(w http.ResponseWriter, ok bool, error string, response []opm.MapObject) {
	r := ApiResponse{Ok: ok, Error: error, Response: response}
	err := json.NewEncoder(w).Encode(r)
	if err != nil {
		log.Println(err)
	}
}

func getMapResult(trainer *util.TrainerSession, lat float64, lng float64) ([]opm.MapObject, error) {
	// Set location
	location := &api.Location{Lat: lat, Lon: lng}
	trainer.MoveTo(location)
	// Login trainer
	err := trainer.Login()
	if err == api.ErrInvalidAuthToken {
		trainer.ForceLogin = true
		err = trainer.Login()
	}
	if err != nil {
		if err != api.ErrProxyDead {
			log.Printf("Login error (%s):\n\t\t%s\n", trainer.Account.Username, err.Error())
		}
		return nil, err
	}
	// Query api
	<-ticks
	mapObjects, err := trainer.GetPlayerMap()
	if err != nil && err != api.ErrNewRPCURL {
		if err != api.ErrProxyDead {
			log.Printf("Error getting map objects (%s):\n\t\t%s\n", trainer.Account.Username, err.Error())
		}
		return nil, err
	}
	// Parse and return result
	return parseMapObjects(mapObjects), nil
}

func parseMapObjects(r *protos.GetMapObjectsResponse) []opm.MapObject {
	objects := make([]opm.MapObject, 0)
	// Cells
	for _, c := range r.MapCells {
		// Pokemon
		for _, p := range c.WildPokemons {
			tth := p.TimeTillHiddenMs
			bestBefore := time.Now().Add(time.Duration(tth) * time.Millisecond).Unix()
			objects = append(objects, opm.MapObject{
				Id:        strconv.FormatUint(p.EncounterId, 36),
				PokemonId: int(p.PokemonData.PokemonId),
				Lat:       p.Latitude,
				Lng:       p.Longitude,
				Expiry:    bestBefore,
			})
		}
		// Forts
		for _, f := range c.Forts {
			switch f.Type {
			case protos.FortType_CHECKPOINT:
				objects = append(objects, opm.MapObject{
					Type:  opm.POKESTOP,
					Id:    f.Id,
					Lat:   f.Latitude,
					Lng:   f.Longitude,
					Lured: f.ActiveFortModifier != nil,
				})
			case protos.FortType_GYM:
				objects = append(objects, opm.MapObject{
					Type: opm.GYM,
					Id:   f.Id,
					Lat:  f.Latitude,
					Lng:  f.Longitude,
					Team: int(f.OwnedByTeam),
				})
			}
		}
	}
	return objects
}
