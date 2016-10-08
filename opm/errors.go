package opm

import "errors"

var ErrBusy = errors.New("All our minions are busy")
var ErrScanTimeout = errors.New("Scan timed out")
var ErrWrongMethod = errors.New("Wrong method")
var ErrNoProxiesAvailable = errors.New("No proxy available.")
var ErrProxyNotFound = errors.New("Proxy not found")
var ErrTimeout = errors.New("Timeout")
