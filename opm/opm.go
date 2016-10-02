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
	Id   int64
	Use  bool
	Dead bool
}

type ApiResponse struct {
	Ok         bool
	Error      string
	MapObjects []MapObject
}

type MapObject struct {
	Type         int     `json:"type"`
	PokemonId    int     `json:"pokemonId,omitempty"`
	SpawnpointId string  `json:"-"`
	Id           string  `json:""`
	Lat          float64 `json:"lat"`
	Lng          float64 `json:"lng"`
	Expiry       int64   `json:"expiry,omitempty"`
	Lured        bool    `json:"lured,omitempty"`
	Team         int     `json:"team,omitempty"`
	Source       string  `json:"-"`
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

type StatusEntry struct {
	AccountName string
	ProxyId     int64
}

type ApiKey struct {
	Key      string
	Verified bool
	Enabled  bool
}
