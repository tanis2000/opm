package main

import "github.com/femot/openmap-tools/opm"

// PGMWebhookFormat is the format for incoming webhooks from PokemonGo-Map
type PGMWebhookFormat struct {
	Message struct {
		EncounterID   string  `json:"encounter_id"`
		PokemonID     int     `json:"pokemon_id"`
		DisappearTime int64   `json:"disappear_time"`
		Lat           float64 `json:"latitude"`
		Lng           float64 `json:"longitude"`
	} `json:"message"`
	Type string `json:"type"`
}

// MapObject converts a PGMPokemonMessage to a opm.MapObject
func (p PGMWebhookFormat) MapObject() opm.MapObject {
	return opm.MapObject{
		Type:      opm.POKEMON,
		Id:        p.Message.EncounterID,
		PokemonId: p.Message.PokemonID,
		Expiry:    p.Message.DisappearTime,
		Lat:       p.Message.Lat,
		Lng:       p.Message.Lng,
	}
}
