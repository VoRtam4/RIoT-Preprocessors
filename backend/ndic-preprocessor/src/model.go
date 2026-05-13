/**
 * @file model.go
 * @brief Datové struktury NDIC preprocesoru pro DATEX snapshoty, TMC metadata a runtime stav.
 *
 * @author Dominik Vondruška
 * @author Vojtěch Hubáček
 * @ingroup riot_ndic_preprocessor
 *
 * @par Autorský podíl
 * - Dominik Vondruška: základní model transformovaných NDIC záznamů.
 * - Vojtěch Hubáček: rozšíření modelu o EventTime, TMC metadata, tagy a runtime stav silničních segmentů.
 */
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
	SourceIdentification   string
	EventTime              time.Time
	PrimaryLocationCode    string
	SecondaryLocationCode  string
	TMCMetadata            *tmcMetadata
	TrafficLevelAnyVehicle *int
	TrafficSpeedAnyVehicle *float64
	TravelTimeAnyVehicle   *float64
	RawEntries             []ndicRawEntry
}

type ndicRawEntry struct {
	Type              string    `json:"type"`
	EventTime         time.Time `json:"-"`
	VehicleType       string    `json:"vehicleType,omitempty"`
	QualifierType     string    `json:"qualifierType,omitempty"`
	TrafficLevelValue string    `json:"trafficLevelValue,omitempty"`
	Speed             *float64  `json:"speed,omitempty"`
	Duration          *float64  `json:"duration,omitempty"`
}

type tmcMetadata struct {
	LocationCode string
	PointName    string
	AreaRef      string
	AreaName     string
	RoadLCD      string
	SegmentLCD   string
	RoadNumber   string
	RoadName     string
	Latitude     *float64
	Longitude    *float64
}

type runtimeInstanceState struct {
	UID             string
	Label           string
	Tags            map[string]string
	SeenSinceStart  bool
	CurrentlyActive bool
	LastEventTime   time.Time
}
