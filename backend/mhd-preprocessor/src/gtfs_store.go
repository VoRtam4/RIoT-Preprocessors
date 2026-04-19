package main

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type GTFSStore struct {
	config appConfig

	mu                 sync.RWMutex
	loadedWeekKey      string
	definitionsByUID   map[string]*tripDefinition
	lineRouteIndex     map[string][]*tripDefinition
	registeredWeekUIDs map[string]struct{}
}

func newGTFSStore(config appConfig) *GTFSStore {
	return &GTFSStore{
		config:             config,
		definitionsByUID:   make(map[string]*tripDefinition),
		lineRouteIndex:     make(map[string][]*tripDefinition),
		registeredWeekUIDs: make(map[string]struct{}),
	}
}

func (s *GTFSStore) refresh(now time.Time) error {
	resp, err := http.Get(s.config.GTFSURL)
	if err != nil {
		return fmt.Errorf("download GTFS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download GTFS: status %s", resp.Status)
	}

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read GTFS body: %w", err)
	}

	files, err := unzipFiles(payload)
	if err != nil {
		return err
	}

	weekStart, weekEnd := weekBounds(now, s.config.GTFSLocation)
	weekKey := isoWeekKey(weekStart)
	definitions, index, err := s.buildDefinitions(files, weekStart, weekEnd, weekKey)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.loadedWeekKey = weekKey
	s.definitionsByUID = definitions
	s.lineRouteIndex = index
	s.mu.Unlock()

	log.Printf("[MHD] GTFS refreshed for week %s | definitions=%d", weekKey, len(definitions))
	return nil
}

func unzipFiles(payload []byte) (map[string][]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(payload), int64(len(payload)))
	if err != nil {
		return nil, fmt.Errorf("open GTFS zip: %w", err)
	}

	result := make(map[string][]byte, len(reader.File))
	for _, file := range reader.File {
		rc, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("open GTFS entry %s: %w", file.Name, err)
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("read GTFS entry %s: %w", file.Name, err)
		}
		result[file.Name] = data
	}
	return result, nil
}

func (s *GTFSStore) buildDefinitions(files map[string][]byte, weekStart time.Time, weekEnd time.Time, weekKey string) (map[string]*tripDefinition, map[string][]*tripDefinition, error) {
	routes, err := parseRoutesCSV(files["routes.txt"])
	if err != nil {
		return nil, nil, err
	}
	stops, err := parseStopsCSV(files["stops.txt"])
	if err != nil {
		return nil, nil, err
	}
	stopTimes, err := parseStopTimesCSV(files["stop_times.txt"])
	if err != nil {
		return nil, nil, err
	}
	trips, err := parseTripsCSV(files["trips.txt"])
	if err != nil {
		return nil, nil, err
	}
	calendar, err := parseCalendarCSV(files["calendar.txt"], s.config.GTFSLocation)
	if err != nil {
		return nil, nil, err
	}
	calendarDates, err := parseCalendarDatesCSV(files["calendar_dates.txt"], s.config.GTFSLocation)
	if err != nil {
		return nil, nil, err
	}
	apiMap := parseAPIMap(files["api.txt"])

	definitions := make(map[string]*tripDefinition)
	index := make(map[string][]*tripDefinition)

	for tripID, trip := range trips {
		tripStops := stopTimes[tripID]
		if len(tripStops) < 2 {
			continue
		}

		serviceDates := activeServiceDates(trip.ServiceID, calendar, calendarDates, weekStart, weekEnd)
		if len(serviceDates) == 0 {
			continue
		}

		stopIDs := make([]string, 0, len(tripStops))
		stopDetails := make([]stopMetadata, 0, len(tripStops))
		for _, item := range tripStops {
			stopIDs = append(stopIDs, item.StopID)
			if stop, ok := stops[item.StopID]; ok {
				stopDetails = append(stopDetails, stop)
			} else {
				stopDetails = append(stopDetails, stopMetadata{ID: item.StopID, Name: item.StopID})
			}
		}

		departureTime := tripStops[0].DepartureTime
		routeShortName := ""
		if route, ok := routes[trip.RouteID]; ok {
			routeShortName = route.RouteShortName
		}
		fromStop := stops[tripStops[0].StopID]
		toStop := stops[tripStops[len(tripStops)-1].StopID]
		fromStopName := fromStop.Name
		toStopName := toStop.Name

		lineID := ""
		liveRouteID := ""
		if pair, ok := apiMap[tripID]; ok {
			lineID = pair[0]
			liveRouteID = pair[1]
		}

		serviceDays := weekdaysToCodes(distinctWeekdays(serviceDates))
		uid := buildTripInstanceUID(trip, departureTime, stopIDs)
		definition, exists := definitions[uid]
		if !exists {
			definition = &tripDefinition{
				UID:            uid,
				Label:          buildTripLabel(routeShortName, departureTime, fromStopName, toStopName),
				TripID:         trip.TripID,
				RouteID:        trip.RouteID,
				RouteShortName: routeShortName,
				DirectionID:    trip.DirectionID,
				TripHeadsign:   trip.TripHeadsign,
				DepartureTime:  departureTime,
				FromStopID:     tripStops[0].StopID,
				FromStopName:   fromStopName,
				ToStopID:       tripStops[len(tripStops)-1].StopID,
				ToStopName:     toStopName,
				ServiceID:      trip.ServiceID,
				StopIDs:        stopIDs,
				StopMetadata:   stopDetails,
				LineID:         lineID,
				LiveRouteID:    liveRouteID,
				ServiceDays:    serviceDays,
				Occurrences:    make([]tripOccurrence, 0, len(serviceDates)),
			}
			definitions[uid] = definition
			if lineID != "" && liveRouteID != "" {
				key := lineRouteIndexKey(lineID, liveRouteID)
				index[key] = append(index[key], definition)
			}
		} else {
			definition.ServiceDays = mergeServiceDays(definition.ServiceDays, serviceDays)
		}

		for _, serviceDate := range serviceDates {
			start, okStart := parseGTFSClock(serviceDate, tripStops[0].DepartureTime, s.config.GTFSLocation)
			end, okEnd := parseGTFSClock(serviceDate, tripStops[len(tripStops)-1].ArrivalTime, s.config.GTFSLocation)
			if !okStart || !okEnd {
				continue
			}

			definition.Occurrences = append(definition.Occurrences, tripOccurrence{
				TripID:         trip.TripID,
				ServiceID:      trip.ServiceID,
				ServiceDate:    serviceDate,
				ScheduledStart: start,
				ScheduledEnd:   end,
			})
		}
	}

	for _, defs := range index {
		sort.Slice(defs, func(i, j int) bool {
			return defs[i].DepartureTime < defs[j].DepartureTime
		})
	}
	for _, definition := range definitions {
		sort.Slice(definition.Occurrences, func(i, j int) bool {
			return definition.Occurrences[i].ScheduledStart.Before(definition.Occurrences[j].ScheduledStart)
		})
	}
	return definitions, index, nil
}

func (s *GTFSStore) definitionsFor(lineID string, liveRouteID string) []*tripDefinition {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lineRouteIndex[lineRouteIndexKey(lineID, liveRouteID)]
}

func (s *GTFSStore) weekUIDs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]string, 0, len(s.definitionsByUID))
	for uid := range s.definitionsByUID {
		result = append(result, uid)
	}
	sort.Strings(result)
	return result
}

func (s *GTFSStore) markWeekUIDRegistered(uid string) {
	s.mu.Lock()
	s.registeredWeekUIDs[uid] = struct{}{}
	s.mu.Unlock()
}

func (s *GTFSStore) isWeekUIDRegistered(uid string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.registeredWeekUIDs[uid]
	return ok
}

func parseCSVRows(data []byte) ([][]string, error) {
	if len(data) == 0 {
		return nil, nil
	}
	reader := csv.NewReader(bytes.NewReader(data))
	reader.FieldsPerRecord = -1
	return reader.ReadAll()
}

func parseRoutesCSV(data []byte) (map[string]tripCSVRecord, error) {
	rows, err := parseCSVRows(data)
	if err != nil || len(rows) == 0 {
		return nil, fmt.Errorf("parse routes.txt: %w", err)
	}

	headers := make(map[string]int)
	for idx, header := range rows[0] {
		headers[strings.TrimPrefix(header, "\ufeff")] = idx
	}

	result := make(map[string]tripCSVRecord)
	for _, row := range rows[1:] {
		routeID := row[headers["route_id"]]
		result[routeID] = tripCSVRecord{
			RouteID:        routeID,
			RouteShortName: row[headers["route_short_name"]],
		}
	}
	return result, nil
}

func parseStopsCSV(data []byte) (map[string]stopMetadata, error) {
	rows, err := parseCSVRows(data)
	if err != nil || len(rows) == 0 {
		return nil, fmt.Errorf("parse stops.txt: %w", err)
	}

	headers := make(map[string]int)
	for idx, header := range rows[0] {
		headers[strings.TrimPrefix(header, "\ufeff")] = idx
	}

	result := make(map[string]stopMetadata)
	for _, row := range rows[1:] {
		lat, _ := strconv.ParseFloat(row[headers["stop_lat"]], 64)
		lng, _ := strconv.ParseFloat(row[headers["stop_lon"]], 64)
		result[row[headers["stop_id"]]] = stopMetadata{
			ID:   row[headers["stop_id"]],
			Name: row[headers["stop_name"]],
			Lat:  lat,
			Lng:  lng,
		}
	}
	return result, nil
}

func parseStopTimesCSV(data []byte) (map[string][]stopTimeRecord, error) {
	rows, err := parseCSVRows(data)
	if err != nil || len(rows) == 0 {
		return nil, fmt.Errorf("parse stop_times.txt: %w", err)
	}

	headers := make(map[string]int)
	for idx, header := range rows[0] {
		headers[strings.TrimPrefix(header, "\ufeff")] = idx
	}

	result := make(map[string][]stopTimeRecord)
	for _, row := range rows[1:] {
		sequence, _ := strconv.Atoi(row[headers["stop_sequence"]])
		record := stopTimeRecord{
			TripID:        row[headers["trip_id"]],
			StopID:        row[headers["stop_id"]],
			ArrivalTime:   row[headers["arrival_time"]],
			DepartureTime: row[headers["departure_time"]],
			StopSequence:  sequence,
		}
		result[record.TripID] = append(result[record.TripID], record)
	}

	for tripID := range result {
		sort.Slice(result[tripID], func(i, j int) bool {
			return result[tripID][i].StopSequence < result[tripID][j].StopSequence
		})
	}
	return result, nil
}

func parseTripsCSV(data []byte) (map[string]tripCSVRecord, error) {
	rows, err := parseCSVRows(data)
	if err != nil || len(rows) == 0 {
		return nil, fmt.Errorf("parse trips.txt: %w", err)
	}

	headers := make(map[string]int)
	for idx, header := range rows[0] {
		headers[strings.TrimPrefix(header, "\ufeff")] = idx
	}

	result := make(map[string]tripCSVRecord)
	for _, row := range rows[1:] {
		record := tripCSVRecord{
			TripID:       row[headers["trip_id"]],
			RouteID:      row[headers["route_id"]],
			ServiceID:    row[headers["service_id"]],
			TripHeadsign: row[headers["trip_headsign"]],
			DirectionID:  row[headers["direction_id"]],
		}
		result[record.TripID] = record
	}
	return result, nil
}

type calendarRow struct {
	ServiceID string
	StartDate time.Time
	EndDate   time.Time
	Days      map[time.Weekday]bool
}

func parseCalendarCSV(data []byte, location *time.Location) (map[string]calendarRow, error) {
	if len(data) == 0 {
		return map[string]calendarRow{}, nil
	}
	rows, err := parseCSVRows(data)
	if err != nil || len(rows) == 0 {
		return nil, fmt.Errorf("parse calendar.txt: %w", err)
	}

	headers := make(map[string]int)
	for idx, header := range rows[0] {
		headers[strings.TrimPrefix(header, "\ufeff")] = idx
	}

	result := make(map[string]calendarRow)
	for _, row := range rows[1:] {
		startDate, okStart := mapGTFSDate(row[headers["start_date"]], location)
		endDate, okEnd := mapGTFSDate(row[headers["end_date"]], location)
		if !okStart || !okEnd {
			continue
		}
		result[row[headers["service_id"]]] = calendarRow{
			ServiceID: row[headers["service_id"]],
			StartDate: startDate,
			EndDate:   endDate,
			Days: map[time.Weekday]bool{
				time.Monday:    row[headers["monday"]] == "1",
				time.Tuesday:   row[headers["tuesday"]] == "1",
				time.Wednesday: row[headers["wednesday"]] == "1",
				time.Thursday:  row[headers["thursday"]] == "1",
				time.Friday:    row[headers["friday"]] == "1",
				time.Saturday:  row[headers["saturday"]] == "1",
				time.Sunday:    row[headers["sunday"]] == "1",
			},
		}
	}
	return result, nil
}

type calendarDateException struct {
	Date          time.Time
	ServiceID     string
	ExceptionType string
}

func parseCalendarDatesCSV(data []byte, location *time.Location) (map[string][]calendarDateException, error) {
	if len(data) == 0 {
		return map[string][]calendarDateException{}, nil
	}
	rows, err := parseCSVRows(data)
	if err != nil || len(rows) == 0 {
		return nil, fmt.Errorf("parse calendar_dates.txt: %w", err)
	}

	headers := make(map[string]int)
	for idx, header := range rows[0] {
		headers[strings.TrimPrefix(header, "\ufeff")] = idx
	}

	result := make(map[string][]calendarDateException)
	for _, row := range rows[1:] {
		date, ok := mapGTFSDate(row[headers["date"]], location)
		if !ok {
			continue
		}
		exception := calendarDateException{
			Date:          date,
			ServiceID:     row[headers["service_id"]],
			ExceptionType: row[headers["exception_type"]],
		}
		result[exception.ServiceID] = append(result[exception.ServiceID], exception)
	}
	return result, nil
}

func parseAPIMap(data []byte) map[string][2]string {
	result := make(map[string][2]string)
	if len(data) == 0 {
		return result
	}

	re := regexp.MustCompile(`(\d+)/(\d+)\s*=\s*(\d+)$`)
	for _, line := range strings.Split(string(data), "\n") {
		match := re.FindStringSubmatch(strings.TrimSpace(line))
		if len(match) == 4 {
			result[match[3]] = [2]string{match[1], match[2]}
		}
	}
	return result
}

func activeServiceDates(serviceID string, calendar map[string]calendarRow, exceptions map[string][]calendarDateException, weekStart time.Time, weekEnd time.Time) []time.Time {
	result := make([]time.Time, 0, 7)
	dayCursor := weekStart
	row, hasCalendar := calendar[serviceID]

	for dayCursor.Before(weekEnd) {
		active := false
		if hasCalendar && !dayCursor.Before(row.StartDate) && !dayCursor.After(row.EndDate) {
			active = row.Days[dayCursor.Weekday()]
		}
		for _, exception := range exceptions[serviceID] {
			if exception.Date.Equal(dayCursor) {
				active = exception.ExceptionType == "1"
			}
		}
		if active {
			result = append(result, dayCursor)
		}
		dayCursor = dayCursor.AddDate(0, 0, 1)
	}

	return result
}

func distinctWeekdays(dates []time.Time) []time.Weekday {
	seen := make(map[time.Weekday]struct{})
	result := make([]time.Weekday, 0, len(dates))
	for _, date := range dates {
		if _, ok := seen[date.Weekday()]; ok {
			continue
		}
		seen[date.Weekday()] = struct{}{}
		result = append(result, date.Weekday())
	}
	sort.Slice(result, func(i, j int) bool {
		return weekdayOrder(result[i]) < weekdayOrder(result[j])
	})
	return result
}

func buildTripLabel(routeShortName string, departureTime string, fromStop string, toStop string) string {
	return fmt.Sprintf("%s %s | %s -> %s",
		friendlyValue(routeShortName),
		friendlyValue(departureTime),
		friendlyValue(fromStop),
		friendlyValue(toStop),
	)
}

func lineRouteIndexKey(lineID string, liveRouteID string) string {
	return lineID + "/" + liveRouteID
}

func buildTripInstanceUID(trip tripCSVRecord, departureTime string, stopIDs []string) string {
	return fmt.Sprintf("%s_%s", mhdSDTypeUID, stableTripHash(trip.RouteID, departureTime, stopIDs, trip.DirectionID))
}

func mergeServiceDays(left []string, right []string) []string {
	seen := make(map[string]struct{}, len(left)+len(right))
	result := make([]string, 0, len(left)+len(right))

	for _, item := range left {
		if _, exists := seen[item]; exists {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	for _, item := range right {
		if _, exists := seen[item]; exists {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}

	sortServiceDayCodes(result)
	return result
}
