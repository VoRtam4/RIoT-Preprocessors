package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

type tmcEnricher struct {
	tmcDir      string
	mutex       sync.RWMutex
	pointsByLCD map[string]map[string]string
}

func newTMCEnricher(config appConfig) *tmcEnricher {
	if strings.TrimSpace(config.TMCDir) == "" {
		return nil
	}

	enricher := &tmcEnricher{
		tmcDir:      config.TMCDir,
		pointsByLCD: make(map[string]map[string]string),
	}

	if err := enricher.load(); err != nil {
		log.Printf("[NDIC] TMC enrichment unavailable: %v", err)
	}
	return enricher
}

func (e *tmcEnricher) enrichFetch(fetch *parsedFetch) {
	if fetch == nil {
		return
	}
	if !e.isLoaded() {
		return
	}

	for _, snapshot := range fetch.Snapshots {
		locationCode := firstNonEmpty(snapshot.PrimaryLocationCode, snapshot.SecondaryLocationCode)
		if locationCode == "" {
			continue
		}
		if metadata := e.lookup(locationCode); metadata != nil {
			snapshot.TMCMetadata = metadata
		}
	}
}

func (e *tmcEnricher) isLoaded() bool {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	return len(e.pointsByLCD) > 0
}

func (e *tmcEnricher) load() error {
	points, err := loadTMCPoints(e.tmcDir)
	if err != nil {
		return err
	}

	rebuilt := make(map[string]map[string]string, len(points))
	for _, point := range points {
		lcd := strings.TrimSpace(point["LCD"])
		if lcd == "" {
			continue
		}
		rebuilt[lcd] = point
	}

	e.mutex.Lock()
	e.pointsByLCD = rebuilt
	e.mutex.Unlock()
	log.Printf("[NDIC] Loaded %d TMC points from %s", len(rebuilt), e.tmcDir)
	return nil
}

func loadTMCPoints(baseDir string) ([]map[string]string, error) {
	pointsPath := filepath.Join(baseDir, "ltcze10_1_points.txt")
	file, err := os.Open(pointsPath)
	if err != nil {
		return nil, fmt.Errorf("open TMC points file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = ';'
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1

	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("read TMC headers: %w", err)
	}

	points := make([]map[string]string, 0)
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read TMC record: %w", err)
		}

		point := make(map[string]string, len(headers))
		for i, header := range headers {
			if i < len(record) {
				point[header] = strings.TrimSpace(record[i])
			}
		}
		points = append(points, point)
	}
	return points, nil
}

func (e *tmcEnricher) lookup(locationCode string) *tmcMetadata {
	e.mutex.RLock()
	point, exists := e.pointsByLCD[locationCode]
	e.mutex.RUnlock()
	if !exists {
		return nil
	}

	latitude := parseLocalizedFloat(point["WGS84_Y"])
	longitude := parseLocalizedFloat(point["WGS84_X"])

	return &tmcMetadata{
		LocationCode: locationCode,
		PointName:    composePointName(point),
		AreaRef:      strings.TrimSpace(point["AREA_REF"]),
		AreaName:     strings.TrimSpace(point["AREA_NAME"]),
		RoadLCD:      strings.TrimSpace(point["ROA_LCD"]),
		SegmentLCD:   strings.TrimSpace(point["SEG_LCD"]),
		RoadNumber:   strings.TrimSpace(point["ROADNUMBER"]),
		RoadName:     strings.TrimSpace(point["ROADNAME"]),
		Latitude:     latitude,
		Longitude:    longitude,
	}
}

func composePointName(point map[string]string) string {
	first := strings.TrimSpace(point["FIRSTNAME"])
	second := strings.TrimSpace(point["SECONDNAME"])
	switch {
	case first == "" && second == "":
		return ""
	case second == "":
		return first
	case first == "":
		return second
	default:
		return first + " - " + second
	}
}

func parseLocalizedFloat(value string) *float64 {
	trimmed := strings.TrimSpace(strings.ReplaceAll(value, ",", "."))
	if trimmed == "" {
		return nil
	}
	parsed, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return nil
	}
	return &parsed
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
