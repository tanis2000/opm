package db

import (
	"errors"
	"log"
	"time"

	"github.com/femot/openmap-tools/opm"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type OpenMapDb struct {
	mongoSession *mgo.Session
	DbName       string
	DbHost       string
}

type proxy struct {
	Id   int64
	Use  bool
	Dead bool
}

type location struct {
	Type        string
	Coordinates []float64
}

type object struct {
	Type         int
	PokemonId    int
	SpawnpointId string
	Id           string
	Loc          location
	Expiry       int64
	Lured        bool
	Team         int
	Source       string
}

// NewOpenMapDb creates a new connection to
func NewOpenMapDb(dbName, dbHost, user, password string) (*OpenMapDb, error) {
	db := &OpenMapDb{DbName: dbName, DbHost: dbHost}
	s, err := mgo.Dial(db.DbHost)
	if err != nil {
		return db, err
	}
	db.mongoSession = s
	if user != "" && password != "" {
		err = db.mongoSession.DB(db.DbName).Login(user, password)
		if err != nil {
			return db, err
		}
	}
	err = db.ensureIndex()
	return db, err
}

func (db *OpenMapDb) ensureIndex() error {
	err := db.mongoSession.DB("OpenPogoMap").C("Objects").EnsureIndex(mgo.Index{Key: []string{"$2dsphere:loc"}})
	if err != nil {
		return err
	}
	err = db.mongoSession.DB("OpenPogoMap").C("Objects").EnsureIndex(mgo.Index{Key: []string{"id"}, Unique: true, DropDups: true})
	if err != nil {
		return err
	}
	err = db.mongoSession.DB("OpenPogoMap").C("Accounts").EnsureIndex(mgo.Index{Key: []string{"username"}, Unique: true, DropDups: true})
	if err != nil {
		return err
	}
	err = db.mongoSession.DB("OpenPogoMap").C("Keys").EnsureIndex(mgo.Index{Key: []string{"key"}, Unique: true, DropDups: true})
	if err != nil {
		return err
	}
	return db.mongoSession.DB(db.DbName).C("Proxy").EnsureIndex(mgo.Index{Key: []string{"id"}, Unique: true, DropDups: true})
}

func (db *OpenMapDb) Login(user, password string) error {
	return db.mongoSession.DB(db.DbName).Login(user, password)
}

// Cleanup updates the use status of all proxies/accounts based on the input status entries
func (db *OpenMapDb) Cleanup(list []opm.StatusEntry) (int, error) {
	// Get usernames and proxy ids
	usernames := make([]string, len(list))
	proxies := make([]int64, len(list))
	for i, v := range list {
		usernames[i] = v.AccountName
		proxies[i] = v.ProxyId
	}
	// Update accounts
	inAcc := bson.M{
		"username": bson.M{
			"$in": usernames,
		},
	}
	ninAcc := bson.M{
		"username": bson.M{
			"$nin": usernames,
		},
	}
	total := 0
	change, err := db.mongoSession.DB(db.DbName).C("Accounts").UpdateAll(inAcc, bson.M{
		"$set": bson.M{
			"used": true,
		},
	})
	if err != nil {
		return total, err
	}
	total += change.Updated
	change, err = db.mongoSession.DB(db.DbName).C("Accounts").UpdateAll(ninAcc, bson.M{
		"$set": bson.M{
			"used": false,
		},
	})
	if err != nil {
		return total, err
	}
	total += change.Updated
	// Update proxies
	inProxies := bson.M{
		"id": bson.M{
			"$in": proxies,
		},
	}
	ninProxies := bson.M{
		"id": bson.M{
			"$nin": proxies,
		},
	}
	change, err = db.mongoSession.DB(db.DbName).C("Proxy").UpdateAll(inProxies, bson.M{
		"$set": bson.M{
			"use": true,
		},
	})
	if err != nil {
		return total, err
	}
	total += change.Updated
	change, err = db.mongoSession.DB(db.DbName).C("Proxy").UpdateAll(ninProxies, bson.M{
		"$set": bson.M{
			"use": false,
		},
	})
	if err != nil {
		return total, err
	}
	total += change.Updated

	return total, nil

}

// MapObjectStats returns stats about MapObjects
func (db *OpenMapDb) MapObjectStats() (int, int, int, int) {
	c := db.mongoSession.DB(db.DbName).C("Objects")
	totalPokemon, _ := c.Find(bson.M{"type": opm.POKEMON}).Count()
	alivePokemon, _ := c.Find(bson.M{
		"type": opm.POKEMON,
		"expiry": bson.M{
			"$gt": time.Now().Unix(),
		},
	}).Count()
	gyms, _ := c.Find(bson.M{"type": opm.GYM}).Count()
	pokestops, _ := c.Find(bson.M{"type": opm.POKESTOP}).Count()
	return totalPokemon, alivePokemon, gyms, pokestops
}

// AddPokemon adds a pokemon to the db
func (db *OpenMapDb) AddPokemon(p opm.Pokemon) error {
	o := object{
		Type:      opm.POKEMON,
		PokemonId: p.PokemonId,
		Id:        p.EncounterId,
		Expiry:    p.DisappearTime,
		Loc: location{
			Type:        "Point",
			Coordinates: []float64{p.Lng, p.Lat},
		},
	}
	return db.mongoSession.DB(db.DbName).C("Objects").Insert(o)
}

// AddPokestop adds a pokestop to the db
func (db *OpenMapDb) AddPokestop(ps opm.Pokestop) {
	o := object{
		Type:  opm.POKESTOP,
		Id:    ps.Id,
		Lured: ps.Lured,
		Loc: location{
			Type:        "Point",
			Coordinates: []float64{ps.Lng, ps.Lat},
		},
	}
	db.mongoSession.DB(db.DbName).C("Objects").Insert(o)
}

// AddGym adds a gym to the db
func (db *OpenMapDb) AddGym(g opm.Gym) {
	o := object{
		Type: opm.GYM,
		Id:   g.Id,
		Team: g.Team,
		Loc: location{
			Type:        "Point",
			Coordinates: []float64{g.Lng, g.Lat},
		},
	}
	db.mongoSession.DB(db.DbName).C("Objects").Insert(o)
}

// AddMapObject adds a opm.MapObject to the db
func (db *OpenMapDb) AddMapObject(m opm.MapObject) {
	o := object{
		Type:         m.Type,
		PokemonId:    m.PokemonId,
		SpawnpointId: m.SpawnpointId,
		Id:           m.Id,
		Loc: location{
			Type:        "Point",
			Coordinates: []float64{m.Lng, m.Lat},
		},
		Expiry: m.Expiry,
		Team:   m.Team,
		Source: m.Source,
	}
	if o.Type != opm.POKEMON {
		db.mongoSession.DB(db.DbName).C("Objects").Upsert(bson.M{"id": o.Id}, o)
	} else {
		db.mongoSession.DB(db.DbName).C("Objects").Insert(o)
	}
}

func (db *OpenMapDb) AddMapObjects(m []opm.MapObject) {
	for _, o := range m {
		db.AddMapObject(o)
	}
}

// GetMapObjects returns all objects within a radius (in meters) of the given lat/lng
func (db *OpenMapDb) GetMapObjects(lat, lng float64, types []int, radius int) ([]opm.MapObject, error) {
	// Build query
	q := bson.M{
		"loc": bson.M{
			"$near": bson.M{
				"$geometry": bson.M{
					"type":        "Point",
					"coordinates": []float64{lng, lat}},
				"$maxDistance": radius,
			},
		},
		"$or": []bson.M{
			{"expiry": bson.M{"$gt": time.Now().Unix()}},
			{"expiry": 0},
		},
		"type": bson.M{"$in": types},
	}
	// Query db
	var objects []object
	err := db.mongoSession.DB("OpenPogoMap").C("Objects").Find(q).All(&objects)
	if err != nil {
		return nil, err
	}
	// Convert objects to opm.MapObjects
	mapObjects := make([]opm.MapObject, len(objects))
	for i, o := range objects {
		// Cast coordinates
		mapObjects[i] = opm.MapObject{
			Type:      o.Type,
			PokemonId: o.PokemonId,
			Id:        o.Id,
			Lat:       o.Loc.Coordinates[1],
			Lng:       o.Loc.Coordinates[0],
			Expiry:    o.Expiry,
			Team:      o.Team,
		}
	}
	return mapObjects, nil
}

// RemoveOldPokemon removes all Pokemon that expire before the given unix timestamp.
// It will return the count of removed Pokemon and an error, if removal was not successful.
func (db *OpenMapDb) RemoveOldPokemon(threshold int64) (int, error) {
	filter := bson.M{
		"expiry": bson.M{
			"$lt": threshold,
		},
		"type": opm.POKEMON,
	}
	change, err := db.mongoSession.DB(db.DbName).C("Objects").RemoveAll(filter)
	if err != nil {
		return 0, err
	}
	return change.Removed, nil
}

// MarkAccountsAsUnused sets the used flag for all accounts in the database to false
func (db *OpenMapDb) MarkAccountsAsUnused() (int, error) {
	change, err := db.mongoSession.DB(db.DbName).C("Accounts").UpdateAll(bson.M{"used": true}, bson.M{"$set": bson.M{"used": false}})
	if err != nil {
		return -1, err
	}
	return change.Updated, nil
}

// AccountStats returns total, used and banned number of accounts (in that order)
func (db *OpenMapDb) AccountStats() (int, int, int, error) {
	c := db.mongoSession.DB(db.DbName).C("Accounts")
	total, err := c.Count()
	if err != nil {
		return 0, 0, 0, err
	}
	used, err := c.Find(bson.M{"used": true, "banned": false}).Count()
	if err != nil {
		return 0, 0, 0, err
	}
	banned, err := c.Find(bson.M{"banned": true}).Count()
	return total, used, banned, err
}

// GetBannedAccounts returns all accounts that are flagged as banned from the db
func (db *OpenMapDb) GetBannedAccounts() ([]opm.Account, error) {
	var accounts []opm.Account
	err := db.mongoSession.DB(db.DbName).C("Accounts").Find(bson.M{"banned": true}).All(&accounts)
	return accounts, err
}

// GetAccount tries to get an account from the db that is neither in use, nor banned
func (db *OpenMapDb) GetAccount() (opm.Account, error) {
	// Get account from db
	var a opm.Account
	err := db.mongoSession.DB(db.DbName).C("Accounts").Find(bson.M{"used": false, "banned": false}).One(&a)
	if err != nil {
		return opm.Account{}, err
	}
	// Mark account as used
	db_col := bson.M{"username": a.Username}
	a.Used = true
	err = db.mongoSession.DB(db.DbName).C("Accounts").Update(db_col, a)
	if err != nil {
		log.Println(err)
	}
	// Return account
	return a, nil
}

// ReturnAccount puts an Account back in the db and marks it as not used
func (db *OpenMapDb) ReturnAccount(a opm.Account) {
	db_col := bson.M{"username": a.Username}
	a.Used = false
	db.mongoSession.DB(db.DbName).C("Accounts").Update(db_col, a)
}

// AddAccount adds an Account to the database
func (db *OpenMapDb) AddAccount(a opm.Account) {
	db.mongoSession.DB("OpenPogoMap").C("Accounts").Insert(a)
}

// UpdateAccount updates the account information in the database
func (db *OpenMapDb) UpdateAccount(a opm.Account) {
	db.mongoSession.DB(db.DbName).C("Accounts").Update(bson.M{"username": a.Username}, a)
}

// MarkProxiesAsUnused sets the used flag for all accounts in the database to false
func (db *OpenMapDb) MarkProxiesAsUnused() (int, error) {
	change, err := db.mongoSession.DB(db.DbName).C("Proxy").UpdateAll(bson.M{"use": true}, bson.M{"$set": bson.M{"use": false}})
	if err != nil {
		return -1, err
	}
	return change.Updated, nil
}

// AddProxy adds a new proxy to the database
func (db *OpenMapDb) AddProxy(p opm.Proxy) error {
	return db.mongoSession.DB(db.DbName).C("Proxy").Insert(p)
}

// UpdateProxy updates a proxy in the database
func (db *OpenMapDb) UpdateProxy(p opm.Proxy) error {
	_, err := db.mongoSession.DB(db.DbName).C("Proxy").Upsert(bson.M{"id": p.Id}, p)
	return err
}

func (db *OpenMapDb) MaxProxyId() (int64, error) {
	var proxy opm.Proxy
	err := db.mongoSession.DB(db.DbName).C("Proxy").Find(nil).Sort("-id").Limit(1).One(&proxy)
	if err != nil {
		return 0, err
	}
	log.Println(proxy.Id)
	return proxy.Id, err
}

// DropProxies removes ALL proxies from the database
func (db *OpenMapDb) DropProxies() error {
	return db.mongoSession.DB(db.DbName).C("Proxy").DropCollection()
}

// RemoveDeadProxies removes dead proxies from the database
func (db *OpenMapDb) RemoveDeadProxies() (int, error) {
	change, err := db.mongoSession.DB(db.DbName).C("Proxy").RemoveAll(bson.M{"dead": true})
	if err != nil {
		return -1, err
	}
	return change.Removed, nil
}

// ProxyStats returns the number of currently alive/used proxies (in that order)
func (db *OpenMapDb) ProxyStats() (int, int, error) {
	alive, err := db.mongoSession.DB(db.DbName).C("Proxy").Find(bson.M{"dead": false}).Count()
	if err != nil {
		return 0, 0, err
	}
	aliveUsed, err := db.mongoSession.DB(db.DbName).C("Proxy").Find(bson.M{"dead": false, "use": true}).Count()
	return alive, aliveUsed, err
}

// GetProxy gets a new Proxy from the db
func (db *OpenMapDb) GetProxy() (opm.Proxy, error) {
	var p proxy
	err := db.mongoSession.DB(db.DbName).C("Proxy").Find(bson.M{"use": false, "dead": false}).One(&p)
	if err != nil {
		return opm.Proxy{}, errors.New("No proxy available.")
	}
	// Mark proxy as used
	db_col := bson.M{"id": p.Id}
	p.Use = true
	err = db.mongoSession.DB(db.DbName).C("Proxy").Update(db_col, p)
	if err != nil {
		log.Println(err)
	}
	// Return proxy
	return opm.Proxy{Id: p.Id}, nil
}

// ReturnProxy returns a Proxy back to the db and marks it as not used
func (db *OpenMapDb) ReturnProxy(p opm.Proxy) {
	db_col := bson.M{"id": p.Id}
	change := proxy{Id: p.Id, Dead: false, Use: false}
	db.mongoSession.DB(db.DbName).C("Proxy").Update(db_col, change)
}

func (db *OpenMapDb) AddApiKey(k opm.ApiKey) error {
	return db.mongoSession.DB(db.DbName).C("Keys").Insert(k)
}

func (db *OpenMapDb) GetApiKey(k string) (opm.ApiKey, error) {
	var key opm.ApiKey
	err := db.mongoSession.DB(db.DbName).C("Keys").Find(bson.M{"key": k}).One(&key)
	return key, err
}

func (db *OpenMapDb) UpdateApiKey(k opm.ApiKey) error {
	return db.mongoSession.DB(db.DbName).C("Keys").Update(bson.M{"key": k.Key}, k)
}

func (db *OpenMapDb) ApiKeyStats() map[string]int {
	result := make(map[string]int)
	// Get API keys
	var keys []opm.ApiKey
	err := db.mongoSession.DB(db.DbName).C("Keys").Find(nil).All(&keys)
	if err != nil {
		return result
	}
	// Get alive pokemon for all of them
	for _, k := range keys {
		count, _ := db.mongoSession.DB(db.DbName).C("Objects").Find(bson.M{"source": k.Key, "expiry": bson.M{"$gt": time.Now().Unix()}}).Count()
		result[k.Name] = count
	}
	// Return result
	return result
}
