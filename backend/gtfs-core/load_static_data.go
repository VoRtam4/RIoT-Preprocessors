/**
 * @File: load_static_data.go
 * @Author: Dominik Vondruška
 * @Project: Bakalářská práce — Systém pro monitorování otevřených dat v reálném čase
 * @Description: Načítání statických GTFS dat do paměti.
 */

package main

import (
	"log"
	"fmt"
	"os"
	"encoding/csv"
)


type StopTime struct {
	TripID        string
	StopID        string
	DepartureTime string
	StopSequence  int
}

type Trip struct {
	TripID    string
	RouteID   string
	ServiceID string
}

type Route struct {
	RouteID        string `json:"route_id"`
	RouteShortName string `json:"route_short_name"`
	RouteType      string `json:"route_type"`
}

type Stop struct {
	StopID   string
	StopName string
}

var trips map[string]Trip
var stopTimes map[string][]StopTime
var routes []Route
var stopNames map[string]string
var allStops []Stop

// Funkce pro načtení záznamů z trips.txt a jejich uložení do mapy podle trip_id.
func readTrips(path string) (map[string]Trip, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1

	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	headers := make(map[string]int)
	for i, h := range records[0] {
		headers[h] = i
	}

	trips := make(map[string]Trip)
	for _, row := range records[1:] {
		trip := Trip{
			TripID:    row[headers["trip_id"]],
			RouteID:   row[headers["route_id"]],
			ServiceID: row[headers["service_id"]],
		}
		trips[trip.TripID] = trip
	}
	return trips, nil
}

// Funkce pro načtení záznamů z stop_times.txt a jejich seskupení podle trip_id.
func readStopTimes(path string) (map[string][]StopTime, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1

	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	headers := make(map[string]int)
	for i, h := range records[0] {
		headers[h] = i
	}

	stopTimes := make(map[string][]StopTime)
	for _, row := range records[1:] {
		seq := 0
		_, _ = fmt.Sscanf(row[headers["stop_sequence"]], "%d", &seq)
		st := StopTime{
			TripID:        row[headers["trip_id"]],
			StopID:        row[headers["stop_id"]],
			DepartureTime: row[headers["departure_time"]],
			StopSequence:  seq,
		}
		stopTimes[st.TripID] = append(stopTimes[st.TripID], st)
	}
	return stopTimes, nil
}

// Funkce pro načtení seznamu tras z routes.txt.
func readRoutes(path string) ([]Route, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)

	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	headers := make(map[string]int)
	for i, h := range records[0] {
		headers[h] = i
	}

	routes := []Route{}
	for _, row := range records[1:] {
		routes = append(routes, Route{
			RouteID:        row[headers["route_id"]],
			RouteShortName: row[headers["route_short_name"]],
			RouteType:      row[headers["route_type"]],
		})
	}
	return routes, nil
}

// Funkce pro načtení mapy stop_id → stop_name, přičemž ignoruje přípony nástupišť.
func readStopNames(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	headers := make(map[string]int)
	for i, h := range records[0] {
		headers[h] = i
	}

	nameMap := make(map[string]string)
	for _, row := range records[1:] {
		stopID := row[headers["stop_id"]]
		baseID := stripPlatformSuffix(stopID)
		if _, exists := nameMap[baseID]; !exists {
			nameMap[baseID] = row[headers["stop_name"]]
		}
	}
	return nameMap, nil
}

// Funkce pro načtení všech zastávek z stops.txt.
func readStops(path string) ([]Stop, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	headers := make(map[string]int)
	for i, h := range records[0] {
		headers[h] = i
	}

	stops := []Stop{}
	for _, row := range records[1:] {
		stops = append(stops, Stop{
			StopID:   row[headers["stop_id"]],
			StopName: row[headers["stop_name"]],
		})
	}
	return stops, nil
}

// Funkce load() načte všechny potřebné statické soubory a vypíše souhrn.
func load(){
	var err error

	trips, err = readTrips("static_data/trips.txt")
	if err != nil {
		log.Fatalf("failed to read trips: %v", err)
	}

	stopTimes, err = readStopTimes("static_data/stop_times.txt")
	if err != nil {
		log.Fatalf("failed to read stop_times: %v", err)
	}

	routes, err = readRoutes("static_data/routes.txt")
	if err != nil {
		log.Fatalf("failed to read routes: %v", err)
	}

	stopNames, err = readStopNames("static_data/stops.txt")
	if err != nil {
		log.Fatalf("failed to read stop_names: %v", err)
	}

	allStops, err = readStops("static_data/stops.txt")
	if err != nil {
		log.Fatalf("failed to read all stops: %v", err)
	}

	totalStopTimes := 0
	for _, sts := range stopTimes {
		totalStopTimes += len(sts)
	}

	fmt.Printf("Na\u010dteno %d trip\u016f, %d stopTime z\u00e1znam\u016f a %d tras\n",
		len(trips), totalStopTimes, len(routes))
	
}