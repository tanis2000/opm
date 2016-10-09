package opm

import (
	"encoding/json"
	"io/ioutil"
)

// DefaultSettings are the default value for Settings
var DefaultSettings = Settings{
	AllowOrigin:          "*",
	CacheRadius:          1000,
	DbHost:               "localhost",
	DbName:               "OPM",
	APIListenAddress:     "localhost",
	APIListenPort:        80,
	ProxyListenAddress:   "localhost",
	ProxyListenPort:      8080,
	ProxyWSListenAddress: "localhost",
	ProxyWSListenPort:    8081,
	ScannerListenAddress: "localhost",
	ScannerListenPort:    8100,
	StatsListenAddress:   "localhost",
	StatsListenPort:      8200,
}

// Settings is a struct for storing OPM settings that are relevant for most packages
type Settings struct {
	// Security
	Secret      string
	AllowOrigin string
	// General
	CacheRadius int
	// DB
	DbHost     string
	DbName     string
	DbUser     string
	DbPassword string
	// Listen addresses
	APIListenAddress     string
	APIListenPort        int
	ProxyListenAddress   string
	ProxyListenPort      int
	ProxyWSListenAddress string
	ProxyWSListenPort    int
	ScannerListenAddress string
	ScannerListenPort    int
	StatsListenAddress   string
	StatsListenPort      int
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
