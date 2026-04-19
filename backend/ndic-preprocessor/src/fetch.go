package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/MichalBures-OG/bp-bures-RIoT-commons/src/rabbitmq"
)

func fetchAndProcessNDICData(client rabbitmq.Client, config appConfig) time.Time {
	fetch, rawXML, err := fetchNDIC(config)
	if err != nil {
		log.Printf("[NDIC] Fetch failed: %v", err)
		return time.Time{}
	}

	saveLatestRaw(string(rawXML))
	persistRawXML(config.RawStorageDir, rawXML, fetch.PublicationTime)
	processFetchResult(client, config, fetch)
	return fetch.PublicationTime
}

func fetchNDIC(config appConfig) (*parsedFetch, []byte, error) {
	resp, err := http.Get(config.NDICURL)
	if err != nil {
		return nil, nil, fmt.Errorf("download NDIC: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("download NDIC: status %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("read NDIC body: %w", err)
	}

	xmlBytes, err := unwrapXML(body, resp.Header.Get("Content-Type"))
	if err != nil {
		return nil, nil, err
	}

	parsed, err := parseNDICXML(xmlBytes)
	if err != nil {
		return nil, nil, err
	}
	return parsed, xmlBytes, nil
}

func unwrapXML(body []byte, contentType string) ([]byte, error) {
	if strings.Contains(contentType, "application/json") {
		var wrapper fetchEnvelope
		if err := json.Unmarshal(body, &wrapper); err != nil {
			return nil, fmt.Errorf("parse NDIC JSON wrapper: %w", err)
		}
		return []byte(wrapper.LatestRaw), nil
	}
	return body, nil
}

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

		sourceID, dataType, entry, err := decodeElaboratedData(decoder)
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

func decodeElaboratedData(decoder *xml.Decoder) (string, string, *ndicRawEntry, error) {
	var sourceID string
	for {
		token, err := decoder.Token()
		if err != nil {
			return "", "", nil, err
		}

		switch typed := token.(type) {
		case xml.StartElement:
			if typed.Name.Local == "source" {
				var src struct {
					ID string `xml:"sourceIdentification"`
				}
				if err := decoder.DecodeElement(&src, &typed); err == nil {
					sourceID = strings.TrimSpace(src.ID)
				}
				continue
			}

			dataType := basicDataType(typed)
			if dataType == "" {
				if err := decoder.Skip(); err != nil {
					return "", "", nil, err
				}
				continue
			}

			entry, keep, err := decodeBasicData(decoder, typed, dataType)
			if err != nil {
				return "", "", nil, err
			}
			if !keep {
				return sourceID, dataType, nil, nil
			}
			return sourceID, dataType, entry, nil
		case xml.EndElement:
			if typed.Name.Local == "elaboratedData" {
				return sourceID, "", nil, nil
			}
		}
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
