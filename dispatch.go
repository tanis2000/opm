package main

import (
	"errors"
	"time"
)

// Dispatcher coordinates distribution of proxies and accounts to sessions
type Dispatcher struct {
	accounts      chan Account
	proxies       chan Proxy
	sessionsIn    chan Session
	sessionsOut   chan Session
	sessionBuffer []Session
	retryDelay    time.Duration
}

func NewDispatcher(retryDelay time.Duration, sessions []Session) *Dispatcher {
	sessionBuffer := sessions
	return &Dispatcher{
		accounts:      make(chan Account),
		proxies:       make(chan Proxy),
		sessionsIn:    make(chan Session),
		sessionsOut:   make(chan Session),
		sessionBuffer: sessionBuffer,
		retryDelay:    retryDelay,
	}
}

// runSessions manages the Session buffer
func (d *Dispatcher) runSessions() {
	for {
		if len(d.sessionBuffer) > 0 {
			select {
			case d.sessionsOut <- d.sessionBuffer[0]:
				d.sessionBuffer = d.sessionBuffer[1:]
			case s := <-d.sessionsIn:
				d.sessionBuffer = append(d.sessionBuffer, s)
			}
		} else {
			s := <-d.sessionsIn
			d.sessionBuffer = append(d.sessionBuffer, s)
		}
	}
}

// runAccounts continuously requests new accounts from the DB
func (d *Dispatcher) runAccounts() {
	for {
		// TODO: inline request
		if a, err := d.requestAccount(); err == nil {
			d.accounts <- a
		} else {
			time.Sleep(d.retryDelay)
		}

	}
}

// runProxies continuously requests new proxies from the DB
func (d *Dispatcher) runProxies() {
	for {
		// TODO: inline request
		if p, err := d.requestProxy(); err == nil {
			d.proxies <- p
		} else {
			time.Sleep(time.Duration(d.retryDelay) * time.Second)
		}
	}
}

// start starts runSessions, runAccounts and runProxies as goroutines
func (d *Dispatcher) Start() {
	go d.runAccounts()
	go d.runProxies()
	go d.runSessions()
}

// RequestAccount tries to get a new Account from DB
func (d *Dispatcher) requestAccount() (Account, error) {
	// TODO: Try to get account from DB

	// Else error
	return Account{}, errors.New("No account available.")
}

// RequestProxy tries to get a new Proxy from DB
func (d *Dispatcher) requestProxy() (Proxy, error) {
	// TODO: Try to get proxy from DB

	// Else error
	return Proxy{}, errors.New("No proxy available.")
}

// GetSession gets a session from the queue
func (d *Dispatcher) GetSession() Session {
	return <-d.sessionsOut
}

// QueueSession returns a session to the queue (nonblocking)
func (d *Dispatcher) QueueSession(s Session) {
	go func(x Session) {
		time.Sleep(time.Duration(settings.ScanDelay) * time.Second)
		d.sessionsIn <- x
	}(s)
}

// AddSession adds a new Session to the dispatcher
func (d *Dispatcher) AddSession(s Session) {
	go func(x Session) {
		d.sessionsIn <- x
	}(s)
}

// GetAccount returns a new Account
func (d *Dispatcher) GetAccount() Account {
	return <-d.accounts
}

// GetProxy returns a new Proxy
func (d *Dispatcher) GetProxy() Proxy {
	// TODO: remove dummy return, when real proxies are ready
	return Proxy{}
	return <-d.proxies
}
