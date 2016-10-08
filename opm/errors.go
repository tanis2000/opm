package opm

import "errors"

var ErrBusy = errors.New("All our minions are busy")
var ErrTimeout = errors.New("Scan timed out")
