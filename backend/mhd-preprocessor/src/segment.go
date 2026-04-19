package main

import (
	"math"
	"regexp"
	"strings"
)

var stopIdentityPattern = regexp.MustCompile(`\d+`)

type segmentMatch struct {
	Index    int
	From     stopMetadata
	To       stopMetadata
	Progress float64
}

func buildSegmentMatch(definition *tripDefinition, record *liveRecord) (*segmentMatch, bool) {
	if definition == nil || len(definition.StopMetadata) < 2 {
		return nil, false
	}

	if index, ok := segmentIndexFromLastStop(definition.StopMetadata, record.LastStopID); ok {
		return newSegmentMatch(definition.StopMetadata, index, record)
	}

	if index, ok := nearestStopSegmentIndex(definition.StopMetadata, record.GeometryLat, record.GeometryLng); ok {
		return newSegmentMatch(definition.StopMetadata, index, record)
	}

	return nil, false
}

func segmentIndexFromLastStop(stops []stopMetadata, lastStopID string) (int, bool) {
	normalizedLastStopID := normalizeStopIdentity(lastStopID)
	if normalizedLastStopID == "" || len(stops) < 2 {
		return 0, false
	}

	bestIndex := -1
	for idx := 0; idx < len(stops)-1; idx++ {
		if normalizeStopIdentity(stops[idx].ID) == normalizedLastStopID {
			bestIndex = idx
		}
	}

	if bestIndex < 0 {
		return 0, false
	}

	return bestIndex, true
}

func normalizeStopIdentity(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	matches := stopIdentityPattern.FindAllString(value, -1)
	if len(matches) > 0 {
		return matches[0]
	}

	return value
}

func newSegmentMatch(stops []stopMetadata, index int, record *liveRecord) (*segmentMatch, bool) {
	if index < 0 || index >= len(stops)-1 {
		return nil, false
	}

	from := stops[index]
	to := stops[index+1]
	return &segmentMatch{
		Index:    index,
		From:     from,
		To:       to,
		Progress: segmentProgress(from, to, record.GeometryLat, record.GeometryLng),
	}, true
}

func nearestStopSegmentIndex(stops []stopMetadata, lat float64, lng float64) (int, bool) {
	if len(stops) < 2 || !validCoords(lat, lng) {
		return 0, false
	}

	bestIndex := -1
	bestDistance := math.MaxFloat64
	for idx := 0; idx < len(stops)-1; idx++ {
		from := stops[idx]
		to := stops[idx+1]
		if !validCoords(from.Lat, from.Lng) || !validCoords(to.Lat, to.Lng) {
			continue
		}

		distance := pointToSegmentDistanceSquared(from.Lat, from.Lng, to.Lat, to.Lng, lat, lng)
		if distance < bestDistance {
			bestDistance = distance
			bestIndex = idx
		}
	}

	if bestIndex < 0 {
		return 0, false
	}

	return bestIndex, true
}

func segmentProgress(from stopMetadata, to stopMetadata, lat float64, lng float64) float64 {
	if !validCoords(from.Lat, from.Lng) || !validCoords(to.Lat, to.Lng) || !validCoords(lat, lng) {
		return 0
	}

	vx := to.Lat - from.Lat
	vy := to.Lng - from.Lng
	lengthSquared := (vx * vx) + (vy * vy)
	if lengthSquared == 0 {
		return 0
	}

	progress := ((lat-from.Lat)*vx + (lng-from.Lng)*vy) / lengthSquared
	if progress < 0 {
		return 0
	}
	if progress > 1 {
		return 1
	}
	return progress
}

func pointToSegmentDistanceSquared(ax float64, ay float64, bx float64, by float64, px float64, py float64) float64 {
	vx := bx - ax
	vy := by - ay
	lengthSquared := (vx * vx) + (vy * vy)
	if lengthSquared == 0 {
		dx := px - ax
		dy := py - ay
		return (dx * dx) + (dy * dy)
	}

	t := ((px-ax)*vx + (py-ay)*vy) / lengthSquared
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}

	projX := ax + (t * vx)
	projY := ay + (t * vy)
	dx := px - projX
	dy := py - projY
	return (dx * dx) + (dy * dy)
}

func validCoords(lat float64, lng float64) bool {
	return !(lat == 0 && lng == 0)
}
