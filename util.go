package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/femot/openmap-tools/opm"
	"github.com/paulbellamy/ratecounter"
)

// APIMetrics stores metrics for API keys
type APIMetrics map[string]APIKeyMetrics

type metrics struct {
	PokemonPerMinute int64
	InvalidPerMinute int64
	ExpiredPerMinute int64
	Stats            []APIKeyMetricsRaw
}

func (m APIMetrics) String() string {
	var metricList []APIKeyMetricsRaw
	metrics := metrics{}
	for _, v := range m {
		metricList = append(metricList, v.Eval())
		metrics.InvalidPerMinute += v.InvalidCounter.Rate()
		metrics.PokemonPerMinute += v.PokemonCounter.Rate()
		metrics.ExpiredPerMinute += v.ExpiredCounter.Rate()
	}
	metrics.Stats = metricList
	b, _ := json.Marshal(metrics)
	return string(b)
}

// APIKeyMetrics stores metrics about individual API keys
type APIKeyMetrics struct {
	Key            opm.ApiKey
	InvalidCounter *ratecounter.RateCounter
	PokemonCounter *ratecounter.RateCounter
	ExpiredCounter *ratecounter.RateCounter
}

func newAPIKeyMetrics(key opm.ApiKey) APIKeyMetrics {
	return APIKeyMetrics{
		Key:            key,
		InvalidCounter: ratecounter.NewRateCounter(time.Minute),
		PokemonCounter: ratecounter.NewRateCounter(time.Minute),
		ExpiredCounter: ratecounter.NewRateCounter(time.Minute),
	}
}

type APIKeyMetricsRaw struct {
	Key              string
	InvalidPerMinute int64
	PokemonPerMinute int64
}

func (m APIKeyMetrics) Eval() APIKeyMetricsRaw {
	return APIKeyMetricsRaw{
		Key:              m.Key.Name,
		InvalidPerMinute: m.InvalidCounter.Rate(),
		PokemonPerMinute: m.PokemonCounter.Rate(),
	}
}

type settings struct {
	ListenAddr string // Listen address for http
	DbName     string // Name of the db
	DbHost     string // Host of the db
	DbUser     string
	DbPassword string
}

func loadSettings() (settings, error) {
	// Try to find system settings file
	bytes, err := ioutil.ReadFile("/etc/opm/api.json")
	if err != nil {
		// Use local config
		bytes, err = ioutil.ReadFile("config.json")
		if err != nil {
			return settings{}, err
		}
	}
	// Unmarshal json
	var s settings
	err = json.Unmarshal(bytes, &s)
	if err != nil {
		return s, err
	}
	return s, err
}

func handleFuncDecorator(inner func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// Log start time
		start := time.Now()
		// Metadata
		remoteAddr := r.RemoteAddr
		if r.Header.Get("CF-Connecting-IP") != "" {
			remoteAddr = r.Header.Get("CF-Connecting-IP")
		}
		// Check blacklist
		// ACAH headers
		// Handle request
		inner(w, r)
		// Metrics
		dt := time.Since(start)
		// Logging
		log.Printf("%-6s %-10s\t%-15s\t%s", r.Method, r.URL.Path, dt, remoteAddr)
	}
}
