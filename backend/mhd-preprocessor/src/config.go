package main

import (
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultWSURL                 = "wss://gis.brno.cz/geoevent/ws/services/stream_kordis_26/StreamServer/subscribe"
	defaultGTFSURL               = "https://kordis-jmk.cz/gtfs/gtfs.zip"
	defaultGTFSLocation          = "Europe/Prague"
	mhdSDTypeUID                 = "MHD_TRIP"
	mhdSDTypeLabel               = "MHD Trip"
	defaultGTFSRefreshInterval   = 12 * time.Hour
	defaultTripEndReserve        = 10 * time.Minute
	defaultMatchingWindow        = 30 * time.Minute
	defaultStartupGracePeriod    = 60 * time.Second
	defaultSyntheticJitter       = 500 * time.Millisecond
	defaultClosingLoopInterval   = 2 * time.Second
	defaultReconnectDelay        = 5 * time.Second
	defaultActivePublishInterval = 90 * time.Second
)

type appConfig struct {
	WSURL                 string
	GTFSURL               string
	GTFSLocation          *time.Location
	GTFSRefreshInterval   time.Duration
	TripEndReserve        time.Duration
	MatchingWindow        time.Duration
	StartupGracePeriod    time.Duration
	SyntheticJitter       time.Duration
	ClosingLoopInterval   time.Duration
	ReconnectDelay        time.Duration
	ActivePublishInterval time.Duration
}

func loadConfig() appConfig {
	locationName := getEnv("MHD_GTFS_TIMEZONE", defaultGTFSLocation)
	location, err := time.LoadLocation(locationName)
	if err != nil {
		location = time.FixedZone(defaultGTFSLocation, 3600)
	}

	return appConfig{
		WSURL:                 normalizeStreamWSURL(getEnv("MHD_WS_URL", defaultWSURL)),
		GTFSURL:               getEnv("MHD_GTFS_URL", defaultGTFSURL),
		GTFSLocation:          location,
		GTFSRefreshInterval:   getEnvDurationMinutes("MHD_GTFS_REFRESH_MINUTES", defaultGTFSRefreshInterval),
		TripEndReserve:        getEnvDurationMinutes("MHD_TRIP_END_RESERVE_MINUTES", defaultTripEndReserve),
		MatchingWindow:        getEnvDurationMinutes("MHD_MATCHING_WINDOW_MINUTES", defaultMatchingWindow),
		StartupGracePeriod:    getEnvDurationSeconds("MHD_STARTUP_GRACE_SECONDS", defaultStartupGracePeriod),
		SyntheticJitter:       getEnvDurationMilliseconds("MHD_SYNTHETIC_JITTER_MS", defaultSyntheticJitter),
		ClosingLoopInterval:   getEnvDurationSeconds("MHD_CLOSING_LOOP_SECONDS", defaultClosingLoopInterval),
		ReconnectDelay:        getEnvDurationSeconds("MHD_RECONNECT_DELAY_SECONDS", defaultReconnectDelay),
		ActivePublishInterval: getEnvDurationSeconds("MHD_ACTIVE_PUBLISH_SECONDS", defaultActivePublishInterval),
	}
}

func normalizeStreamWSURL(value string) string {
	if value == "" {
		return defaultWSURL
	}
	if strings.HasSuffix(value, "/subscribe") {
		return value
	}
	return strings.TrimRight(value, "/") + "/subscribe"
}

func getEnv(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvDurationMinutes(key string, fallback time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			return time.Duration(parsed) * time.Minute
		}
	}
	return fallback
}

func getEnvDurationSeconds(key string, fallback time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			return time.Duration(parsed) * time.Second
		}
	}
	return fallback
}

func getEnvDurationMilliseconds(key string, fallback time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed >= 0 {
			return time.Duration(parsed) * time.Millisecond
		}
	}
	return fallback
}
