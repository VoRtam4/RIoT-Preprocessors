/**
 * @File: resolve_trip_details_handler.go
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
	"sort"
)

type TripDetailResponse struct {
	RouteShortName string   `json:"route_short_name"`
	FromStopName   string   `json:"from_stop_name"`
	ToStopName     string   `json:"to_stop_name"`
	DepartureTime  string   `json:"departure_time"`
	Days           []string `json:"days"`
}

func resolveTripDetailsHandler(w http.ResponseWriter, r *http.Request) {
	hashID := r.URL.Query().Get("hash_id") // nově očekáváme hash_id
	if hashID == "" {
		http.Error(w, "Missing hash_id parameter", http.StatusBadRequest)
		return
	}

	tripID := resolveTripIDFromHash(hashID)
	if tripID == "" {
		http.Error(w, "No trip_id found for given hash_id", http.StatusNotFound)
		return
	}

	trip, exists := trips[tripID]
	if !exists {
		http.Error(w, "Trip not found", http.StatusNotFound)
		return
	}

	// Najdi trasu (číslo linky).
	var routeShortName string
	for _, route := range routes {
		if route.RouteID == trip.RouteID {
			routeShortName = route.RouteShortName
			break
		}
	}

	// Získej zastávky pro tento trip.
	stopTimeEntries, ok := stopTimes[tripID]
	if !ok || len(stopTimeEntries) == 0 {
		http.Error(w, "StopTimes not found for trip", http.StatusNotFound)
		return
	}

	// Seřazení podle pořadí zastávky.
	sort.Slice(stopTimeEntries, func(i, j int) bool {
		return stopTimeEntries[i].StopSequence < stopTimeEntries[j].StopSequence
	})

	fromStopID := stopTimeEntries[0].StopID
	toStopID := stopTimeEntries[len(stopTimeEntries)-1].StopID
	departureTime := stopTimeEntries[0].DepartureTime

	fromStopName := resolveStopName(fromStopID)
	toStopName := resolveStopName(toStopID)

	// Načtení dnů platnosti služby z calendar.txt.
	calendarEntries, err := readCalendar("static_data/calendar.txt")
	if err != nil {
		http.Error(w, "Failed to read calendar.txt", http.StatusInternalServerError)
		return
	}

	var days []string
	for _, entry := range calendarEntries {
		if entry.ServiceID == trip.ServiceID {
			if entry.Monday == "1" {
				days = append(days, "Po")
			}
			if entry.Tuesday == "1" {
				days = append(days, "Út")
			}
			if entry.Wednesday == "1" {
				days = append(days, "St")
			}
			if entry.Thursday == "1" {
				days = append(days, "Čt")
			}
			if entry.Friday == "1" {
				days = append(days, "Pá")
			}
			if entry.Saturday == "1" {
				days = append(days, "So")
			}
			if entry.Sunday == "1" {
				days = append(days, "Ne")
			}
			break
		}
	}

	// Výstupní JSON odpověď.
	response := TripDetailResponse{
		RouteShortName: routeShortName,
		FromStopName:   fromStopName,
		ToStopName:     toStopName,
		DepartureTime:  departureTime,
		Days:           days,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// resolveStopName převádí stop_id na čitelný název zastávky.
func resolveStopName(stopID string) string {
	for _, stop := range allStops {
		if stop.StopID == stopID {
			return stop.StopName
		}
	}
	return stopID // fallback, pokud jméno není známo
}

// resolveTripIDFromHash načte trip_id odpovídající hash_id z trip_key_map.csv.
func resolveTripIDFromHash(hashID string) string {
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
		if record[0] == hashID {
			return record[1] // vrátí trip_id
		}
	}
	return ""
}

// Registrace handleru.
func init() {
	http.HandleFunc("/gtfs/resolve-trip-details", resolveTripDetailsHandler)
	fmt.Println("/gtfs/resolve-trip-details endpoint registered")
}
