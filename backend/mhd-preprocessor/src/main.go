package main

import (
	"log"
	"os"
	"time"

	"github.com/MichalBures-OG/bp-bures-RIoT-commons/src/rabbitmq"
	"github.com/MichalBures-OG/bp-bures-RIoT-commons/src/sharedModel"
)

func main() {
	log.SetOutput(os.Stderr)

	config := loadConfig()
	log.Printf("[MHD] Configured WebSocket endpoints: %v", config.WSURLs)
	client := rabbitmq.NewClient()
	defer client.Dispose()

	registerSDType(client)
	time.Sleep(5 * time.Second)
	go checkForSetOfSDInstancesUpdates(client)

	store := newGTFSStore(config)
	if err := store.refresh(time.Now().UTC()); err != nil {
		log.Fatalf("[MHD] Initial GTFS refresh failed: %v", err)
	}
	registerCurrentWeekInstances(client, store)

	go refreshGTFSPolling(client, store, config)
	go closingLoop(client, config)
	runWebSocketLoop(client, store, config)
}

func refreshGTFSPolling(client rabbitmq.Client, store *GTFSStore, config appConfig) {
	ticker := time.NewTicker(config.GTFSRefreshInterval)
	defer ticker.Stop()

	for range ticker.C {
		if err := store.refresh(time.Now().UTC()); err != nil {
			log.Printf("[MHD] GTFS refresh failed: %v", err)
			continue
		}
		registerCurrentWeekInstances(client, store)
	}
}

func registerCurrentWeekInstances(client rabbitmq.Client, store *GTFSStore) {
	messages := make([]sharedModel.SDInstanceRegistrationRequestISCMessage, 0)
	for _, uid := range store.weekUIDs() {
		if store.isWeekUIDRegistered(uid) {
			continue
		}
		store.mu.RLock()
		definition := store.definitionsByUID[uid]
		store.mu.RUnlock()
		if definition == nil {
			continue
		}
		if shouldRegisterInstance(definition.UID) {
			messages = append(messages, buildSDInstanceRegistrationMessage(definition.UID, definition.Label))
			markInstanceRegistered(definition.UID)
		}
		store.markWeekUIDRegistered(uid)
	}
	publishSDInstanceRegistrations(client, messages)
}

func closingLoop(client rabbitmq.Client, config appConfig) {
	ticker := time.NewTicker(config.ClosingLoopInterval)
	defer ticker.Stop()

	for now := range ticker.C {
		closeExpiredInstances(client, config, now.UTC())
	}
}
