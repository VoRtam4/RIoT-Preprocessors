package main

import (
	"log"
	"os"
	"time"

	"github.com/MichalBures-OG/bp-bures-RIoT-commons/src/rabbitmq"
)

func main() {
	log.SetOutput(os.Stderr)

	config := loadConfig()
	log.Printf("[NDIC] Configured source URL: %s", config.NDICURL)
	client := rabbitmq.NewClient()
	defer client.Dispose()
	enricher := newTMCEnricher(config)

	registerSDType(client)
	time.Sleep(5 * time.Second)
	go checkForSetOfSDInstancesUpdates(client)

	for {
		publicationTime := fetchAndProcessNDICData(client, config, enricher)
		time.Sleep(nextFetchDelay(config.FetchDelay, publicationTime))
	}
}

func nextFetchDelay(baseDelay time.Duration, publicationTime time.Time) time.Duration {
	if publicationTime.IsZero() {
		return baseDelay
	}
	nextFetch := publicationTime.Add(baseDelay)
	delay := time.Until(nextFetch)
	if delay < 0 {
		return 0
	}
	return delay
}
