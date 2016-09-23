package util

import (
	"context"
	"log"

	"github.com/femot/openmap-tools/opm"
	"github.com/femot/pgoapi-go/api"
	"github.com/femot/pgoapi-go/auth"
	"github.com/pogodevorg/POGOProtos-go"
)

type TrainerSession struct {
	Account    opm.Account
	context    context.Context
	crypto     api.Crypto
	failCount  int
	Feed       api.Feed
	Location   *api.Location
	Proxy      opm.Proxy
	session    *api.Session
	ForceLogin bool
}

func NewTrainerSession(account opm.Account, location *api.Location, feed api.Feed, crypto api.Crypto) *TrainerSession {
	ctx := context.Background()
	return &TrainerSession{
		Account:  account,
		Location: location,
		Feed:     feed,
		session:  &api.Session{},
		context:  ctx,
		crypto:   crypto,
	}
}

// LoadTrainers creates TrainerSessions for a slice of Accounts
func LoadTrainers(accounts []opm.Account, feed api.Feed, crypto api.Crypto) []*TrainerSession {
	trainers := make([]*TrainerSession, 0)
	for _, a := range accounts {
		trainers = append(trainers, NewTrainerSession(a, &api.Location{}, feed, crypto))
	}
	return trainers
}

// Login initializes a (new) session. This can be used to login again, after the session is expired.
func (t *TrainerSession) Login() error {
	if !t.session.IsExpired() && !t.ForceLogin {
		return nil
	}
	t.ForceLogin = false
	provider, err := auth.NewProvider(t.Account.Provider, t.Account.Username, t.Account.Password)
	if err != nil {
		return err
	}
	session := api.NewSession(provider, t.Location, t.Feed, t.crypto, false)
	err = session.Init(t.context, t.Proxy.Id)
	if err != nil {
		return err
	}
	t.session = session
	return nil
}

func (t *TrainerSession) SetProxy(p opm.Proxy) {
	log.Printf("Using proxy %d for %s", p.Id, t.Account.Username)
	t.Proxy = p
}

func (t *TrainerSession) SetAccount(a opm.Account) {
	t.Account = a
}

// Wrap session functions for trainer sessions
func (t *TrainerSession) Announce() (*protos.GetMapObjectsResponse, error) {
	return t.session.Announce(t.context, t.Proxy.Id)
}
func (t *TrainerSession) Call(requests []*protos.Request) (*protos.ResponseEnvelope, error) {
	return t.session.Call(t.context, requests, t.Proxy.Id)
}
func (t *TrainerSession) GetInventory() (*protos.GetInventoryResponse, error) {
	return t.session.GetInventory(t.context, t.Proxy.Id)
}
func (t *TrainerSession) GetPlayer() (*protos.GetPlayerResponse, error) {
	return t.session.GetPlayer(t.context, t.Proxy.Id)
}
func (t *TrainerSession) GetPlayerMap() (*protos.GetMapObjectsResponse, error) {
	return t.session.GetPlayerMap(t.context, t.Proxy.Id)
}
func (t *TrainerSession) MoveTo(location *api.Location) {
	t.Location = location
	t.session.MoveTo(location)
}
