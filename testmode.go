package main

import (
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/femot/pgoapi-go/api"
	"github.com/pogodevorg/POGOProtos-go"
)

type MockSession struct {
	DefaultResponse *protos.GetMapObjectsResponse
}

func (t *MockSession) GetPlayerMap() (*protos.GetMapObjectsResponse, error) {
	return t.DefaultResponse, nil
}

// Only needed for Session interface. Dummy methods
func (t *MockSession) Login() error { return nil }
func (t *MockSession) Announce() (*protos.GetMapObjectsResponse, error) {
	return &protos.GetMapObjectsResponse{}, nil
}
func (t *MockSession) Call(requests []*protos.Request) (*protos.ResponseEnvelope, error) {
	return &protos.ResponseEnvelope{}, nil
}
func (t *MockSession) GetInventory() (*protos.GetInventoryResponse, error) {
	return &protos.GetInventoryResponse{}, nil
}
func (t *MockSession) GetPlayer() (*protos.GetPlayerResponse, error) {
	return &protos.GetPlayerResponse{}, nil
}
func (t *MockSession) MoveTo(location *api.Location) {}

func (t *MockSession) SetProxy(p Proxy) {}

func runTestMode(n int) {
	log.Println("Starting test mode")

	var err error
	// Load settings
	settings, err = loadSettings()
	if err != nil {
		log.Fatal("Could not load settings")
	}
	settings.ApiCallRate = 1
	settings.ScanDelay = 0
	// Init mock response
	f, err := os.Open("dump/mapobjects.json")
	mapObjects := new(protos.GetMapObjectsResponse)
	err = json.NewDecoder(f).Decode(mapObjects)
	if err != nil {
		log.Fatal(err)
	}

	// Create channels
	ticks = make(chan bool)

	// Create mock sessions
	trainers := make([]Session, n)
	for i := range trainers {
		trainers[i] = &MockSession{DefaultResponse: mapObjects}
	}

	// Init dispatcher
	dispatcher = NewDispatcher(time.Millisecond, trainers)
	dispatcher.Start()

	// Start ticker
	go func(d time.Duration) {
		for {
			ticks <- true
			//time.Sleep(d)
		}
	}(time.Duration(settings.ApiCallRate) * time.Millisecond)

	listenAndServe()
}
