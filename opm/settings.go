package opm

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"reflect"
	"strconv"
	"strings"
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
func LoadSettings(settingsFile string) Settings {
	settings := DefaultSettings
	// Use default file, if no file is specified
	if settingsFile == "" {
		settingsFile = "/etc/opm/opm.json"
	}
	// Read file
	bytes, err := ioutil.ReadFile(settingsFile)
	if err == nil {
		// Parse json
		err = json.Unmarshal(bytes, &settings)
	}
	// Get environment vars
	LoadStructFromEnv(&settings)
	return settings
}

// LoadStructFromEnv sets struct fields to the value of environment variables with the same name.
//	The environment variables must be in all caps and add the prefix "OPM" to the field's name.
//	If the environment variable is not set, the field is skipped.
//
//	Supported field types:
//		int, string
func LoadStructFromEnv(v interface{}) error {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Struct {
		return errors.New("Please pass reference to struct")
	}
	elem := val.Elem()
	typeOf := elem.Type()
	for i := 0; i < elem.NumField(); i++ {
		field := elem.Field(i)
		env := os.Getenv("OPM" + strings.ToUpper(typeOf.Field(i).Name))
		if env == "" {
			continue
		}
		switch field.Kind() {
		case reflect.Int:
			intVal, err := strconv.Atoi(env)
			if err == nil {
				field.SetInt(int64(intVal))
			}
		case reflect.String:
			field.SetString(env)
		}
	}
	return nil
}
