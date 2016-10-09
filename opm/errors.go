package opm

import "errors"

var ErrBusy = errors.New("All our minions are busy")
var ErrScanTimeout = errors.New("Scan timed out")
var ErrWrongMethod = errors.New("Wrong method")
var ErrNoProxiesAvailable = errors.New("No proxy available.")
var ErrProxyNotFound = errors.New("Proxy not found")
var ErrTimeout = errors.New("Timeout")
var ErrInvalidWebhook = errors.New("Invalid webhook")
var ErrPokemonExpired = errors.New("Pokemon already expired")
var ErrPokemonFuture = errors.New("Pokemons disappear time too far in the future")
