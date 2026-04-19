/**
 * @File: departure_times_handler.go
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
    "strings"
)

type CalendarEntry struct {
    ServiceID string
    Monday    string
    Tuesday   string
    Wednesday string
    Thursday  string
    Friday    string
    Saturday  string
    Sunday    string
}

var calendar []CalendarEntry

// Funkce pro načtení kalendáře z calendar.txt a uložení do seznamu struktur CalendarEntry.
func readCalendar(path string) ([]CalendarEntry, error) {
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

    calendar := []CalendarEntry{}
    for _, row := range records[1:] {
        calendar = append(calendar, CalendarEntry{
            ServiceID: row[headers["service_id"]],
            Monday:    row[headers["monday"]],
            Tuesday:   row[headers["tuesday"]],
            Wednesday: row[headers["wednesday"]],
            Thursday:  row[headers["thursday"]],
            Friday:    row[headers["friday"]],
            Saturday:  row[headers["saturday"]],
            Sunday:    row[headers["sunday"]],
        })
    }
    return calendar, nil
}

// Funkce pro zpracování HTTP požadavku a vrácení časů odjezdů spojů pro zadanou linku, zastávky a den.
func departureTimesHandler(w http.ResponseWriter, r *http.Request) {
	routeID := r.URL.Query().Get("route_id")
	fromStop := r.URL.Query().Get("from_stop")
	toStop := r.URL.Query().Get("to_stop")
	day := strings.ToLower(r.URL.Query().Get("day"))
	direction := strings.ToLower(r.URL.Query().Get("direction"))

	if routeID == "" || fromStop == "" || toStop == "" || day == "" {
		http.Error(w, "Missing required parameters", http.StatusBadRequest)
		return
	}

	// Aktivní služby podle dne.
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

	departureTimes := []string{}
	for tripID, trip := range trips {
		if trip.RouteID != routeID || !activeServices[trip.ServiceID] {
			continue
		}

		sts := stopTimes[tripID]
		var fromTime string
		fromSeq := -1
		toSeq := -1
		for _, st := range sts {
			if strings.HasPrefix(st.StopID, fromStop) {
				fromTime = st.DepartureTime
				fromSeq = st.StopSequence
			}
			if strings.HasPrefix(st.StopID, toStop) {
				toSeq = st.StopSequence
			}
		}

		if fromSeq != -1 && toSeq != -1 {
			if direction == "backward" && fromSeq > toSeq {
				departureTimes = append(departureTimes, fromTime)
			} else if direction != "backward" && fromSeq < toSeq {
				departureTimes = append(departureTimes, fromTime)
			}
		}
	}

	sort.Strings(departureTimes)
	w.Header().Set("Content-Type", "application/json")
	if len(departureTimes) == 0 {
		json.NewEncoder(w).Encode(map[string]string{
			"message": "No trips found for this combination of route, stops and day",
		})
		return
	}
	json.NewEncoder(w).Encode(departureTimes)
}

// Registrace handleru při spuštění.
func init() {
    var err error
    calendar, err = readCalendar("static_data/calendar.txt")
    if err != nil {
        panic(err)
    }

    http.HandleFunc("/gtfs/departure-times", departureTimesHandler)
    fmt.Println("/gtfs/departure-times endpoint registered")
}