package main

import (
	"encoding/json"
	"hash/fnv"
	"sort"
	"strconv"
	"time"
)

func buildAggregatesFromJam(jam map[string]interface{}) []*deviceAggregate {
	segments := parseSegments(jam["segments"])
	if len(segments) == 0 {
		return nil
	}

	country := normalizeDisplayTag(jam["country"])
	city := normalizeDisplayTag(jam["city"])
	street := normalizeDisplayTag(jam["street"])
	roadType := normalizeDisplayTag(jam["roadType"])
	eventTime, pubMillis := extractEventTime(jam)
	rawJam := reduceRawJam(jam)
	aggregates := make([]*deviceAggregate, 0, len(segments))

	for _, segment := range segments {
		directionLabel := buildDirectionLabel(segment.IsForward)
		segmentID := strconv.FormatInt(segment.ID, 10)

		tags := map[string]string{
			"country":   country,
			"city":      city,
			"street":    street,
			"roadType":  roadType,
			"segmentId": segmentID,
			"fromNode":  safeInt64String(segment.FromNode),
			"toNode":    safeInt64String(segment.ToNode),
			"isForward": strconv.FormatBool(segment.IsForward),
		}

		aggregates = append(aggregates, &deviceAggregate{
			UID:             buildInstanceUID(segment.ID, segment.IsForward),
			Label:           buildInstanceLabel(city, street, segmentID, directionLabel),
			EventTime:       eventTime,
			Tags:            tags,
			RawJams:         []map[string]interface{}{rawJam},
			JamCount:        1,
			Delay:           normalizeAggregatedDelay(extractFloat(jam["delay"])),
			Length:          extractFloat(jam["length"]),
			Level:           extractFloat(jam["level"]),
			Speed:           extractFloat(jam["speed"]),
			SpeedKPH:        extractFloat(jam["speedKMH"]),
			PubMillisLatest: pubMillis,
		})
	}

	return aggregates
}

func mergeAggregate(target *deviceAggregate, addition *deviceAggregate) {
	target.RawJams = append(target.RawJams, addition.RawJams...)
	target.JamCount += addition.JamCount

	if isGreaterDelay(addition.Delay, target.Delay) {
		target.Delay = addition.Delay
	}
	target.Length += addition.Length
	if addition.Level > target.Level {
		target.Level = addition.Level
	}
	if target.Speed == 0 || (addition.Speed > 0 && addition.Speed < target.Speed) {
		target.Speed = addition.Speed
	}
	if target.SpeedKPH == 0 || (addition.SpeedKPH > 0 && addition.SpeedKPH < target.SpeedKPH) {
		target.SpeedKPH = addition.SpeedKPH
	}
	if addition.PubMillisLatest > target.PubMillisLatest {
		target.PubMillisLatest = addition.PubMillisLatest
		target.EventTime = addition.EventTime
	}

	sortRawJams(target.RawJams)
}

func buildActiveParams(aggregate *deviceAggregate) map[string]interface{} {
	params := tagsToParams(aggregate.Tags)

	params["delay"] = aggregate.Delay
	params["length"] = aggregate.Length
	params["level"] = aggregate.Level
	params["speed"] = aggregate.Speed
	params["speedKPH"] = aggregate.SpeedKPH
	params["jamCount"] = float64(aggregate.JamCount)
	params["pubMillisLatest"] = float64(aggregate.PubMillisLatest)

	sortRawJams(aggregate.RawJams)
	rawJamsJSON, err := json.Marshal(aggregate.RawJams)
	if err == nil {
		params["rawJams"] = string(rawJamsJSON)
	}

	return params
}

func buildZeroDelayParams(tags map[string]string) map[string]interface{} {
	params := tagsToParams(tags)
	params["delay"] = float64(0)
	params["jamCount"] = float64(0)
	params["rawJams"] = "[]"
	return params
}

func buildStartupZeroDelayEventTime(uid string, activeEventTime time.Time) time.Time {
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(uid))

	maxJitterMillis := int64(startupJitter / time.Millisecond)
	if maxJitterMillis <= 0 {
		return activeEventTime.Add(-time.Millisecond)
	}

	offsetMillis := int64(hasher.Sum32()%uint32(maxJitterMillis)) + 1
	return activeEventTime.Add(-time.Duration(offsetMillis) * time.Millisecond)
}

func shouldPublishStartupZeroDelay(eventTime time.Time) bool {
	return !eventTime.Before(preprocessorStartedAt)
}

func tagsToParams(tags map[string]string) map[string]interface{} {
	params := make(map[string]interface{}, len(tags)+1)
	keys := make([]string, 0, len(tags))
	for key := range tags {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		params[key] = tags[key]
	}
	return params
}

func cloneTags(src map[string]string) map[string]string {
	dst := make(map[string]string, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

func buildDirectionLabel(isForward bool) string {
	if isForward {
		return "Forward"
	}
	return "Reverse"
}

func normalizeAggregatedDelay(delay float64) float64 {
	return delay
}

func isGreaterDelay(left float64, right float64) bool {
	if left == -1 {
		return right != -1
	}
	if right == -1 {
		return false
	}
	return left > right
}
