package opm

import (
	"encoding/json"
	"math/rand"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestLoadSettingsFakePath(t *testing.T) {
	// Load default settings by providing fake path
	settings := LoadSettings("pidgeyfinder.io")
	if settings != DefaultSettings {
		t.Error("LoadSettings(\"pidgeyfinder.io\") => failed to load default settings")
	}
}

func TestLoadSettingsFromFile(t *testing.T) {
	// Create some random settings
	rand.Seed(time.Now().Unix())
	settings := DefaultSettings
	val := reflect.ValueOf(&settings).Elem()
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		rnd := rand.Int63n(666)
		switch field.Kind() {
		case reflect.Int:
			field.SetInt(rnd)
		case reflect.String:
			field.SetString(strconv.FormatInt(rnd, 2))
		}
	}
	// Write them to a temp file
	tempPath := "temp.settings"
	f, err := os.Create(tempPath)
	defer func() {
		f.Close()
		os.Remove(tempPath)
	}()
	if err != nil {
		t.Errorf("Failed to create temp settings file: \"%s\"", err)
		t.FailNow()
	}
	if json.NewEncoder(f).Encode(settings) != nil {
		t.Errorf("Failed to write to temp settings file: \"%s\"", err)
		t.FailNow()
	}
	// And read it back + compare
	if settings != LoadSettings(tempPath) {
		t.Errorf("LoadSettings(tempPath) => failed to read back correct settings")
	}
}

func TestLoadSettingsFromEnv(t *testing.T) {
	// Set env vars to random values
	rand.Seed(time.Now().Unix())
	settings := DefaultSettings
	val := reflect.ValueOf(&settings).Elem()
	s := reflect.TypeOf(settings)
	for i := 0; i < s.NumField(); i++ {
		field := val.Field(i)
		rnd := rand.Int63n(666)
		os.Setenv("OPM"+strings.ToUpper(s.Field(i).Name), strconv.FormatInt(rnd, 10))
		switch field.Kind() {
		case reflect.Int:
			field.SetInt(rnd)
		case reflect.String:
			field.SetString(strconv.FormatInt(rnd, 10))
		}
	}
	// LoadSettings with any param should return with env values
	loaded := LoadSettings("")
	if settings != loaded {
		t.Errorf("LoadSettings(\"\") => expected:\n%s\ngot:\n%s", toString(settings), toString(loaded))
	}
}

func toString(v interface{}) string {
	b, _ := json.MarshalIndent(v, "", " ")
	return string(b)
}
