package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"time"

	"github.com/femot/pgoapi-go/api"
	"github.com/femot/pgoapi-go/auth"
	"github.com/pogodevorg/POGOProtos-go"
)

type Settings struct {
	Accounts    []Account
	ListenAddr  string
	GmapsKey    string
	ScanDelay   int // Time between scans per account in seconds
	ApiCallRate int // Time between API calls in milliseconds
}

type Account struct {
	Username string
	Password string
	Provider string
}

type Proxy struct {
	Id string
}

func loadSettings() (Settings, error) {
	bytes, err := ioutil.ReadFile("config.json")
	if err != nil {
		return Settings{}, err
	}
	var settings Settings
	err = json.Unmarshal(bytes, &settings)
	return settings, err
}

type Session interface {
	Login() error
	Announce() (*protos.GetMapObjectsResponse, error)
	Call(requests []*protos.Request) (*protos.ResponseEnvelope, error)
	GetInventory() (*protos.GetInventoryResponse, error)
	GetPlayer() (*protos.GetPlayerResponse, error)
	GetPlayerMap() (*protos.GetMapObjectsResponse, error)
	MoveTo(location *api.Location)
}

type TrainerSession struct {
	account   Account
	context   context.Context
	crypto    api.Crypto
	failCount int
	feed      api.Feed
	location  *api.Location
	session   *api.Session
}

func NewTrainerSession(account Account, location *api.Location, feed api.Feed, crypto api.Crypto) *TrainerSession {
	ctx := context.Background()
	return &TrainerSession{
		account:  account,
		location: location,
		feed:     feed,
		session:  &api.Session{},
		context:  ctx,
		crypto:   crypto,
	}
}

// LoadTrainers creates TrainerSessions for a slice of Accounts
func LoadTrainers(accounts []Account, feed api.Feed, crypto api.Crypto) []*TrainerSession {
	trainers := make([]*TrainerSession, 0)
	for _, a := range accounts {
		trainers = append(trainers, NewTrainerSession(a, &api.Location{}, feed, crypto))
	}
	return trainers
}

// Login initializes a (new) session. This can be used to login again, after the session is expired.
func (t *TrainerSession) Login() error {
	if !t.session.IsExpired() {
		return nil
	}
	provider, err := auth.NewProvider(t.account.Provider, t.account.Username, t.account.Password)
	if err != nil {
		return err
	}
	session := api.NewSession(provider, t.location, t.feed, t.crypto, false)
	err = session.Init(t.context)
	if err != nil {
		return err
	}
	t.session = session
	return nil
}

// Wrap session functions for trainer sessions
func (t *TrainerSession) Announce() (*protos.GetMapObjectsResponse, error) {
	return t.session.Announce(t.context)
}
func (t *TrainerSession) Call(requests []*protos.Request) (*protos.ResponseEnvelope, error) {
	return t.session.Call(t.context, requests)
}
func (t *TrainerSession) GetInventory() (*protos.GetInventoryResponse, error) {
	return t.session.GetInventory(t.context)
}
func (t *TrainerSession) GetPlayer() (*protos.GetPlayerResponse, error) {
	return t.session.GetPlayer(t.context)
}
func (t *TrainerSession) GetPlayerMap() (*protos.GetMapObjectsResponse, error) {
	return t.session.GetPlayerMap(t.context)
}
func (t *TrainerSession) MoveTo(location *api.Location) {
	t.location = location
	t.session.MoveTo(location)
}

func getTrainer() Session {
	t := <-trainerQueue
	return t
}

func queueTrainer(s Session) {
	// Trainer will have to wait 10s before he can accept the next call. Wrap it in goroutine to not block the caller.
	go func(x Session) {
		time.Sleep(time.Duration(settings.ScanDelay) * time.Second)
		trainerQueue <- x
	}(s)
}
