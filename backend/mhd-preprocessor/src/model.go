package main

import "time"

type rawEnvelope struct {
	Attributes map[string]interface{} `json:"attributes"`
	Filter     map[string]interface{} `json:"filter"`
	Error      interface{}            `json:"error"`
	Geometry   map[string]interface{} `json:"geometry"`
}

type liveRecord struct {
	RawMessage             string
	Attributes             map[string]interface{}
	SourceTimestamp        time.Time
	GeometryLat            float64
	GeometryLng            float64
	ObjectID               string
	GlobalID               string
	VehicleRuntimeID       string
	VehicleType            string
	LineType               string
	LineID                 string
	LineName               string
	LiveRouteID            string
	Course                 string
	LowFloor               string
	Delay                  float64
	Bearing                float64
	LastStopID             string
	LastPostID             string
	FinalStopID            string
	FinalStopName          string
	ServiceID              string
	ObservedDepartureTime  time.Time
	ObservedDepartureValid bool
	OCFinalStopID          string
	OCFinalStopName        string
	OCLineID               string
	OCLineName             string
	OCRouteID              string
	State                  string
	TMFlagText             string
	IsInactive             bool
}

type stopMetadata struct {
	ID   string
	Name string
	Lat  float64
	Lng  float64
}

type tripOccurrence struct {
	TripID         string
	ServiceID      string
	ServiceDate    time.Time
	ScheduledStart time.Time
	ScheduledEnd   time.Time
}

type tripDefinition struct {
	UID            string
	Label          string
	TripID         string
	RouteID        string
	RouteShortName string
	DirectionID    string
	TripHeadsign   string
	DepartureTime  string
	FromStopID     string
	FromStopName   string
	ToStopID       string
	ToStopName     string
	ServiceID      string
	ServiceDays    []string
	StopIDs        []string
	StopMetadata   []stopMetadata
	LineID         string
	LiveRouteID    string
	Occurrences    []tripOccurrence
}

type tripMatch struct {
	Definition *tripDefinition
	Occurrence tripOccurrence
	Score      time.Duration
}

type runtimeInstanceState struct {
	UID              string
	Label            string
	Tags             map[string]string
	SeenSinceStart   bool
	CurrentlyActive  bool
	CloseAt          time.Time
	LastSourceTime   time.Time
	LastOccurrenceAt time.Time
	LastVehicleID    string
}

type tripCSVRecord struct {
	TripID         string
	RouteID        string
	ServiceID      string
	TripHeadsign   string
	DirectionID    string
	RouteShortName string
	LineID         string
	LiveRouteID    string
}

type stopTimeRecord struct {
	TripID        string
	StopID        string
	ArrivalTime   string
	DepartureTime string
	StopSequence  int
}
