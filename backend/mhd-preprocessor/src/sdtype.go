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
		{Denotation: "lineid", Label: "Line ID", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleTag},
		{Denotation: "routeid", Label: "Route ID", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleTag},
		{Denotation: "service_date", Label: "Service Date", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleTag},
		{Denotation: "trip_id", Label: "Trip ID", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleTag},
		{Denotation: "route_id", Label: "GTFS Route ID", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleTag},
		{Denotation: "direction_id", Label: "Direction ID", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleTag},
		{Denotation: "departure_time", Label: "Departure Time", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleTag},
		{Denotation: "from_stop_id", Label: "From Stop ID", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleTag},
		{Denotation: "to_stop_id", Label: "To Stop ID", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleTag},
		{Denotation: "finalstopid", Label: "Final Stop ID", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleTag},
		{Denotation: "vtype", Label: "Vehicle Type", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleTag},
		{Denotation: "serviceDays", Label: "Service Days", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleTag},
		{Denotation: "globalid", Label: "Global ID", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleField},
		{Denotation: "id", Label: "Vehicle Runtime ID", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleField},
		{Denotation: "linename", Label: "Line Name", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleField},
		{Denotation: "course", Label: "Course", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleField},
		{Denotation: "ltype", Label: "Line Type", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleField},
		{Denotation: "lf", Label: "Low Floor", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleField},
		{Denotation: "laststopid", Label: "Last Stop ID", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleField},
		{Denotation: "lastpostid", Label: "Last Post ID", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleField},
		{Denotation: "lastupdate", Label: "Last Update", Type: sharedModel.SDParameterTypeNumber, Role: sharedModel.SDParameterRoleField},
		{Denotation: "lat", Label: "Latitude", Type: sharedModel.SDParameterTypeNumber, Role: sharedModel.SDParameterRoleField},
		{Denotation: "lng", Label: "Longitude", Type: sharedModel.SDParameterTypeNumber, Role: sharedModel.SDParameterRoleField},
		{Denotation: "bearing", Label: "Bearing", Type: sharedModel.SDParameterTypeNumber, Role: sharedModel.SDParameterRoleField},
		{Denotation: "delay", Label: "Delay", Type: sharedModel.SDParameterTypeNumber, Role: sharedModel.SDParameterRoleField},
		{Denotation: "finalstopname", Label: "Final Stop Name", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleField},
		{Denotation: "serviceid", Label: "Live Service ID", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleField},
		{Denotation: "departuredt", Label: "Departure Datetime", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleField},
		{Denotation: "ocfinalstopid", Label: "OC Final Stop ID", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleField},
		{Denotation: "ocfinalstopname", Label: "OC Final Stop Name", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleField},
		{Denotation: "oclineid", Label: "OC Line ID", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleField},
		{Denotation: "oclinename", Label: "OC Line Name", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleField},
		{Denotation: "ocrouteid", Label: "OC Route ID", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleField},
		{Denotation: "state", Label: "Live State", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleField},
		{Denotation: "segment_from_stop_id", Label: "Segment From Stop ID", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleField},
		{Denotation: "segment_to_stop_id", Label: "Segment To Stop ID", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleField},
		{Denotation: "tmflagtext", Label: "TM Flag Text", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleField},
		{Denotation: "isinactive", Label: "Is Inactive", Type: sharedModel.SDParameterTypeBoolean, Role: sharedModel.SDParameterRoleField},
	}

	message := sharedModel.SDTypeRegistrationRequestISCMessage{
		SDTypeUID:  mhdSDTypeUID,
		Label:      mhdSDTypeLabel,
		Parameters: parameters,
	}

	if err := rabbitmq.PublishJSONBatches(
		client,
		sharedUtils.NewEmptyOptional[string](),
		sharedUtils.NewOptionalOf(sharedConstants.SDTypeRegistrationRequestsQueueName),
		[]sharedModel.SDTypeRegistrationRequestISCMessage{message},
		publishBatchLimit,
	); err != nil {
		log.Printf("[MHD] Failed to register SDType: %v", err)
		return
	}

	log.Printf("[MHD] SDType registration published: %s", mhdSDTypeUID)
}
