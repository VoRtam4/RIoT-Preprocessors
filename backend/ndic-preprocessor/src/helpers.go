/**
 * @file helpers.go
 * @brief Pomocné funkce pro normalizaci hodnot a skládání NDIC identifikátorů.
 *
 * @author Vojtěch Hubáček
 * @ingroup riot_ndic_preprocessor
 *
 * @par Autorský podíl
 * - Vojtěch Hubáček: návrh a implementace pomocných funkcí pro rozšířený NDIC model.
 */
package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"
)

func jitterTime(base time.Time, maxJitter time.Duration) time.Time {
	if maxJitter <= 0 {
		return base
	}
	delta := rand.Int63n((2 * maxJitter.Nanoseconds()) + 1)
	return base.Add(time.Duration(delta-maxJitter.Nanoseconds()) * time.Nanosecond)
}

func cloneTags(input map[string]string) map[string]string {
	result := make(map[string]string, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
}

func mustJSON(value interface{}) string {
	payload, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(payload)
}

func buildNDICInstanceLabel(snapshot *ndicSnapshot) string {
	if snapshot == nil {
		return "Unknown | Unknown [Unknown]"
	}

	locationCode := firstNonEmpty(
		valueOrEmpty(snapshot.TMCMetadata, func(meta *tmcMetadata) string { return meta.LocationCode }),
		snapshot.PrimaryLocationCode,
		snapshot.SecondaryLocationCode,
	)
	road := firstNonEmpty(
		valueOrEmpty(snapshot.TMCMetadata, func(meta *tmcMetadata) string { return meta.RoadNumber }),
		valueOrEmpty(snapshot.TMCMetadata, func(meta *tmcMetadata) string { return meta.RoadName }),
	)
	point := firstNonEmpty(
		valueOrEmpty(snapshot.TMCMetadata, func(meta *tmcMetadata) string { return meta.PointName }),
		valueOrEmpty(snapshot.TMCMetadata, func(meta *tmcMetadata) string { return meta.AreaName }),
	)

	return fmt.Sprintf(
		"%s | %s [%s]",
		friendlyNDICLabel(road),
		friendlyNDICLabel(point),
		friendlyNDICLabel(locationCode),
	)
}

func valueOrEmpty[T any](value *T, getter func(*T) string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(getter(value))
}

func friendlyNDICLabel(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "Unknown"
	}
	return value
}
