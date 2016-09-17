package main

import (
	"errors"
	"log"
	"strconv"
	"time"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// Dispatcher coordinates distribution of proxies and accounts to sessions
type Dispatcher struct {
	accounts      chan Account
	proxies       chan Proxy
	sessionsIn    chan *TrainerSession
	sessionsOut   chan *TrainerSession
	sessionBuffer []*TrainerSession
	retryDelay    time.Duration
	mongoSession  *mgo.Session
}

type ProxyDB struct {
	Id   int
	Use  bool
	Dead bool
}

func NewDispatcher(retryDelay time.Duration) *Dispatcher {
	//TODO Add mongo url in config and check error
	mongo, err := mgo.Dial("localhost")
	if err != nil {
		log.Print("Mongo error!")
	}

	d := &Dispatcher{
		accounts:      make(chan Account),
		proxies:       make(chan Proxy),
		sessionsIn:    make(chan *TrainerSession),
		sessionsOut:   make(chan *TrainerSession),
		sessionBuffer: nil,
		retryDelay:    retryDelay,
		mongoSession:  mongo,
	}

	// Load accounts from DB
	accounts := make([]Account, 0)

	for i := 0; i < settings.Accounts; i++ {
		if a, err := d.GetAccount(); err == nil {
			accounts = append(accounts, a)
		} else {
			log.Fatal("Not enough accounts")
		}
	}

	// Load sessions
	trainers := LoadTrainers(accounts, feed, crypto)
	d.sessionBuffer = trainers

	// Load proxies
	for _, t := range trainers {
		if p, err := d.GetProxy(); err == nil {
			t.SetProxy(p)
		} else {
			t.SetProxy(Proxy{Id: "0"})
		}
	}
	return d
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

// start starts runSessions, runAccounts and runProxies as goroutines
func (d *Dispatcher) Start() {
	go d.runAccounts()
	go d.runSessions()
}

// RequestAccount tries to get a new Account from DB
func (d *Dispatcher) requestAccount() (Account, error) {
	// TODO: Try to get account from DB

	// Else error
	return Account{}, errors.New("No account available.")
}

// GetSession gets a session from the queue
func (d *Dispatcher) GetSession() *TrainerSession {
	return <-d.sessionsOut
}

// QueueSession returns a session to the queue (nonblocking)
func (d *Dispatcher) QueueSession(s *TrainerSession) {
	go func(x *TrainerSession) {
		time.Sleep(time.Duration(settings.ScanDelay) * time.Second)
		d.sessionsIn <- x
	}(s)
}

// AddSession adds a new Session to the dispatcher
func (d *Dispatcher) AddSession(s *TrainerSession) {
	go func(x *TrainerSession) {
		d.sessionsIn <- x
	}(s)
}

// GetAccount returns a new Account
func (d *Dispatcher) GetAccount() (Account, error) {
	// Get account from db
	var a Account
	err := d.mongoSession.DB("OpenPogoMap").C("Accounts").Find(bson.M{"used": false, "banned": false}).One(&a)
	if err != nil {
		return Account{}, err
	}
	// Mark account as used
	db_col := bson.M{"username": a.Username}
	a.Used = true
	err = d.mongoSession.DB("OpenPogoMap").C("Accounts").Update(db_col, a)
	if err != nil {
		log.Println(err)
	}
	// Return account
	return a, nil
}

// GetProxy returns a new Proxy
func (d *Dispatcher) GetProxy() (Proxy, error) {
	var proxy ProxyDB
	err := d.mongoSession.DB("OpenPogoMap").C("Proxy").Find(bson.M{"dead": false, "use": false}).Select(bson.M{"use": false}).One(&proxy)
	if err != nil {
		return Proxy{}, errors.New("No proxy available.")
	}
	// Mark proxy as used
	db_col := bson.M{"id": proxy.Id}
	change := ProxyDB{Id: proxy.Id, Dead: false, Use: true}
	d.mongoSession.DB("OpenPogoMap").C("Proxy").Update(db_col, change)

	return Proxy{strconv.Itoa(proxy.Id)}, nil
}
