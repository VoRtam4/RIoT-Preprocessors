/**
 * @File: resolve_hash_id_handler.go
 * @Author: Dominik Vondruška
 * @Project: Bakalářská práce — Systém pro monitorování otevřených dat v reálném čase
 * @Description: 
 */

package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

// Handler pro /gtfs/resolve-hash-id – dohledá trip_hash podle vstupních parametrů.
func resolveHashIDHandler(w http.ResponseWriter, r *http.Request) {
	// Načtení parametrů z dotazu
	routeID := r.URL.Query().Get("route_id")
	fromStop := r.URL.Query().Get("from_stop")
	toStop := r.URL.Query().Get("to_stop")
	direction := r.URL.Query().Get("direction")
	departureTime := r.URL.Query().Get("departure_time")

	// Kontrola povinných vstupů.
	if routeID == "" || fromStop == "" || toStop == "" || departureTime == "" {
		http.Error(w, "Missing required parameters", http.StatusBadRequest)
		return
	}

	// Debug výpis.
	fmt.Printf("DEBUG: Searching trip_id for route=%s, from=%s, to=%s, time=%s, direction=%s\n",
		routeID, fromStop, toStop, departureTime, direction)

	// Iterace přes všechny tripy.
	for tripID, trip := range trips {
		if trip.RouteID != routeID {
			continue
		}

		sts := stopTimes[tripID]
		fromSeq, toSeq := -1, -1
		actualDeparture := ""

		for _, st := range sts {
			if strings.HasPrefix(st.StopID, fromStop) && fromSeq == -1 {
				fromSeq = st.StopSequence
				actualDeparture = st.DepartureTime
			}
			if strings.HasPrefix(st.StopID, toStop) {
				toSeq = st.StopSequence
			}
		}

		if fromSeq != -1 && toSeq != -1 {
			valid := false
			if direction == "backward" {
				valid = fromSeq > toSeq
			} else {
				valid = fromSeq < toSeq
			}

			if valid && actualDeparture == departureTime {
				// TRIP_ID nalezen, nyní dohledáme hash v CSV.
				hash := findHashForTripID(tripID)
				if hash != "" {
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(map[string]string{
						"trip_hash": hash,
					})
					return
				}

				// Pokud nebyl nalezen hash pro daný trip_id.
				http.Error(w, "Hash not found for resolved trip_id", http.StatusNotFound)
				return
			}
		}
	}

	// Pokud se trip_id nepodařilo najít.
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "No matching trip found",
	})
}

// Funkce najde hash pro dané trip_id v souboru trip_key_map.csv.
func findHashForTripID(tripID string) string {
	file, err := os.Open("trip_key_map.csv")
	if err != nil {
		fmt.Println("ERROR: Cannot open trip_key_map.csv:", err)
		return ""
	}
	defer file.Close()

	reader := csv.NewReader(file)
	for {
		record, err := reader.Read()
		if err != nil {
			break
		}
		if len(record) != 2 {
			continue
		}
		if record[1] == tripID {
			return record[0] // hash je v prvním sloupci
		}
	}
	return ""
}

// Registrace handleru při spuštění.
func init() {
	http.HandleFunc("/gtfs/resolve-hash-id", resolveHashIDHandler)
	fmt.Println("/gtfs/resolve-hash-id endpoint registered")
}
