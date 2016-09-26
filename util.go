package main

import (
	"encoding/json"
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

func NewScannerMetrics() *ScannerMetrics {
	return &ScannerMetrics{
		BlockedRequestsPerMinute:   ratecounter.NewRateCounter(time.Minute),
		ScansPerMinute:             ratecounter.NewRateCounter(time.Minute),
		ScanFailsPerMinute:         ratecounter.NewRateCounter(time.Minute),
		ScanBusyPerMinute:          ratecounter.NewRateCounter(time.Minute),
		ScanResponseTimesMs:        NewBuffer(256),
		CacheRequestsPerMinute:     ratecounter.NewRateCounter(time.Minute),
		CacheRequestFailsPerMinute: ratecounter.NewRateCounter(time.Minute),
		CacheResponseTimesNs:       NewBuffer(256),
	}
}

type scannerMetricsData struct {
	ScansPerMinute             int64   `json:"scans_per_minute"`
	ScanFailsPerMinute         int64   `json:"scan_fails_per_minute"`
	ScanBusyPerMinute          int64   `json:"scan_busy_per_minute"`
	ScanResponseTimesMs        []int64 `json:"scan_reponse_times_ms"`
	CacheRequestsPerMinute     int64   `json:"cache_requests_per_minute"`
	CacheRequestFailsPerMinute int64   `json:"cache_fails_per_minute"`
	CacheResponseTimesNs       []int64 `json:"cache_response_times_ns"`
}

func (s *ScannerMetrics) String() string {
	data := scannerMetricsData{
		ScansPerMinute:             s.ScanBusyPerMinute.Rate(),
		ScanFailsPerMinute:         s.ScanFailsPerMinute.Rate(),
		ScanBusyPerMinute:          s.ScanBusyPerMinute.Rate(),
		ScanResponseTimesMs:        s.ScanResponseTimesMs.buffer,
		CacheRequestsPerMinute:     s.CacheRequestsPerMinute.Rate(),
		CacheRequestFailsPerMinute: s.CacheRequestFailsPerMinute.Rate(),
		CacheResponseTimesNs:       s.CacheResponseTimesNs.buffer,
	}
	bytes, _ := json.Marshal(data)
	return string(bytes)
}

type RingBuffer struct {
	buffer []int64
	c      chan int64
}

func (r *RingBuffer) Add(value int64) {
	r.c <- value
}

func (r *RingBuffer) Buffer() []int64 {
	var output []int64
	copy(r.buffer, output)
	return output
}

func (r *RingBuffer) String() string {
	data, _ := json.Marshal(r.buffer)
	return string(data)
}

func NewBuffer(length int) *RingBuffer {
	b := &RingBuffer{
		buffer: make([]int64, length),
		c:      make(chan int64),
	}
	// Goroutine for handling Add()
	go func(rb *RingBuffer) {
		i := 0
		for {
			v := <-rb.c
			rb.buffer[i] = v
			i++
			if i > len(rb.buffer) {
				i = 0
			}
		}
	}(b)
	// Return buffer
	return b
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

func handleFuncDecorator(inner func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// Metadata
		remoteAddr := r.RemoteAddr
		if r.Header.Get("CF-Connecting-IP") != "" {
			remoteAddr = r.Header.Get("CF-Connecting-IP")
		}
		// Check blacklist
		if blacklist[remoteAddr] {
			w.WriteHeader(http.StatusForbidden)
			metrics.BlockedRequestsPerMinute.Incr(1)
			return
		}
		// Log start time
		start := time.Now()
		// Handle request
		inner(w, r)
		// Metrics
		dt := time.Since(start)
		if r.URL.Path == "/q" {
			metrics.ScansPerMinute.Incr(1)
			metrics.ScanResponseTimesMs.Add(dt.Nanoseconds() / 1000000)
		} else if r.URL.Path == "/c" {
			metrics.CacheRequestsPerMinute.Incr(1)
			metrics.CacheResponseTimesNs.Add(dt.Nanoseconds())
		}
		// Logging
		if r.Method != "POST" {
			log.Printf("%-6s %-5s\t%-22s\t%s", r.Method, r.URL.Path, remoteAddr, dt)
		} else {
			log.Printf("%-6s %-5s %-20s,%-20s\t%-22s\t%s", r.Method, r.URL.Path, r.FormValue("lat"), r.FormValue("lng"), remoteAddr, dt)
		}
	}
}
