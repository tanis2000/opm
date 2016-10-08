package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/pogointel/opm/opm"
	"github.com/paulbellamy/ratecounter"
)

type RingBuffer struct {
	buffer []int64
	cap    int
	c      chan int64
}

func (r *RingBuffer) Add(value int64) {
	r.c <- value
}

func (r *RingBuffer) Buffer() []int64 {
	output := make([]int64, len(r.buffer))
	copy(output, r.buffer)
	return output
}

func (r *RingBuffer) String() string {
	data, _ := json.Marshal(r.buffer)
	return string(data)
}

func NewBuffer(cap int) *RingBuffer {
	b := &RingBuffer{
		buffer: make([]int64, 0),
		c:      make(chan int64),
		cap:    cap,
	}
	// Goroutine for handling Add()
	go func(rb *RingBuffer) {
		for {
			v := <-rb.c
			rb.buffer = append(rb.buffer, v)
			if len(rb.buffer) >= rb.cap {
				break
			}
		}
		i := 0
		for {
			v := <-rb.c
			rb.buffer[i] = v
			i++
			if i >= len(rb.buffer) {
				i = 0
			}
		}
	}(b)
	// Return buffer
	return b
}

// APIMetrics stores metrics for API keys
type KeyMetrics map[string]APIKeyMetrics

type APIMetrics struct {
	KeyMetrics KeyMetrics
	// Requests
	BlockedRequestsPerMinute *ratecounter.RateCounter
	// Scans
	ScansPerMinute      *ratecounter.RateCounter
	ScanFailsPerMinute  *ratecounter.RateCounter
	ScanBusyPerMinute   *ratecounter.RateCounter
	ScanResponseTimesMs *RingBuffer
	// Cache
	CacheRequestsPerMinute     *ratecounter.RateCounter
	CacheRequestFailsPerMinute *ratecounter.RateCounter
	CacheResponseTimesNs       *RingBuffer
}

type metrics struct {
	PokemonPerMinute int64
	InvalidPerMinute int64
	ExpiredPerMinute int64
	Stats            []APIKeyMetricsRaw
}

func (m KeyMetrics) String() string {
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
	Key            opm.APIKey
	InvalidCounter *ratecounter.RateCounter
	PokemonCounter *ratecounter.RateCounter
	ExpiredCounter *ratecounter.RateCounter
}

func newAPIKeyMetrics(key opm.APIKey) APIKeyMetrics {
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
	ExpiredPerMinute int64
}

func (m APIKeyMetrics) Eval() APIKeyMetricsRaw {
	return APIKeyMetricsRaw{
		Key:              m.Key.Name,
		InvalidPerMinute: m.InvalidCounter.Rate(),
		PokemonPerMinute: m.PokemonCounter.Rate(),
		ExpiredPerMinute: m.ExpiredCounter.Rate(),
	}
}

type settings struct {
	ListenAddr  string // Listen address for http
	DbName      string // Name of the db
	DbHost      string // Host of the db
	DbUser      string // User for db authentication
	DbPassword  string // Password for db authentication
	CacheRadius int    // Cache radius
	ScannerAddr string // Address of the scanner
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
