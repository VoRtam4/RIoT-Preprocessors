package main

import "time"

type fetchEnvelope struct {
	LatestRaw string `json:"latest_raw"`
}

type parsedFetch struct {
	PublicationTime time.Time
	Snapshots       map[string]*ndicSnapshot
}

type ndicSnapshot struct {
	SourceIdentification string
	TrafficLevelAnyVehicle *int
	TrafficSpeedAnyVehicle *float64
	TravelTimeAnyVehicle  *float64
	RawEntries            []ndicRawEntry
}

type ndicRawEntry struct {
	Type              string   `json:"type"`
	VehicleType       string   `json:"vehicleType,omitempty"`
	QualifierType     string   `json:"qualifierType,omitempty"`
	TrafficLevelValue string   `json:"trafficLevelValue,omitempty"`
	Speed             *float64 `json:"speed,omitempty"`
	Duration          *float64 `json:"duration,omitempty"`
}

type runtimeInstanceState struct {
	UID            string
	Label          string
	Tags           map[string]string
	SeenSinceStart bool
	CurrentlyActive bool
	LastEventTime  time.Time
}
