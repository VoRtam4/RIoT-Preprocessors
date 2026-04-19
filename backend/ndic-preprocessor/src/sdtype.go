package main

import (
	"log"

	"github.com/MichalBures-OG/bp-bures-RIoT-commons/src/rabbitmq"
	"github.com/MichalBures-OG/bp-bures-RIoT-commons/src/sharedConstants"
	"github.com/MichalBures-OG/bp-bures-RIoT-commons/src/sharedModel"
	"github.com/MichalBures-OG/bp-bures-RIoT-commons/src/sharedUtils"
)

func registerSDType(client rabbitmq.Client) {
	parameters := []sharedModel.SDParameter{
		{Denotation: "sourceIdentification", Label: "Source Identification", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleTag},
		{Denotation: "trafficLevelAnyVehicle", Label: "Traffic Level Any Vehicle", Type: sharedModel.SDParameterTypeNumber, Role: sharedModel.SDParameterRoleField},
		{Denotation: "trafficSpeedAnyVehicle", Label: "Traffic Speed Any Vehicle", Type: sharedModel.SDParameterTypeNumber, Role: sharedModel.SDParameterRoleField},
		{Denotation: "travelTimeAnyVehicle", Label: "Travel Time Any Vehicle", Type: sharedModel.SDParameterTypeNumber, Role: sharedModel.SDParameterRoleField},
		{Denotation: "publicationTimestamp", Label: "Publication Timestamp", Type: sharedModel.SDParameterTypeNumber, Role: sharedModel.SDParameterRoleField},
		{Denotation: "isInactive", Label: "Is Inactive", Type: sharedModel.SDParameterTypeBoolean, Role: sharedModel.SDParameterRoleField},
	}

	message := sharedModel.SDTypeRegistrationRequestISCMessage{
		SDTypeUID:  ndicSDTypeUID,
		Label:      ndicSDTypeLabel,
		Parameters: parameters,
	}

	jsonResult := sharedUtils.SerializeToJSON(message)
	if jsonResult.IsFailure() {
		log.Printf("[NDIC] Failed to serialize SDType registration: %v", jsonResult.GetError())
		return
	}

	if err := client.PublishJSONMessage(
		sharedUtils.NewEmptyOptional[string](),
		sharedUtils.NewOptionalOf(sharedConstants.SDTypeRegistrationRequestsQueueName),
		jsonResult.GetPayload(),
	); err != nil {
		log.Printf("[NDIC] Failed to register SDType: %v", err)
		return
	}

	log.Printf("[NDIC] SDType registration published: %s", ndicSDTypeUID)
}
