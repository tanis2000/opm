package main

import (
	"encoding/json"
	"expvar"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/paulbellamy/ratecounter"

	"github.com/femot/openmap-tools/opm"
	"github.com/femot/openmap-tools/util"
	"github.com/femot/pgoapi-go/api"
)

type Settings struct {
	Accounts    int    // Number of accounts to load from db
	ListenAddr  string // Listen address for http
	ProxyHost   string // Address of the openmap-proxy
	CacheRadius int    // Radius in meters for getting cached MapObjects
	ScanDelay   int    // Time between scans per account in seconds
	ApiCallRate int    // Time between API calls in milliseconds
	DbName      string // Name of the db
	DbHost      string // Host of the db
	DbUser      string
	DbPassword  string
	Secret      string
	AllowOrigin string
}

type Status map[string]opm.StatusEntry

type ScannerMetrics struct {
	ScansPerMinute             *ratecounter.RateCounter
	ScanFailsPerMinute         *ratecounter.RateCounter
	ScanBusyPerMinute          *ratecounter.RateCounter
	CacheRequestsPerMinute     *ratecounter.RateCounter
	CacheRequestFailsPerMinute *ratecounter.RateCounter
}

var (
	scansPerMinute         = expvar.NewInt("scans_per_minute")
	scanFailsPerMinute     = expvar.NewInt("scan_fails_per_minute")
	scanBusyPerMinute      = expvar.NewInt("scan_busy_busy_per_minute")
	cacheRequestsPerMinute = expvar.NewInt("cache_requests_per_minute")
	cacheFailsPerMinute    = expvar.NewInt("cache_fails_per_minute")
)

func NewScannerMetrics() *ScannerMetrics {
	return &ScannerMetrics{
		ScansPerMinute:             ratecounter.NewRateCounter(time.Minute),
		ScanFailsPerMinute:         ratecounter.NewRateCounter(time.Minute),
		ScanBusyPerMinute:          ratecounter.NewRateCounter(time.Minute),
		CacheRequestsPerMinute:     ratecounter.NewRateCounter(time.Minute),
		CacheRequestFailsPerMinute: ratecounter.NewRateCounter(time.Minute),
	}
}

func loadSettings() (Settings, error) {
	// Read from file
	bytes, err := ioutil.ReadFile("config.json")
	if err != nil {
		return Settings{}, err
	}
	// Unmarshal json
	var settings Settings
	err = json.Unmarshal(bytes, &settings)
	if err != nil {
		return settings, err
	}
	return settings, err
}

func NewTrainerFromDb() (*util.TrainerSession, error) {
	p, err := database.GetProxy()
	if err != nil {
		return &util.TrainerSession{}, ErrBusy
	}
	a, err := database.GetAccount()
	if err != nil {
		database.ReturnProxy(p)
		return &util.TrainerSession{}, ErrBusy
	}
	trainer := util.NewTrainerSession(a, &api.Location{}, feed, crypto)
	trainer.SetProxy(p)
	return trainer, nil
}

func logDecorator(inner func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// Log start time
		start := time.Now()
		// Handle request
		inner(w, r)
		// Metadata
		remoteAddr := r.RemoteAddr
		if r.Header.Get("CF-Connecting-IP") != "" {
			remoteAddr = r.Header.Get("CF-Connecting-IP")
		}
		dt := time.Since(start)
		// Metrics
		if r.URL.Path == "/q" {
			metrics.ScansPerMinute.Incr(1)
			scansPerMinute.Set(metrics.ScansPerMinute.Rate())
		} else if r.URL.Path == "/c" {
			metrics.CacheRequestsPerMinute.Incr(1)
			cacheRequestsPerMinute.Set(metrics.CacheRequestsPerMinute.Rate())
		}
		// Logging
		if r.Method != "POST" {
			log.Printf("%-6s %-5s\t%-22s\t%s", r.Method, r.URL.Path, remoteAddr, dt)
		} else {
			log.Printf("%-6s %-5s %-20s,%-20s\t%-22s\t%s", r.Method, r.URL.Path, r.FormValue("lat"), r.FormValue("lng"), remoteAddr, dt)
		}
	}
}
