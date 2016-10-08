package opm

import (
	"encoding/json"
	"io/ioutil"
)

// DefaultSettings are the default value for Settings
var DefaultSettings = Settings{
	DbHost:               "localhost",
	DbName:               "OPM",
	APIListenAddress:     ":80",
	ProxyListenAddress:   ":8080",
	ProxyWSListenAddress: ":8081",
	ScannerListenAddress: ":8100",
	StatsListenAddress:   ":8200",
}

// Settings is a struct for storing OPM settings that are relevant for most packages
type Settings struct {
	// General
	Secret string
	// DB
	DbHost     string
	DbName     string
	DbUser     string
	DbPassword string
	// Listen addresses
	APIListenAddress     string
	ProxyListenAddress   string
	ProxyWSListenAddress string
	ScannerListenAddress string
	StatsListenAddress   string
}

// LoadSettings parses the content of the provided settings file as json
func LoadSettings(settingsFile string) (Settings, error) {
	settings := DefaultSettings
	// Use default file, if no file is specified
	if settingsFile == "" {
		settingsFile = "/etc/opm/opm.json"
	}
	// Read file
	bytes, err := ioutil.ReadFile(settingsFile)
	if err != nil {
		return settings, err
	}
	// Parse json
	err = json.Unmarshal(bytes, &settings)
	// Return result
	return settings, err
}
