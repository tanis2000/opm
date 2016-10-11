package opm

import "testing"

func TestLoadSettings(t *testing.T) {
	// Load default settings by providing fake path
	settings := LoadSettings("pidgeyfinder.io")
	if settings != DefaultSettings {
		t.Error("LoadSettings(\"pidgeyfinder.io\") => failed to load default settings")
	}
}
