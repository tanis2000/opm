package db

import (
	"errors"
	"log"
	"strconv"
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
	Id   int
	Use  bool
	Dead bool
}

type location struct {
	Type        string
	Coordinates []float64
}

type object struct {
	Type      int
	PokemonId int
	Id        string
	Loc       location
	Expiry    int64
	Lured     bool
	Team      int
}

// NewOpenMapDb creates a new connection to
func NewOpenMapDb(dbName, dbHost string) (*OpenMapDb, error) {
	db := &OpenMapDb{DbName: dbName, DbHost: dbHost}
	s, err := mgo.Dial(db.DbHost)
	if err != nil {
		return db, err
	}
	db.mongoSession = s
	err = db.mongoSession.DB("OpenPogoMap").C("Objects").EnsureIndex(mgo.Index{Key: []string{"$2dsphere:loc"}})
	err = db.mongoSession.DB("OpenPogoMap").C("Objects").EnsureIndex(mgo.Index{Key: []string{"id"}, Unique: true, DropDups: true})
	return db, err
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
		Type:      m.Type,
		PokemonId: m.PokemonId,
		Id:        m.Id,
		Loc: location{
			Type:        "Point",
			Coordinates: []float64{m.Lng, m.Lat},
		},
		Expiry: m.Expiry,
		Lured:  m.Lured,
		Team:   m.Team,
	}
	db.mongoSession.DB(db.DbName).C("Objects").Insert(o)
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
	var objects []object
	// Query db
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
			Lured:     o.Lured,
			Team:      o.Team,
		}
	}
	return mapObjects, nil
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

// GetProxy returns a new Proxy
func (db *OpenMapDb) GetProxy() (opm.Proxy, error) {
	var p proxy
	err := db.mongoSession.DB(db.DbName).C("Proxy").Find(bson.M{"dead": false, "use": false}).Select(bson.M{"use": false}).One(&p)
	if err != nil {
		return opm.Proxy{}, errors.New("No proxy available.")
	}
	// Mark proxy as used
	db_col := bson.M{"id": p.Id}
	change := proxy{Id: p.Id, Dead: false, Use: true}
	db.mongoSession.DB(db.DbName).C("Proxy").Update(db_col, change)
	// Return proxy
	return opm.Proxy{Id: strconv.Itoa(p.Id)}, nil
}
