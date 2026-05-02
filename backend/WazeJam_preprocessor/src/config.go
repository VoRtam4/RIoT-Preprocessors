package main

import "os"

const defaultWazeURL = "https://www.waze.com/row-partnerhub-api/partners/16198912488/waze-feeds/9c8b4163-e3c2-436f-86b7-3db2058ce7a1?format=1"

type appConfig struct {
	WazeURL string
}

func loadConfig() appConfig {
	return appConfig{
		WazeURL: getEnv("WAZE_URL", defaultWazeURL),
	}
}

func getEnv(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
