package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

func parseNDICXML(xmlBytes []byte) (*parsedFetch, error) {
	publicationTime := extractPublicationTime(xmlBytes)
	if publicationTime.IsZero() {
		publicationTime = time.Now().UTC()
	}

	decoder := xml.NewDecoder(bytes.NewReader(xmlBytes))
	result := &parsedFetch{
		PublicationTime: publicationTime,
		Snapshots:       make(map[string]*ndicSnapshot),
	}

	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("parse NDIC XML: %w", err)
		}

		start, ok := token.(xml.StartElement)
		if !ok || start.Name.Local != "elaboratedData" {
			continue
		}

		payload, err := decodeElaboratedPayload(decoder, start)
		if err != nil {
			return nil, err
		}

		sourceID, dataType, location, entry, err := parseElaboratedData([]byte(payload))
		if err != nil {
			return nil, err
		}
		if sourceID == "" || entry == nil {
			continue
		}

		snapshot, exists := result.Snapshots[sourceID]
		if !exists {
			snapshot = &ndicSnapshot{SourceIdentification: sourceID}
			result.Snapshots[sourceID] = snapshot
		}

		if snapshot.PrimaryLocationCode == "" {
			snapshot.PrimaryLocationCode = location.PrimaryLocationCode
		}
		if snapshot.SecondaryLocationCode == "" {
			snapshot.SecondaryLocationCode = location.SecondaryLocationCode
		}
		if snapshot.AlertCDirection == "" {
			snapshot.AlertCDirection = location.AlertCDirection
		}

		snapshot.RawEntries = append(snapshot.RawEntries, *entry)

		switch dataType {
		case "TrafficStatus":
			if entry.TrafficLevelValue != "" {
				level := trafficLevelValue(entry.TrafficLevelValue)
				snapshot.TrafficLevelAnyVehicle = &level
			}
		case "TrafficSpeed":
			if entry.Speed != nil {
				speed := *entry.Speed
				snapshot.TrafficSpeedAnyVehicle = &speed
			}
		case "TravelTimeData":
			if entry.Duration != nil {
				duration := *entry.Duration
				snapshot.TravelTimeAnyVehicle = &duration
			}
		}
	}

	return result, nil
}

func extractPublicationTime(xmlBytes []byte) time.Time {
	decoder := xml.NewDecoder(bytes.NewReader(xmlBytes))
	for {
		token, err := decoder.Token()
		if err != nil {
			return time.Time{}
		}
		start, ok := token.(xml.StartElement)
		if !ok || start.Name.Local != "publicationTime" {
			continue
		}
		var text string
		if err := decoder.DecodeElement(&text, &start); err != nil {
			return time.Time{}
		}
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(text))
		if err != nil {
			return time.Time{}
		}
		return parsed.UTC()
	}
}

func decodeElaboratedPayload(decoder *xml.Decoder, start xml.StartElement) (string, error) {
	var payload struct {
		InnerXML string `xml:",innerxml"`
	}
	if err := decoder.DecodeElement(&payload, &start); err != nil {
		return "", fmt.Errorf("decode elaboratedData payload: %w", err)
	}
	return payload.InnerXML, nil
}

type elaboratedLocation struct {
	PrimaryLocationCode   string
	SecondaryLocationCode string
	AlertCDirection       string
}

func parseElaboratedData(payload []byte) (string, string, elaboratedLocation, *ndicRawEntry, error) {
	decoder := xml.NewDecoder(bytes.NewReader(payload))
	location := elaboratedLocation{}

	var (
		sourceID                   string
		dataType                   string
		entry                      *ndicRawEntry
		inPrimaryLocationContext   bool
		inSecondaryLocationContext bool
	)

	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				return sourceID, dataType, location, entry, nil
			}
			return "", "", elaboratedLocation{}, nil, fmt.Errorf("parse elaboratedData payload: %w", err)
		}

		switch typed := token.(type) {
		case xml.StartElement:
			switch typed.Name.Local {
			case "source":
				var src struct {
					ID string `xml:"sourceIdentification"`
				}
				if err := decoder.DecodeElement(&src, &typed); err == nil {
					sourceID = strings.TrimSpace(src.ID)
				}
			case "alertCDirectionCoded":
				var direction string
				if err := decoder.DecodeElement(&direction, &typed); err == nil {
					location.AlertCDirection = strings.TrimSpace(direction)
				}
			case "specificLocation":
				var code string
				if err := decoder.DecodeElement(&code, &typed); err == nil {
					switch {
					case inPrimaryLocationContext && location.PrimaryLocationCode == "":
						location.PrimaryLocationCode = strings.TrimSpace(code)
					case inSecondaryLocationContext && location.SecondaryLocationCode == "":
						location.SecondaryLocationCode = strings.TrimSpace(code)
					}
				}
			default:
				if isPrimaryLocationElement(typed.Name.Local) {
					inPrimaryLocationContext = true
					continue
				}
				if isSecondaryLocationElement(typed.Name.Local) {
					inSecondaryLocationContext = true
					continue
				}

				if currentType := basicDataType(typed); currentType != "" {
					parsedEntry, keep, err := decodeBasicData(decoder, typed, currentType)
					if err != nil {
						return "", "", elaboratedLocation{}, nil, err
					}
					if keep {
						dataType = currentType
						entry = parsedEntry
					}
				}
			}
		case xml.EndElement:
			if isPrimaryLocationElement(typed.Name.Local) {
				inPrimaryLocationContext = false
			}
			if isSecondaryLocationElement(typed.Name.Local) {
				inSecondaryLocationContext = false
			}
		}
	}
}

func isPrimaryLocationElement(local string) bool {
	switch local {
	case "alertCMethod2PrimaryPointLocation", "alertCMethod4PrimaryPointLocation", "alertCMethod4PrimaryPointIntermediateLocation":
		return true
	default:
		return false
	}
}

func isSecondaryLocationElement(local string) bool {
	switch local {
	case "alertCMethod2SecondaryPointLocation", "alertCMethod4SecondaryPointLocation":
		return true
	default:
		return false
	}
}

func basicDataType(start xml.StartElement) string {
	for _, attr := range start.Attr {
		if attr.Name.Local == "type" {
			return strings.TrimSpace(attr.Value)
		}
	}
	return ""
}

func decodeBasicData(decoder *xml.Decoder, start xml.StartElement, dataType string) (*ndicRawEntry, bool, error) {
	switch dataType {
	case "TrafficStatus":
		var payload struct {
			Extension struct {
				NDIC struct {
					VehicleType  string `xml:"vehicleType"`
					TrafficLevel struct {
						Value string `xml:"trafficLevelValue"`
					} `xml:"trafficLevel"`
				} `xml:"ndicFcdExtension"`
			} `xml:"trafficStatusExtension"`
		}
		if err := decoder.DecodeElement(&payload, &start); err != nil {
			return nil, false, err
		}
		entry := &ndicRawEntry{
			Type:              dataType,
			VehicleType:       payload.Extension.NDIC.VehicleType,
			TrafficLevelValue: payload.Extension.NDIC.TrafficLevel.Value,
		}
		return entry, payload.Extension.NDIC.VehicleType == "anyVehicle", nil
	case "TrafficSpeed":
		var payload struct {
			ForVehicles struct {
				VehicleType string `xml:"vehicleType"`
			} `xml:"forVehiclesWithCharacteristicsOf"`
			AverageVehicleSpeed struct {
				Speed float64 `xml:"speed"`
			} `xml:"averageVehicleSpeed"`
		}
		if err := decoder.DecodeElement(&payload, &start); err != nil {
			return nil, false, err
		}
		speed := payload.AverageVehicleSpeed.Speed
		entry := &ndicRawEntry{
			Type:          dataType,
			VehicleType:   payload.ForVehicles.VehicleType,
			QualifierType: payload.ForVehicles.VehicleType,
			Speed:         &speed,
		}
		return entry, payload.ForVehicles.VehicleType == "anyVehicle", nil
	case "TravelTimeData":
		var payload struct {
			VehicleType string `xml:"vehicleType"`
			TravelTime  struct {
				Duration float64 `xml:"duration"`
			} `xml:"travelTime"`
		}
		if err := decoder.DecodeElement(&payload, &start); err != nil {
			return nil, false, err
		}
		duration := payload.TravelTime.Duration
		entry := &ndicRawEntry{
			Type:        dataType,
			VehicleType: payload.VehicleType,
			Duration:    &duration,
		}
		return entry, payload.VehicleType == "anyVehicle", nil
	default:
		if err := decoder.Skip(); err != nil {
			return nil, false, err
		}
		return nil, false, nil
	}
}

func trafficLevelValue(value string) int {
	trimmed := strings.TrimSpace(value)
	if strings.HasPrefix(trimmed, "level") {
		if parsed, err := strconv.Atoi(strings.TrimPrefix(trimmed, "level")); err == nil {
			return parsed
		}
	}
	return 0
}
