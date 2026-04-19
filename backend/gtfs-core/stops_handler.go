/**
 * @File: stops_handler.go
 * @Author: Dominik Vondruška
 * @Project: Bakalářská práce — Systém pro monitorování otevřených dat v reálném čase
 * @Description: 
 */

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
)

type StopResponse struct {
	StopID       string `json:"stop_id"`       // bez nástupiště
	StopName     string `json:"stop_name"`     // přidáno
	StopSequence int    `json:"stop_sequence"` // zůstává
}

// Funkce stopsHandler zpracovává HTTP požadavek a vrací seznam unikátních zastávek pro danou linku seřazených podle pořadí.
func stopsHandler(w http.ResponseWriter, r *http.Request) {
	routeID := r.URL.Query().Get("route_id")
	if routeID == "" {
		http.Error(w, "Missing route_id parameter", http.StatusBadRequest)
		return
	}

	tripIDs := []string{}
	for _, trip := range trips {
		if strings.EqualFold(trip.RouteID, routeID) {
			tripIDs = append(tripIDs, trip.TripID)
		}
	}

	baseStopMap := make(map[string]StopResponse)
	for _, tripID := range tripIDs {
		sts := stopTimes[tripID]
		for _, st := range sts {
			// Základní stop_id bez nástupiště (např. U15325).
			baseID := stripPlatformSuffix(st.StopID)
			if _, exists := baseStopMap[baseID]; !exists {
				baseStopMap[baseID] = StopResponse{
					StopID:       baseID,
					StopName:     stopNames[baseID],
					StopSequence: st.StopSequence,
				}
			}
		}
	}

	// Převod na slice a seřazení.
	stops := []StopResponse{}
	for _, stop := range baseStopMap {
		stops = append(stops, stop)
	}
	sort.Slice(stops, func(i, j int) bool {
		return stops[i].StopSequence < stops[j].StopSequence
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stops)
}

// Registrace endpointu.
func init() {
	http.HandleFunc("/gtfs/stops", stopsHandler)
	fmt.Println("/gtfs/stops endpoint registered")
}
