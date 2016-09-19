package opm

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
)

type Api struct {
	Key      string
	Endpoint string
	client   *http.Client
}

// NewApi creates a new API object
func NewApi(key, endpoint string) *Api {
	return &Api{
		Key:      key,
		Endpoint: endpoint,
		client:   new(http.Client),
	}
}

// ScanLocation sends a request to the OpenPokeMap API and returns MapObjects
func (a *Api) ScanLocation(lat, lng float64) ([]MapObject, error) {
	// Prepare request
	values := url.Values{
		"lat": {strconv.FormatFloat(lat, 'f', 12, 64)},
		"lng": {strconv.FormatFloat(lng, 'f', 12, 64)},
		"key": {a.Key},
	}
	// Call Api
	return a.call(values, a.Endpoint+"/q")
}

// GetCachedObjects requests already known MapObjects from the OpenPokeMap API
func (a *Api) GetCachedObjects(lat, lng float64, filter []int) ([]MapObject, error) {
	// Prepare request
	values := url.Values{
		"lat": {strconv.FormatFloat(lat, 'f', 12, 64)},
		"lng": {strconv.FormatFloat(lng, 'f', 12, 64)},
		"key": {a.Key},
	}
	// Apply filter
	for _, t := range filter {
		if t == POKEMON {
			values["p"] = []string{strconv.Itoa(POKEMON)}
		} else if t == POKESTOP {
			values["s"] = []string{strconv.Itoa(POKESTOP)}
		} else if t == GYM {
			values["g"] = []string{strconv.Itoa(GYM)}
		}
	}
	// Call Api
	return a.call(values, a.Endpoint+"/c")
}

// call sends the prepared post request to the target url
func (a *Api) call(values url.Values, target string) ([]MapObject, error) {
	// Perform Api call
	r, err := a.client.PostForm(target, values)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	// Decode response
	var response ApiResponse
	err = json.NewDecoder(r.Body).Decode(&response)
	if err != nil {
		return nil, err
	}
	// Check response
	if !response.Ok {
		return nil, errors.New(response.Error)
	}
	// Return MapObjects
	return response.MapObjects, nil
}
