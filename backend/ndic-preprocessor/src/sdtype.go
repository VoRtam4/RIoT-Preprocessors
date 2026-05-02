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
		{Denotation: "primaryLocationCode", Label: "Primary Location Code", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleTag},
		{Denotation: "secondaryLocationCode", Label: "Secondary Location Code", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleTag},
		{Denotation: "alertCDirection", Label: "Alert-C Direction", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleTag},
		{Denotation: "tmcLocationCode", Label: "TMC Location Code", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleTag},
		{Denotation: "tmcPointName", Label: "TMC Point Name", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleTag},
		{Denotation: "tmcAreaRef", Label: "TMC Area Ref", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleTag},
		{Denotation: "tmcAreaName", Label: "TMC Area Name", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleTag},
		{Denotation: "tmcRoadLCD", Label: "TMC Road LCD", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleTag},
		{Denotation: "tmcSegmentLCD", Label: "TMC Segment LCD", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleTag},
		{Denotation: "tmcRoadNumber", Label: "TMC Road Number", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleTag},
		{Denotation: "tmcRoadName", Label: "TMC Road Name", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleTag},
		{Denotation: "trafficLevelAnyVehicle", Label: "Traffic Level Any Vehicle", Type: sharedModel.SDParameterTypeNumber, Role: sharedModel.SDParameterRoleField},
		{Denotation: "trafficSpeedAnyVehicle", Label: "Traffic Speed Any Vehicle", Type: sharedModel.SDParameterTypeNumber, Role: sharedModel.SDParameterRoleField},
		{Denotation: "travelTimeAnyVehicle", Label: "Travel Time Any Vehicle", Type: sharedModel.SDParameterTypeNumber, Role: sharedModel.SDParameterRoleField},
		{Denotation: "publicationTimestamp", Label: "Publication Timestamp", Type: sharedModel.SDParameterTypeNumber, Role: sharedModel.SDParameterRoleField},
		{Denotation: "isInactive", Label: "Is Inactive", Type: sharedModel.SDParameterTypeBoolean, Role: sharedModel.SDParameterRoleField},
		{Denotation: "tmcLatitude", Label: "TMC Latitude", Type: sharedModel.SDParameterTypeNumber, Role: sharedModel.SDParameterRoleField},
		{Denotation: "tmcLongitude", Label: "TMC Longitude", Type: sharedModel.SDParameterTypeNumber, Role: sharedModel.SDParameterRoleField},
	}

	message := sharedModel.SDTypeRegistrationRequestISCMessage{
		SDTypeUID:  ndicSDTypeUID,
		Label:      ndicSDTypeLabel,
		Parameters: parameters,
	}

	if err := rabbitmq.PublishJSONBatches(
		client,
		sharedUtils.NewEmptyOptional[string](),
		sharedUtils.NewOptionalOf(sharedConstants.SDTypeRegistrationRequestsQueueName),
		[]sharedModel.SDTypeRegistrationRequestISCMessage{message},
		publishBatchLimit,
	); err != nil {
		log.Printf("[NDIC] Failed to register SDType: %v", err)
		return
	}

	log.Printf("[NDIC] SDType registration published: %s", ndicSDTypeUID)
}
