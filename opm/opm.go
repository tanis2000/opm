package opm

// MapObject types
const (
	POKEMON  = 1
	POKESTOP = 2
	GYM      = 3
)

// Account represents a PGO account
type Account struct {
	Username       string
	Password       string
	Provider       string
	Used           bool
	Banned         bool
	CaptchaFlagged bool
}

// Proxy represents a proxy that is connected to the hub
type Proxy struct {
	ID   int64
	Use  bool
	Dead bool
}

// APIResponse represents a response sent back to the requesting client
// This response type is used for cache and scan requests.
type APIResponse struct {
	Ok         bool
	Error      string
	MapObjects []MapObject
}

// MapObject represents an object on the map (Pokemon, Gym or Pokestop)
type MapObject struct {
	Type         int     `json:"type"`
	PokemonID    int     `json:"pokemonID,omitempty"`
	SpawnpointID string  `json:"-"`
	ID           string  `json:"id"`
	Lat          float64 `json:"lat"`
	Lng          float64 `json:"lng"`
	Expiry       int64   `json:"expiry,omitempty"`
	Lured        bool    `json:"lured,omitempty"`
	Team         int     `json:"team,omitempty"`
	Source       string  `json:"source,omitempty"`
}

// Pokemon represents a Pokemon MapObject
type Pokemon struct {
	EncounterID   string
	PokemonID     int
	Lat           float64
	Lng           float64
	DisappearTime int64
}

// Pokestop represents a Pokestop MapObject
type Pokestop struct {
	ID    string
	Lat   float64
	Lng   float64
	Lured bool
}

// Gym represents a Gym MapObject
type Gym struct {
	ID   string
	Lat  float64
	Lng  float64
	Team int
}

// StatusEntry represents a key-value pair for account names and proxy IDs
// This is used by the scanner to report accounts/proxies in use
type StatusEntry struct {
	AccountName string
	ProxyId     int64
}

// APIKey is used for for managing ingress/egress via API
type APIKey struct {
	PrivateKey string
	PublicKey  string
	Name       string
	URL        string
	Verified   bool
	Enabled    bool
}
