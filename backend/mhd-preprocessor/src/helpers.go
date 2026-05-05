package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"time"
)

func parseRawEnvelope(message []byte) (*rawEnvelope, error) {
	var payload map[string]interface{}
	if err := json.Unmarshal(message, &payload); err != nil {
		return nil, err
	}

	envelope := &rawEnvelope{}
	if attributes, ok := payload["attributes"].(map[string]interface{}); ok {
		envelope.Attributes = attributes
	}
	if geometry, ok := payload["geometry"].(map[string]interface{}); ok {
		envelope.Geometry = geometry
	}
	if filter, ok := payload["filter"].(map[string]interface{}); ok {
		envelope.Filter = filter
	}
	if rawErr, ok := payload["error"]; ok {
		envelope.Error = rawErr
	}

	if len(envelope.Attributes) == 0 && len(envelope.Geometry) == 0 {
		envelope.Attributes = payload
	}

	return envelope, nil
}

func lookupAttribute(attributes map[string]interface{}, keys ...string) interface{} {
	if len(attributes) == 0 {
		return nil
	}

	for _, key := range keys {
		if value, ok := attributes[key]; ok {
			return value
		}
	}

	for _, key := range keys {
		for candidate, value := range attributes {
			if strings.EqualFold(candidate, key) {
				return value
			}
		}
	}

	return nil
}

func extractString(value interface{}) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case json.Number:
		return typed.String()
	case float64:
		return strconv.FormatInt(int64(typed), 10)
	case float32:
		return strconv.FormatInt(int64(typed), 10)
	case int:
		return strconv.Itoa(typed)
	case int32:
		return strconv.FormatInt(int64(typed), 10)
	case int64:
		return strconv.FormatInt(typed, 10)
	case bool:
		return strconv.FormatBool(typed)
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", value))
	}
}

func extractFloat(value interface{}) (float64, bool) {
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

func extractBool(value interface{}) (bool, bool) {
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

func extractTimestamp(value interface{}) (time.Time, bool) {
	if timestamp, ok := extractUnixMillis(value); ok {
		return timestamp, true
	}

	switch typed := value.(type) {
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return time.Time{}, false
		}
		if parsed, err := time.Parse(time.RFC3339Nano, trimmed); err == nil {
			return parsed.UTC(), true
		}
		if parsed, err := time.Parse(time.RFC3339, trimmed); err == nil {
			return parsed.UTC(), true
		}
	}

	return time.Time{}, false
}

func extractUnixMillis(value interface{}) (time.Time, bool) {
	switch typed := value.(type) {
	case float64:
		return time.UnixMilli(int64(typed)).UTC(), true
	case int64:
		return time.UnixMilli(typed).UTC(), true
	case int:
		return time.UnixMilli(int64(typed)).UTC(), true
	case json.Number:
		parsed, err := typed.Int64()
		if err != nil {
			return time.Time{}, false
		}
		return time.UnixMilli(parsed).UTC(), true
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
		if err != nil {
			return time.Time{}, false
		}
		return time.UnixMilli(parsed).UTC(), true
	default:
		return time.Time{}, false
	}
}

func isoWeekKey(value time.Time) string {
	year, week := value.ISOWeek()
	return fmt.Sprintf("%04dW%02d", year, week)
}

func isoDateKey(value time.Time) string {
	return value.Format("2006-01-02")
}

func weekBounds(now time.Time, location *time.Location) (time.Time, time.Time) {
	now = now.In(location)
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location).AddDate(0, 0, -(weekday - 1))
	return start, start.AddDate(0, 0, 7)
}

func stableTripHash(routeID string, departureTime string, stopIDs []string, directionID string) string {
	builder := strings.Builder{}
	builder.WriteString(routeID)
	builder.WriteString("|")
	builder.WriteString(departureTime)
	builder.WriteString("|")
	builder.WriteString(strings.Join(stopIDs, "_"))
	builder.WriteString("|")
	builder.WriteString(directionID)

	sum := sha256.Sum256([]byte(builder.String()))
	return hex.EncodeToString(sum[:])
}

func jitterTime(base time.Time, maxJitter time.Duration) time.Time {
	if maxJitter <= 0 {
		return base
	}
	delta := rand.Int63n((2 * maxJitter.Nanoseconds()) + 1)
	return base.Add(time.Duration(delta-maxJitter.Nanoseconds()) * time.Nanosecond)
}

func friendlyValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "Unknown"
	}
	return value
}

func cloneTags(input map[string]string) map[string]string {
	result := make(map[string]string, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
}

func weekdayOrder(day time.Weekday) int {
	switch day {
	case time.Monday:
		return 0
	case time.Tuesday:
		return 1
	case time.Wednesday:
		return 2
	case time.Thursday:
		return 3
	case time.Friday:
		return 4
	case time.Saturday:
		return 5
	case time.Sunday:
		return 6
	default:
		return 7
	}
}

func serviceDayCodeOrder(code string) int {
	switch strings.TrimSpace(code) {
	case "Mo":
		return 0
	case "Tu":
		return 1
	case "We":
		return 2
	case "Th":
		return 3
	case "Fr":
		return 4
	case "Sa":
		return 5
	case "Su":
		return 6
	default:
		return 7
	}
}

func sortServiceDayCodes(codes []string) {
	sort.SliceStable(codes, func(i, j int) bool {
		return serviceDayCodeOrder(codes[i]) < serviceDayCodeOrder(codes[j])
	})
}

func weekdaysToCodes(days []time.Weekday) []string {
	result := make([]string, 0, len(days))
	for _, day := range days {
		switch day {
		case time.Monday:
			result = append(result, "Mo")
		case time.Tuesday:
			result = append(result, "Tu")
		case time.Wednesday:
			result = append(result, "We")
		case time.Thursday:
			result = append(result, "Th")
		case time.Friday:
			result = append(result, "Fr")
		case time.Saturday:
			result = append(result, "Sa")
		case time.Sunday:
			result = append(result, "Su")
		}
	}
	sortServiceDayCodes(result)
	return result
}

func parseGTFSClock(baseDate time.Time, clock string, location *time.Location) (time.Time, bool) {
	parts := strings.Split(strings.TrimSpace(clock), ":")
	if len(parts) != 3 {
		return time.Time{}, false
	}
	hour, errHour := strconv.Atoi(parts[0])
	minute, errMinute := strconv.Atoi(parts[1])
	second, errSecond := strconv.Atoi(parts[2])
	if errHour != nil || errMinute != nil || errSecond != nil {
		return time.Time{}, false
	}

	dayOffset := hour / 24
	hour = hour % 24
	return time.Date(baseDate.Year(), baseDate.Month(), baseDate.Day(), hour, minute, second, 0, location).AddDate(0, 0, dayOffset).UTC(), true
}

func mapGTFSDate(value string, location *time.Location) (time.Time, bool) {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) != 8 {
		return time.Time{}, false
	}
	parsed, err := time.ParseInLocation("20060102", trimmed, location)
	if err != nil {
		return time.Time{}, false
	}
	return parsed, true
}

func mapDayName(day time.Weekday) string {
	switch day {
	case time.Monday:
		return "Monday"
	case time.Tuesday:
		return "Tuesday"
	case time.Wednesday:
		return "Wednesday"
	case time.Thursday:
		return "Thursday"
	case time.Friday:
		return "Friday"
	case time.Saturday:
		return "Saturday"
	case time.Sunday:
		return "Sunday"
	default:
		return ""
	}
}
