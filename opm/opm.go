package opm

const (
	POKEMON  = 1
	POKESTOP = 2
	GYM      = 3
)

type Account struct {
	Username string
	Password string
	Provider string
	Used     bool
	Banned   bool
}

type Proxy struct {
	Id string
}

type ApiResponse struct {
	Ok         bool
	Error      string
	MapObjects []MapObject
}

type MapObject struct {
	Type      int
	PokemonId int
	Id        string
	Lat       float64
	Lng       float64
	Expiry    int64
	Lured     bool
	Team      int
}

type Pokemon struct {
	EncounterId   string
	PokemonId     int
	Lat           float64
	Lng           float64
	DisappearTime int64
}

type Pokestop struct {
	Id    string
	Lat   float64
	Lng   float64
	Lured bool
}

type Gym struct {
	Id   string
	Lat  float64
	Lng  float64
	Team int
}
