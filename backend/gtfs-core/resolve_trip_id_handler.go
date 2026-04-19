
/**
 * @File: resolve_trip_id_handler.go
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

type TripKey struct {
	TripID       string
	RouteID      string
	Departure    string
	StopSequence []string
}

var tripKeyMap map[string]string

// Funkce readTripKeyMap načte mapování stabilních klíčů jízd na jejich trip_id z CSV souboru.
func readTripKeyMap(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	r := csv.NewReader(file)
	records, err := r.ReadAll()
	if err != nil {
		return nil, err
	}

	keyMap := make(map[string]string)
	for _, row := range records {
		if len(row) != 2 {
			continue
		}
		keyMap[row[0]] = row[1]
	}
	return keyMap, nil
}

// Funkce resolveTripIDHandler zpracovává HTTP požadavek a vrací trip_id podle kombinace linky, zastávek a směru.
func resolveTripIDHandler(w http.ResponseWriter, r *http.Request) {
	routeID := r.URL.Query().Get("route_id")
	fromStop := r.URL.Query().Get("from_stop")
	toStop := r.URL.Query().Get("to_stop")
	direction := r.URL.Query().Get("direction")

	if routeID == "" || fromStop == "" || toStop == "" {
		http.Error(w, "Missing required parameters", http.StatusBadRequest)
		return
	}

	fmt.Printf("DEBUG: Searching trip for route: %s, from: %s, to: %s\n", routeID, fromStop, toStop)

	for tripID, trip := range trips {
		if trip.RouteID != routeID {
			continue
		}

		sts := stopTimes[tripID]
		fromSeq, toSeq := -1, -1

		for _, st := range sts {
			if strings.HasPrefix(st.StopID, fromStop) {
				fromSeq = st.StopSequence
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

			if valid {
				key := generateStableTripKey(trip, sts)

				if realTripID, exists := tripKeyMap[key]; exists {
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(map[string]string{
						"trip_id": realTripID,
					})
					return
				} else {
					fmt.Printf("No match for key: %s\n", key)
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "No matching trip_id found",
	})
}

// Registrace handleru.
func init() {
	var err error
	tripKeyMap, err = readTripKeyMap("trip_key_map.csv")
	if err != nil {
		panic(err)
	}

	http.HandleFunc("/gtfs/resolve-trip-id", resolveTripIDHandler)
	fmt.Println("/gtfs/resolve-trip-id endpoint registered")
}
