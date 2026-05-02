package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/MichalBures-OG/bp-bures-RIoT-commons/src/rabbitmq"
	"github.com/MichalBures-OG/bp-bures-RIoT-commons/src/sharedConstants"
	"github.com/MichalBures-OG/bp-bures-RIoT-commons/src/sharedModel"
	"github.com/MichalBures-OG/bp-bures-RIoT-commons/src/sharedUtils"
)

func registerSDType(client rabbitmq.Client) {
	parameters := []sharedModel.SDParameter{
		{Denotation: "country", Label: "Country", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleTag},
		{Denotation: "city", Label: "City", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleTag},
		{Denotation: "street", Label: "Street", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleTag},
		{Denotation: "roadType", Label: "Road Type", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleField},
		{Denotation: "segmentId", Label: "Segment ID", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleTag},
		{Denotation: "fromNode", Label: "From Node", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleTag},
		{Denotation: "toNode", Label: "To Node", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleTag},
		{Denotation: "isForward", Label: "Is Forward", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleTag},
		{Denotation: "delay", Label: "Delay", Type: sharedModel.SDParameterTypeNumber, Role: sharedModel.SDParameterRoleField},
		{Denotation: "length", Label: "Length", Type: sharedModel.SDParameterTypeNumber, Role: sharedModel.SDParameterRoleField},
		{Denotation: "level", Label: "Level", Type: sharedModel.SDParameterTypeNumber, Role: sharedModel.SDParameterRoleField},
		{Denotation: "speed", Label: "Speed", Type: sharedModel.SDParameterTypeNumber, Role: sharedModel.SDParameterRoleField},
		{Denotation: "speedKPH", Label: "Speed KPH", Type: sharedModel.SDParameterTypeNumber, Role: sharedModel.SDParameterRoleField},
		{Denotation: "jamCount", Label: "Jam Count", Type: sharedModel.SDParameterTypeNumber, Role: sharedModel.SDParameterRoleField},
		{Denotation: "pubMillisLatest", Label: "Latest Publication Time", Type: sharedModel.SDParameterTypeNumber, Role: sharedModel.SDParameterRoleField},
		{Denotation: "rawJams", Label: "Raw Jams", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleField},
	}

	message := sharedModel.SDTypeRegistrationRequestISCMessage{
		SDTypeUID:  wazeSDTypeUID,
		Label:      wazeSDTypeLabel,
		Parameters: parameters,
	}

	if err := rabbitmq.PublishJSONBatches(
		client,
		sharedUtils.NewEmptyOptional[string](),
		sharedUtils.NewOptionalOf(sharedConstants.SDTypeRegistrationRequestsQueueName),
		[]sharedModel.SDTypeRegistrationRequestISCMessage{message},
		publishBatchLimit,
	); err != nil {
		log.Printf("[WAZE] Failed to register SDType: %v", err)
		return
	}

	log.Printf("[WAZE] SDType registration published: %s", wazeSDTypeUID)
}

func checkForSetOfSDInstancesUpdates(client rabbitmq.Client) {
	rabbitmq.ConsumeJSONMessages[sharedModel.SDInstanceConfigurationUpdateISCMessage](
		client,
		sharedConstants.SetOfSDInstancesUpdatesQueueName,
		func(messagePayload sharedModel.SDInstanceConfigurationUpdateISCMessage) error {
			sdInstancesMutex.Lock()
			sdInstances = sharedUtils.NewSetFromSlice(messagePayload)
			sdInstancesMutex.Unlock()
			return nil
		},
	)
}

func fetchAndProcessWazeData(client rabbitmq.Client, config appConfig) {
	resp, err := http.Get(config.WazeURL)
	if err != nil {
		log.Printf("[WAZE] Failed to fetch data: %v", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[WAZE] Failed to read response: %v", err)
		return
	}

	var feed wazeFeed
	if err := json.Unmarshal(body, &feed); err != nil {
		log.Printf("[WAZE] Failed to decode feed: %v", err)
		return
	}

	currentDevices := make(map[string]deviceSnapshot)
	aggregates := make(map[string]*deviceAggregate)
	stateMessages := make([]sharedModel.KPIFulfillmentCheckRequestISCMessage, 0)
	registrationMessages := make([]sharedModel.SDInstanceRegistrationRequestISCMessage, 0)

	for _, jam := range feed.Jams {
		for _, aggregate := range buildAggregatesFromJam(jam) {
			existing, exists := aggregates[aggregate.UID]
			if !exists {
				aggregates[aggregate.UID] = aggregate
				continue
			}

			mergeAggregate(existing, aggregate)
		}
	}

	for _, aggregate := range aggregates {
		if determineSDInstanceScenario(aggregate.UID) == "unknown" {
			registrationMessages = append(registrationMessages, buildSDInstanceRegistrationMessage(aggregate.UID, aggregate.Label, aggregate.EventTime))
		}

		currentDevices[aggregate.UID] = deviceSnapshot{
			Label: aggregate.Label,
			Tags:  cloneTags(aggregate.Tags),
		}

		if !wasDeviceActive(aggregate.UID) && shouldPublishStartupZeroDelay(aggregate.EventTime) {
			stateMessages = append(stateMessages, buildStateMessage(aggregate.UID, buildStartupZeroDelayEventTime(aggregate.UID, aggregate.EventTime), buildZeroDelayParams(aggregate.Tags)))
		}

		stateMessages = append(stateMessages, buildStateMessage(aggregate.UID, aggregate.EventTime, buildActiveParams(aggregate)))
	}

	now := time.Now().UTC()

	activeDevicesMu.Lock()
	previousDevices := activeDevices
	activeDevices = currentDevices
	activeDevicesMu.Unlock()

	for uid, snapshot := range previousDevices {
		if _, stillActive := currentDevices[uid]; stillActive {
			continue
		}

		stateMessages = append(stateMessages, buildStateMessage(uid, now, buildZeroDelayParams(snapshot.Tags)))
	}
	if len(registrationMessages) > 0 {
		publishSDInstanceRegistrations(client, registrationMessages)
		sdInstancesMutex.Lock()
		for _, message := range registrationMessages {
			sdInstances.Add(sharedModel.SDInstanceInfo{
				SDInstanceUID:   message.SDInstanceUID,
				ConfirmedByUser: false,
			})
		}
		sdInstancesMutex.Unlock()
	}
	if len(stateMessages) > 0 {
		publishStates(client, stateMessages)
	}
}

func determineSDInstanceScenario(uid string) string {
	sdInstancesMutex.Lock()
	defer sdInstancesMutex.Unlock()

	if sdInstances.Contains(sharedModel.SDInstanceInfo{SDInstanceUID: uid, ConfirmedByUser: true}) {
		return "confirmed"
	}
	if sdInstances.Contains(sharedModel.SDInstanceInfo{SDInstanceUID: uid, ConfirmedByUser: false}) {
		return "notYetConfirmed"
	}
	return "unknown"
}

func wasDeviceActive(uid string) bool {
	activeDevicesMu.Lock()
	defer activeDevicesMu.Unlock()

	_, exists := activeDevices[uid]
	return exists
}
