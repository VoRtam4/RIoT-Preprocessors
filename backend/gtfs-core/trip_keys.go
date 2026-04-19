/**
 * @File: trip_keys.go
 * @Author: Dominik Vondruška
 * @Project: Bakalářská práce — Systém pro monitorování otevřených dat v reálném čase
 * @Description: 
 */

package main

import (
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Odstraní nástupiště ze stop_id (např. "U1429Z1" → "U1429").
func stripPlatformSuffix(stopID string) string {
	if idx := strings.Index(stopID, "Z"); idx > 0 {
		return stopID[:idx]
	}
	return stopID
}

// Vygeneruje stabilní klíč z route_id, času odjezdu a sekvence zastávek.
func generateStableTripKey(trip Trip, stopTimes []StopTime) string {
	sort.Slice(stopTimes, func(i, j int) bool {
		return stopTimes[i].StopSequence < stopTimes[j].StopSequence
	})

	if len(stopTimes) == 0 {
		return ""
	}

	keyStr := trip.RouteID + "|" + stopTimes[0].DepartureTime + "|"
	for i, st := range stopTimes {
		keyStr += st.StopID
		if i < len(stopTimes)-1 {
			keyStr += "_"
		}
	}

	h := sha256.New()
	h.Write([]byte(keyStr))
	return hex.EncodeToString(h.Sum(nil))
}

// Vygeneruje trip_key_map.csv ze souborů trips.txt a stop_times.txt.
func GenerateTripKeyMap(staticDir string) error {
	trips, err := readTrips(filepath.Join(staticDir, "trips.txt"))
	if err != nil {
		return err
	}

	stopTimesByTrip, err := readStopTimes(filepath.Join(staticDir, "stop_times.txt"))
	if err != nil {
		return err
	}

	outFile, err := os.Create("trip_key_map.csv")
	if err != nil {
		return err
	}
	defer outFile.Close()

	writer := csv.NewWriter(outFile)
	defer writer.Flush()

	for tripID, trip := range trips {
		stopTimes := stopTimesByTrip[tripID]
		hash := generateStableTripKey(trip, stopTimes)
		if hash == "" {
			continue
		}
		writer.Write([]string{hash, tripID})
	}

	return nil
}
