package main

import (
	"os"
	"strconv"
	"time"
)

const (
	defaultNDICURL             = "http://80.211.200.65:8000/api/latest"
	defaultFetchDelay          = 310 * time.Second
	defaultStartupGracePeriod  = 60 * time.Second
	defaultSyntheticJitter     = 500 * time.Millisecond
	defaultHTTPListenAddr      = ":8000"
	defaultRawStorageDir       = "/tmp/ndic_messages"
	ndicSDTypeUID              = "NDIC_TRAFFIC"
	ndicSDTypeLabel            = "NDIC Traffic"
)

type appConfig struct {
	NDICURL             string
	PollEnabled         bool
	HTTPEnabled         bool
	HTTPListenAddr      string
	BasicAuthUsername   string
	BasicAuthPassword   string
	RawStorageDir       string
	FetchDelay          time.Duration
	StartupGracePeriod  time.Duration
	SyntheticJitter     time.Duration
}

func loadConfig() appConfig {
	return appConfig{
		NDICURL:            getEnv("NDIC_URL", defaultNDICURL),
		PollEnabled:        getEnvBool("NDIC_POLL_ENABLED", false),
		HTTPEnabled:        getEnvBool("NDIC_HTTP_ENABLED", true),
		HTTPListenAddr:     getEnv("NDIC_HTTP_LISTEN_ADDR", defaultHTTPListenAddr),
		BasicAuthUsername:  os.Getenv("NDIC_HTTP_USERNAME"),
		BasicAuthPassword:  os.Getenv("NDIC_HTTP_PASSWORD"),
		RawStorageDir:      getEnv("NDIC_RAW_STORAGE_DIR", defaultRawStorageDir),
		FetchDelay:         getEnvDurationSeconds("NDIC_FETCH_DELAY_SECONDS", defaultFetchDelay),
		StartupGracePeriod: getEnvDurationSeconds("NDIC_STARTUP_GRACE_SECONDS", defaultStartupGracePeriod),
		SyntheticJitter:    getEnvDurationMilliseconds("NDIC_SYNTHETIC_JITTER_MS", defaultSyntheticJitter),
	}
}

func getEnv(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
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

func getEnvBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}
