package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/context"

	"github.com/femot/pgoapi-go/api"
	"github.com/pogodevorg/POGOProtos-go"
	"github.com/pogointel/opm/opm"
	"github.com/pogointel/opm/util"
)

var checkRequest = func(r *http.Request) bool { return true }

func listenAndServe() {
	// Setup routes
	mux := http.NewServeMux()
	mux.HandleFunc("/status", statusHandler)
	mux.HandleFunc("/scan", requestHandler)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.Handle("/debug/vars", http.DefaultServeMux)

	// Start listening
	s := &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 20 * time.Second,
		Addr:         fmt.Sprintf(":%d", opmSettings.ScannerListenPort),
		Handler:      mux,
	}
	log.Fatal(s.ListenAndServe())
}

func requestHandler(w http.ResponseWriter, r *http.Request) {
	// Create a context
	ctx, cancel := context.WithTimeout(context.Background(), opm.RequestTimeout*time.Second)
	defer cancel()
	// Check method
	if r.Method != "POST" {
		writeScanResponse(w, false, opm.ErrWrongMethod.Error(), nil)
		return
	}
	// Get Latitude and Longitude
	lat, err := strconv.ParseFloat(r.FormValue("lat"), 64)
	if err != nil {
		writeScanResponse(w, false, err.Error(), nil)
		return
	}
	lng, err := strconv.ParseFloat(r.FormValue("lng"), 64)
	if err != nil {
		writeScanResponse(w, false, err.Error(), nil)
		return
	}
	log.Printf("Scanning %f, %f", lat, lng)
	// Mock mode
	if scannerSettings.MockMode {
		mockObject := opm.MapObject{Type: opm.POKEMON, Expiry: time.Now().Add(10 * time.Minute).Unix()}
		mockObject.Source = "mock"

		rand.Seed(time.Now().Unix())
		mockObject.PokemonID = rand.Intn(150)
		randomId := rand.Int63n(372036854775807)
		mockObject.ID = strconv.FormatInt(randomId, 36)
		mockObject.Lat, mockObject.Lng = util.LatLngOffset(lat, lng, 0.02)
		mapObjects := []opm.MapObject{mockObject}
		b, _ := json.Marshal(mockObject)
		log.Printf("Sending mock object: %s", string(b))
		writeScanResponse(w, true, "", mapObjects)
		return
	}
	// Get trainer from queue
	trainer, err := trainerQueue.Get(5 * time.Second)
	if err != nil {
		// Timeout -> try setup a new one
		p, err := database.GetProxy()
		if err != nil {
			writeScanResponse(w, false, opm.ErrBusy.Error(), nil)
			return
		}
		a, err := database.GetAccount()
		if err != nil {
			database.ReturnProxy(p)
			writeScanResponse(w, false, opm.ErrBusy.Error(), nil)
			return
		}
		trainer = util.NewTrainerSession(a, &api.Location{}, feed, crypto)
		trainer.SetProxy(p)
		scannerStatus[trainer.Account.Username] = opm.StatusEntry{AccountName: trainer.Account.Username, ProxyId: trainer.Proxy.ID}
	}
	defer trainerQueue.Queue(trainer, time.Duration(scannerSettings.ScanDelay)*time.Second)
	trainer.Context = ctx
	// Perform scan
	mapObjects, err := getMapResult(trainer, lat, lng)
	// Error handling
	retrySuccess := false
	// Check error/timeout
	if err != nil && ctx.Err() != nil {
		writeScanResponse(w, false, opm.ErrScanTimeout.Error(), mapObjects)
		return
	}
	// Handle proxy death
	if err != nil && err == api.ErrProxyDead {
		trainer.Proxy.Dead = true
		var p opm.Proxy
		p, err = database.GetProxy()
		if err == nil {
			trainer.SetProxy(p)
			scannerStatus[trainer.Account.Username] = opm.StatusEntry{AccountName: trainer.Account.Username, ProxyId: trainer.Proxy.ID}
			// Retry with new proxy
			mapObjects, err = getMapResult(trainer, lat, lng)
			retrySuccess = err == nil
		} else {
			delete(scannerStatus, trainer.Account.Username)
			database.ReturnAccount(trainer.Account)
			log.Println("No proxies available")
			writeScanResponse(w, false, opm.ErrBusy.Error(), nil)
			return
		}
	}
	// Account problems
	if err != nil {
		errString := err.Error()
		if strings.Contains(errString, "Your username or password is incorrect") || err == api.ErrAccountBanned || err.Error() == "Empty response" || strings.Contains(errString, "not yet active") {
			log.Printf("Account %s banned", trainer.Account.Username)
			trainer.Account.Banned = true
			database.UpdateAccount(trainer.Account)
			delete(scannerStatus, trainer.Account.Username)
		} else if err == api.ErrCheckChallenge {
			log.Printf("Account %s flagged for Challenge", trainer.Account.Username)
			trainer.Account.CaptchaFlagged = true
			database.UpdateAccount(trainer.Account)
			delete(scannerStatus, trainer.Account.Username)
		}
	}
	// Just retry when this error comes
	if err == api.ErrInvalidPlatformRequest {
		mapObjects, err = getMapResult(trainer, lat, lng)
	}
	// Final error check
	if err != nil && !retrySuccess {
		writeScanResponse(w, false, err.Error(), nil)
		return
	}
	//Save to db
	for _, o := range mapObjects {
		database.AddMapObject(o)
	}
	writeScanResponse(w, true, "", mapObjects)
}

func writeScanResponse(w http.ResponseWriter, ok bool, e string, response []opm.MapObject) {
	if !ok {
		log.Println(e)
		if e == opm.ErrBusy.Error() {
			scannerMetrics.ScanBusyPerMinute.Incr(1)
		} else {
			scannerMetrics.ScanFailsPerMinute.Incr(1)
		}
	}
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

func getMapResult(trainer *util.TrainerSession, lat float64, lng float64) ([]opm.MapObject, error) {
	// Set location
	trainer.MoveTo(&api.Location{Lat: lat, Lon: lng})
	// Login trainer
	if !trainer.IsLoggedIn() {
		select {
		case <-loginTicks:
		case <-trainer.Context.Done():
			return nil, opm.ErrScanTimeout
		}
		err := trainer.Login()
		if err == api.ErrInvalidAuthToken {
			trainer.ForceLogin = true
			select {
			case <-loginTicks:
			case <-trainer.Context.Done():
				return nil, opm.ErrScanTimeout
			}
			err = trainer.Login()
		}
		if err != nil {
			if err != api.ErrProxyDead {
				log.Printf("Login error (%s): %s\n", trainer.Account.Username, err.Error())
			}
			return nil, err
		}
	}
	// Query api
	<-ticks
	mapObjects, err := trainer.GetPlayerMap()
	if err != nil && err != api.ErrNewRPCURL {
		if err != api.ErrProxyDead {
			log.Printf("Error getting map objects (%s): %s\n", trainer.Account.Username, err.Error())
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
			expiry := time.Now().Add(time.Duration(p.TimeTillHiddenMs) * time.Millisecond).Unix()
			if expiry > time.Now().Add(15*time.Minute).Unix() {
				continue
			}
			objects = append(objects, opm.MapObject{
				Type:         opm.POKEMON,
				ID:           strconv.FormatUint(p.EncounterId, 36),
				PokemonID:    int(p.PokemonData.PokemonId),
				SpawnpointID: p.SpawnPointId,
				Lat:          p.Latitude,
				Lng:          p.Longitude,
				Expiry:       expiry,
			})
		}
		// Forts
		for _, f := range c.Forts {
			switch f.Type {
			case protos.FortType_CHECKPOINT:
				if f.LureInfo != nil {
					// Lured pokemon found!
					objects = append(objects, opm.MapObject{
						Type:      opm.POKEMON,
						ID:        strconv.FormatUint(f.LureInfo.EncounterId, 36),
						PokemonID: int(f.LureInfo.ActivePokemonId),
						Lat:       f.Latitude,
						Lng:       f.Longitude,
						Expiry:    f.LureInfo.LureExpiresTimestampMs / 1000,
					})
				}
				objects = append(objects, opm.MapObject{
					Type:  opm.POKESTOP,
					ID:    f.Id,
					Lat:   f.Latitude,
					Lng:   f.Longitude,
					Lured: f.ActiveFortModifier != nil,
				})
			case protos.FortType_GYM:
				objects = append(objects, opm.MapObject{
					Type: opm.GYM,
					ID:   f.Id,
					Lat:  f.Latitude,
					Lng:  f.Longitude,
					Team: int(f.OwnedByTeam),
				})
			}
		}
	}
	return objects
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	if r.FormValue("secret") != opmSettings.Secret {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, "nope")
		return
	}

	list := make([]opm.StatusEntry, 0)
	for _, v := range scannerStatus {
		list = append(list, v)
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}
