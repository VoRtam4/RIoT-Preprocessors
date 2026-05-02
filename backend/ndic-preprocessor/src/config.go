package main

import (
	"os"
	"time"
)

const (
	defaultNDICURL            = "http://80.211.200.65:8000/api/latest"
	defaultTMCDir             = "/app/ndic-preprocessor/static_data/tmc"
	defaultFetchDelay         = 310 * time.Second
	defaultStartupGracePeriod = 60 * time.Second
	defaultSyntheticJitter    = 500 * time.Millisecond
	ndicSDTypeUID             = "NDIC_TRAFFIC"
	ndicSDTypeLabel           = "NDIC Traffic"
)

type appConfig struct {
	NDICURL            string
	TMCDir             string
	FetchDelay         time.Duration
	StartupGracePeriod time.Duration
	SyntheticJitter    time.Duration
}

func loadConfig() appConfig {
	return appConfig{
		NDICURL:            getEnv("NDIC_URL", defaultNDICURL),
		TMCDir:             defaultTMCDir,
		FetchDelay:         defaultFetchDelay,
		StartupGracePeriod: defaultStartupGracePeriod,
		SyntheticJitter:    defaultSyntheticJitter,
	}
}

func getEnv(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
