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
	client := rabbitmq.NewClient()
	defer client.Dispose()

	registerSDType(client)
	time.Sleep(5 * time.Second)
	go checkForSetOfSDInstancesUpdates(client)

	if config.HTTPEnabled {
		go startHTTPServer(client, config)
	}

	if !config.PollEnabled {
		select {}
	}

	for {
		publicationTime := fetchAndProcessNDICData(client, config)
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
