/**
 * @File: stable_trip_key_handler.go
 * @Author: Dominik Vondruška
 * @Project: Bakalářská práce — Systém pro monitorování otevřených dat v reálném čase
 * @Description: 
 */

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
)

// Funkce stableTripKeyHandler zpracovává HTTP požadavek a vrací stabilní hashový klíč (stable_trip_key) pro nalezený trip podle linky, zastávek, dne a času odjezdu.
func stableTripKeyHandler(w http.ResponseWriter, r *http.Request) {
	routeID := r.URL.Query().Get("route_id")
	fromStop := strings.TrimSpace(r.URL.Query().Get("from_stop"))
	toStop := strings.TrimSpace(r.URL.Query().Get("to_stop"))
	day := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("day")))
	time := strings.TrimSpace(r.URL.Query().Get("departure_time"))

	if routeID == "" || fromStop == "" || toStop == "" || day == "" || time == "" {
		http.Error(w, "Missing required parameters", http.StatusBadRequest)
		return
	}

	// Najde aktivní service_id pro daný den.
	activeServices := map[string]bool{}
	for _, entry := range calendar {
		var active string
		switch day {
		case "monday":
			active = entry.Monday
		case "tuesday":
			active = entry.Tuesday
		case "wednesday":
			active = entry.Wednesday
		case "thursday":
			active = entry.Thursday
		case "friday":
			active = entry.Friday
		case "saturday":
			active = entry.Saturday
		case "sunday":
			active = entry.Sunday
		}
		if active == "1" {
			activeServices[entry.ServiceID] = true
		}
	}

	// Hledání odpovídajícího tripu.
	for tripID, trip := range trips {
		if trip.RouteID != routeID || !activeServices[trip.ServiceID] {
			continue
		}

		sts := stopTimes[tripID]
		var fromSeq, toSeq int = -1, -1
		var actualDeparture string
		for _, st := range sts {
			if strings.HasPrefix(st.StopID, fromStop) {
				fromSeq = st.StopSequence
				actualDeparture = st.DepartureTime
			}
			if strings.HasPrefix(st.StopID, toStop) {
				toSeq = st.StopSequence
			}
		}

		if fromSeq != -1 && toSeq != -1 && fromSeq < toSeq && actualDeparture == time {
			// Vytvoření StableTripKey.
			sort.Slice(sts, func(i, j int) bool {
				return sts[i].StopSequence < sts[j].StopSequence
			})
			keyStr := trip.RouteID + "|" + time + "|"
			for i, st := range sts {
				keyStr += st.StopID
				if i < len(sts)-1 {
					keyStr += "_"
				}
			}
			h := sha256.New()
			h.Write([]byte(keyStr))
			stableKey := hex.EncodeToString(h.Sum(nil))

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"trip_id":          tripID,
				"stable_trip_key": stableKey,
			})
			return
		}
	}

	// Pokud nic nenalezeno,
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "No matching trip found",
	})
}

// Registrace endpointu.
func init() {
	http.HandleFunc("/gtfs/stable-trip-key", stableTripKeyHandler)
	fmt.Println("/gtfs/stable-trip-key endpoint registered")
}
