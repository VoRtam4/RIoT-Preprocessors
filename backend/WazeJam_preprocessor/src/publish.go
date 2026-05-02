package main

import (
	"log"
	"time"

	"github.com/MichalBures-OG/bp-bures-RIoT-commons/src/rabbitmq"
	"github.com/MichalBures-OG/bp-bures-RIoT-commons/src/sharedConstants"
	"github.com/MichalBures-OG/bp-bures-RIoT-commons/src/sharedModel"
	"github.com/MichalBures-OG/bp-bures-RIoT-commons/src/sharedUtils"
)

const publishBatchLimit = 500

func buildStateMessage(uid string, eventTime time.Time, params map[string]interface{}) sharedModel.KPIFulfillmentCheckRequestISCMessage {
	return sharedModel.KPIFulfillmentCheckRequestISCMessage{
		EventTime:     eventTime.UTC(),
		SDInstanceUID: uid,
		SDTypeUID:     wazeSDTypeUID,
		Parameters:    params,
	}
}

func publishStates(client rabbitmq.Client, messages []sharedModel.KPIFulfillmentCheckRequestISCMessage) {
	if err := rabbitmq.PublishJSONBatches(
		client,
		sharedUtils.NewEmptyOptional[string](),
		sharedUtils.NewOptionalOf(sharedConstants.KPIFulfillmentCheckRequestsQueueName),
		messages,
		publishBatchLimit,
	); err != nil {
		log.Printf("[WAZE] Failed to publish state tuple: %v", err)
	}
}

func buildSDInstanceRegistrationMessage(uid string, label string, eventTime time.Time) sharedModel.SDInstanceRegistrationRequestISCMessage {
	return sharedModel.SDInstanceRegistrationRequestISCMessage{
		EventTime:     eventTime.UTC(),
		Label:         label,
		SDInstanceUID: uid,
		SDTypeUID:     wazeSDTypeUID,
	}
}

func publishSDInstanceRegistrations(client rabbitmq.Client, messages []sharedModel.SDInstanceRegistrationRequestISCMessage) {
	if err := rabbitmq.PublishJSONBatches(
		client,
		sharedUtils.NewEmptyOptional[string](),
		sharedUtils.NewOptionalOf(sharedConstants.SDInstanceRegistrationRequestsQueueName),
		messages,
		publishBatchLimit,
	); err != nil {
		log.Printf("[WAZE] Failed to publish SDInstance registration tuple: %v", err)
	}
}
