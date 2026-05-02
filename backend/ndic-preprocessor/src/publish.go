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

func buildTags(snapshot *ndicSnapshot) map[string]string {
	tags := map[string]string{
		"sourceIdentification": snapshot.SourceIdentification,
	}
	if snapshot.PrimaryLocationCode != "" {
		tags["primaryLocationCode"] = snapshot.PrimaryLocationCode
	}
	if snapshot.SecondaryLocationCode != "" {
		tags["secondaryLocationCode"] = snapshot.SecondaryLocationCode
	}
	if snapshot.AlertCDirection != "" {
		tags["alertCDirection"] = snapshot.AlertCDirection
	}
	if snapshot.TMCMetadata != nil {
		if snapshot.TMCMetadata.LocationCode != "" {
			tags["tmcLocationCode"] = snapshot.TMCMetadata.LocationCode
		}
		if snapshot.TMCMetadata.PointName != "" {
			tags["tmcPointName"] = snapshot.TMCMetadata.PointName
		}
		if snapshot.TMCMetadata.AreaRef != "" {
			tags["tmcAreaRef"] = snapshot.TMCMetadata.AreaRef
		}
		if snapshot.TMCMetadata.AreaName != "" {
			tags["tmcAreaName"] = snapshot.TMCMetadata.AreaName
		}
		if snapshot.TMCMetadata.RoadLCD != "" {
			tags["tmcRoadLCD"] = snapshot.TMCMetadata.RoadLCD
		}
		if snapshot.TMCMetadata.SegmentLCD != "" {
			tags["tmcSegmentLCD"] = snapshot.TMCMetadata.SegmentLCD
		}
		if snapshot.TMCMetadata.RoadNumber != "" {
			tags["tmcRoadNumber"] = snapshot.TMCMetadata.RoadNumber
		}
		if snapshot.TMCMetadata.RoadName != "" {
			tags["tmcRoadName"] = snapshot.TMCMetadata.RoadName
		}
	}
	return tags
}

func buildActiveParams(tags map[string]string, snapshot *ndicSnapshot, publicationTime time.Time) map[string]interface{} {
	params := tagsToParams(tags)
	if snapshot.TrafficLevelAnyVehicle != nil {
		params["trafficLevelAnyVehicle"] = float64(*snapshot.TrafficLevelAnyVehicle)
	}
	if snapshot.TrafficSpeedAnyVehicle != nil {
		params["trafficSpeedAnyVehicle"] = *snapshot.TrafficSpeedAnyVehicle
	}
	if snapshot.TravelTimeAnyVehicle != nil {
		params["travelTimeAnyVehicle"] = *snapshot.TravelTimeAnyVehicle
	}
	params["publicationTimestamp"] = float64(publicationTime.UnixMilli())
	params["isInactive"] = false
	if snapshot.TMCMetadata != nil {
		if snapshot.TMCMetadata.Latitude != nil {
			params["tmcLatitude"] = *snapshot.TMCMetadata.Latitude
		}
		if snapshot.TMCMetadata.Longitude != nil {
			params["tmcLongitude"] = *snapshot.TMCMetadata.Longitude
		}
	}
	return params
}

func buildInactiveParams(tags map[string]string) map[string]interface{} {
	params := tagsToParams(tags)
	params["isInactive"] = true
	return params
}

func tagsToParams(tags map[string]string) map[string]interface{} {
	params := make(map[string]interface{}, len(tags))
	for key, value := range tags {
		params[key] = value
	}
	return params
}

func buildStateMessage(uid string, eventTime time.Time, params map[string]interface{}) sharedModel.KPIFulfillmentCheckRequestISCMessage {
	return sharedModel.KPIFulfillmentCheckRequestISCMessage{
		EventTime:     eventTime.UTC(),
		SDInstanceUID: uid,
		SDTypeUID:     ndicSDTypeUID,
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
		log.Printf("[NDIC] Failed to publish state tuple: %v", err)
	}
}

func buildSDInstanceRegistrationMessage(uid string, label string) sharedModel.SDInstanceRegistrationRequestISCMessage {
	return sharedModel.SDInstanceRegistrationRequestISCMessage{
		EventTime:     time.Now().UTC(),
		Label:         label,
		SDInstanceUID: uid,
		SDTypeUID:     ndicSDTypeUID,
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
		log.Printf("[NDIC] Failed to publish SD instance registration tuple: %v", err)
	}
}

func registerInstanceIfNeeded(client rabbitmq.Client, uid string, label string) {
	if !shouldRegisterInstance(uid) {
		return
	}

	publishSDInstanceRegistrations(client, []sharedModel.SDInstanceRegistrationRequestISCMessage{buildSDInstanceRegistrationMessage(uid, label)})

	markInstanceRegistered(uid)
	log.Printf("[NDIC] Registered SD instance: %s", uid)
}
