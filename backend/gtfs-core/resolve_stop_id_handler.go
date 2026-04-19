/**
 * @File: resolve_stop_id_handler.go
 * @Author: Dominik Vondruška
 * @Project: Bakalářská práce — Systém pro monitorování otevřených dat v reálném čase
 * @Description: 
 */

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Funkce resolveStopIDHandler zpracovává HTTP požadavek a vrací seznam všech stop_id odpovídajících zadanému base_stop_id.
func resolveStopIDHandler(w http.ResponseWriter, r *http.Request) {

	baseID := r.URL.Query().Get("base_stop_id")
	if baseID == "" {
		http.Error(w, "Missing base_stop_id parameter", http.StatusBadRequest)
		return
	}

	fmt.Printf("resolveStopIDHandler called with base_stop_id = %s\n", baseID)
	fmt.Printf("stopTimes loaded: %d trips\n", len(stopTimes))
	


	fmt.Printf("Requested base_stop_id: %s\n", baseID)
	foundStops := map[string]bool{}
	totalStops := 0
	matches := 0

	for _, sts := range stopTimes {
		for _, st := range sts {
			totalStops++
			if strings.HasPrefix(st.StopID, baseID) {
				foundStops[st.StopID] = true
				matches++
			}
		}
	}
	fmt.Printf("Iterated over %d stopTimes, found %d matches\n", totalStops, matches)

	uniqueStops := []string{}
	for stop := range foundStops {
		uniqueStops = append(uniqueStops, stop)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(uniqueStops)
}

// Registrace handleru při spuštění.
func init() {
	http.HandleFunc("/gtfs/resolve-stop-id", resolveStopIDHandler)
	fmt.Println("/gtfs/resolve-stop-id endpoint registered")
}
