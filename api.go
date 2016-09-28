package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/femot/openmap-tools/opm"
)

func submitHandler(w http.ResponseWriter, r *http.Request) {
	// Process request
	var submission opm.ApiSubmission
	err := json.NewDecoder(r.Body).Decode(&submission)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, err)
		return
	}
	// Check API key
	key, err := database.GetApiKey(submission.Key)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, err)
		return
	}
	if !key.Enabled {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprintln(w, "Key disabled")
		return
	}
	// Add source information
	objects := make([]opm.MapObject, len(submission.MapObjects))
	for i, m := range submission.MapObjects {
		m.Source = key.Key
		objects[i] = m

	}
	// Add to database
	database.AddMapObjects(objects)
	// Write response
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "<3")
}
