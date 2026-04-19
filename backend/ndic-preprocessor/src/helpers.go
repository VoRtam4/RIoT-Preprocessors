package main

import (
	"encoding/json"
	"math/rand"
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
