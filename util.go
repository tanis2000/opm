package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

type Settings struct {
	ListenAddr string // Listen address for http
	DbName     string // Name of the db
	DbHost     string // Host of the db
	DbUser     string
	DbPassword string
}

func loadSettings() (Settings, error) {
	// Try to find system settings file
	bytes, err := ioutil.ReadFile("/etc/opm/api.json")
	if err != nil {
		// Use local config
		bytes, err = ioutil.ReadFile("config.json")
		if err != nil {
			return Settings{}, err
		}
	}
	// Unmarshal json
	var settings Settings
	err = json.Unmarshal(bytes, &settings)
	if err != nil {
		return settings, err
	}
	return settings, err
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
