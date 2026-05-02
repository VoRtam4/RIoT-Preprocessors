package main

import (
	"net/url"
	"os"
	"path"
	"slices"
	"strings"
	"time"
)

const (
	defaultWSURL                 = "wss://gis.brno.cz/geoevent/ws/services/stream_kordis_26/StreamServer/subscribe?outSR=4326"
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
	defaultActivePublishInterval = 1 * time.Minute
	defaultNoDataReconnect       = 45 * time.Second
)

type appConfig struct {
	WSURLs                []string
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
	NoDataReconnect       time.Duration
}

func loadConfig() appConfig {
	location, err := time.LoadLocation(defaultGTFSLocation)
	if err != nil {
		location = time.FixedZone(defaultGTFSLocation, 3600)
	}

	return appConfig{
		WSURLs:                loadStreamWSURLs(),
		GTFSURL:               getEnv("MHD_GTFS_URL", defaultGTFSURL),
		GTFSLocation:          location,
		GTFSRefreshInterval:   defaultGTFSRefreshInterval,
		TripEndReserve:        defaultTripEndReserve,
		MatchingWindow:        defaultMatchingWindow,
		StartupGracePeriod:    defaultStartupGracePeriod,
		SyntheticJitter:       defaultSyntheticJitter,
		ClosingLoopInterval:   defaultClosingLoopInterval,
		ReconnectDelay:        defaultReconnectDelay,
		ActivePublishInterval: defaultActivePublishInterval,
		NoDataReconnect:       defaultNoDataReconnect,
	}
}

func loadStreamWSURLs() []string {
	rawCandidates := make([]string, 0, 2)

	if value := getEnv("MHD_WS_URL", ""); value != "" {
		rawCandidates = append(rawCandidates, value)
	}
	rawCandidates = append(rawCandidates, defaultWSURL)

	normalized := make([]string, 0, len(rawCandidates))
	for _, candidate := range rawCandidates {
		value := normalizeStreamWSURL(candidate)
		if value == "" || slices.Contains(normalized, value) {
			continue
		}
		normalized = append(normalized, value)
	}

	if len(normalized) == 0 {
		return []string{normalizeStreamWSURL(defaultWSURL)}
	}

	return normalized
}

func normalizeStreamWSURL(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		if strings.HasSuffix(value, "/subscribe") {
			return value
		}
		return strings.TrimRight(value, "/") + "/subscribe"
	}

	if !strings.HasSuffix(strings.ToLower(parsed.Path), "/subscribe") {
		parsed.Path = path.Clean(strings.TrimRight(parsed.Path, "/") + "/subscribe")
	}

	return parsed.String()
}

func getEnv(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
