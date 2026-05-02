package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

func normalizeDisplayTag(value interface{}) string {
	raw := strings.TrimSpace(fmt.Sprintf("%v", value))
	raw = sanitizeText(raw)
	if raw == "" || raw == "<nil>" {
		return unknownTagValue
	}
	return raw
}

func sanitizeText(raw string) string {
	if !utf8.ValidString(raw) {
		raw = strings.ToValidUTF8(raw, "")
	}

	var builder strings.Builder
	for len(raw) > 0 {
		r, size := utf8.DecodeRuneInString(raw)
		raw = raw[size:]
		if r == utf8.RuneError && size == 1 {
			continue
		}
		if unicode.IsControl(r) && !unicode.IsSpace(r) {
			continue
		}
		builder.WriteRune(r)
	}

	return strings.TrimSpace(builder.String())
}

func buildInstanceUID(segmentID int64, isForward bool) string {
	suffix := "REV"
	if isForward {
		suffix = "FWD"
	}
	return fmt.Sprintf("WAZE_JAM_%d_%s", segmentID, suffix)
}

func buildInstanceLabel(city string, street string, segmentID string, directionLabel string) string {
	return fmt.Sprintf("%s, %s: %s %s",
		friendlyLabel(city),
		friendlyLabel(street),
		friendlyLabel(segmentID),
		friendlyLabel(directionLabel),
	)
}

func friendlyLabel(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || value == unknownTagValue {
		return "Unknown"
	}
	return value
}

func parseLine(value interface{}) []lineCoordinate {
	rawLine, ok := value.([]interface{})
	if !ok {
		return nil
	}

	result := make([]lineCoordinate, 0, len(rawLine))
	for _, point := range rawLine {
		pointMap, ok := point.(map[string]interface{})
		if !ok {
			continue
		}

		x, okX := extractMaybeFloat(pointMap["x"])
		y, okY := extractMaybeFloat(pointMap["y"])
		if !okX || !okY {
			continue
		}

		result = append(result, lineCoordinate{X: x, Y: y})
	}

	return result
}

func parseSegments(value interface{}) []segmentReference {
	rawSegments, ok := value.([]interface{})
	if !ok {
		return nil
	}

	result := make([]segmentReference, 0, len(rawSegments))
	for _, item := range rawSegments {
		segmentMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		id, okID := extractMaybeInt64(segmentMap["ID"])
		fromNode, okFrom := extractMaybeInt64(segmentMap["fromNode"])
		toNode, okTo := extractMaybeInt64(segmentMap["toNode"])
		isForward, okForward := extractMaybeBool(segmentMap["isForward"])
		if !okID || !okFrom || !okTo || !okForward {
			continue
		}

		result = append(result, segmentReference{
			ID:        id,
			FromNode:  fromNode,
			ToNode:    toNode,
			IsForward: isForward,
		})
	}

	sort.SliceStable(result, func(i, j int) bool {
		if result[i].ID != result[j].ID {
			return result[i].ID < result[j].ID
		}
		if result[i].IsForward != result[j].IsForward {
			return !result[i].IsForward && result[j].IsForward
		}
		if result[i].FromNode != result[j].FromNode {
			return result[i].FromNode < result[j].FromNode
		}
		return result[i].ToNode < result[j].ToNode
	})

	return result
}

func reduceRawJam(jam map[string]interface{}) map[string]interface{} {
	reduced := make(map[string]interface{})
	for key, value := range jam {
		switch key {
		case "country", "city", "street", "roadType":
			continue
		default:
			reduced[key] = value
		}
	}
	return reduced
}

func extractEventTime(jam map[string]interface{}) (time.Time, int64) {
	pubMillis, ok := extractMaybeInt64(jam["pubMillis"])
	if !ok || pubMillis <= 0 {
		now := time.Now().UTC()
		return now, now.UnixMilli()
	}

	return time.UnixMilli(pubMillis).UTC(), pubMillis
}

func extractFloat(value interface{}) float64 {
	result, ok := extractMaybeFloat(value)
	if !ok {
		return 0
	}
	return result
}

func extractMaybeFloat(value interface{}) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case json.Number:
		parsed, err := typed.Float64()
		if err != nil {
			return 0, false
		}
		return parsed, true
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func extractMaybeInt64(value interface{}) (int64, bool) {
	switch typed := value.(type) {
	case int64:
		return typed, true
	case int:
		return int64(typed), true
	case float64:
		return int64(typed), true
	case json.Number:
		parsed, err := typed.Int64()
		if err != nil {
			return 0, false
		}
		return parsed, true
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func extractMaybeBool(value interface{}) (bool, bool) {
	switch typed := value.(type) {
	case bool:
		return typed, true
	case string:
		parsed, err := strconv.ParseBool(strings.TrimSpace(typed))
		if err != nil {
			return false, false
		}
		return parsed, true
	default:
		return false, false
	}
}

func safeInt64String(value int64) string {
	if value == 0 {
		return unknownTagValue
	}
	return strconv.FormatInt(value, 10)
}

func sortRawJams(items []map[string]interface{}) {
	sort.SliceStable(items, func(i, j int) bool {
		return rawJamSortKey(items[i]) < rawJamSortKey(items[j])
	})
}

func rawJamSortKey(item map[string]interface{}) string {
	pubMillis, _ := extractMaybeInt64(item["pubMillis"])
	uuid := strings.TrimSpace(fmt.Sprintf("%v", item["uuid"]))
	blocking := strings.TrimSpace(fmt.Sprintf("%v", item["blockingAlertUuid"]))
	return fmt.Sprintf("%020d|%s|%s", pubMillis, uuid, blocking)
}
