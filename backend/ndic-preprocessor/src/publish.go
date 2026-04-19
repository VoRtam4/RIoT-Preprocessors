package main

import (
	"log"
	"time"

	"github.com/MichalBures-OG/bp-bures-RIoT-commons/src/rabbitmq"
	"github.com/MichalBures-OG/bp-bures-RIoT-commons/src/sharedConstants"
	"github.com/MichalBures-OG/bp-bures-RIoT-commons/src/sharedModel"
	"github.com/MichalBures-OG/bp-bures-RIoT-commons/src/sharedUtils"
)

func buildTags(snapshot *ndicSnapshot) map[string]string {
	return map[string]string{
		"sourceIdentification": snapshot.SourceIdentification,
	}
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

func publishState(client rabbitmq.Client, uid string, eventTime time.Time, params map[string]interface{}) {
	message := sharedModel.KPIFulfillmentCheckRequestISCMessage{
		EventTime:     eventTime.UTC(),
		SDInstanceUID: uid,
		SDTypeUID:     ndicSDTypeUID,
		Parameters:    params,
	}

	jsonResult := sharedUtils.SerializeToJSON(message)
	if jsonResult.IsFailure() {
		log.Printf("[NDIC] Failed to serialize state for %s: %v", uid, jsonResult.GetError())
		return
	}

	if err := client.PublishJSONMessage(
		sharedUtils.NewEmptyOptional[string](),
		sharedUtils.NewOptionalOf(sharedConstants.KPIFulfillmentCheckRequestsQueueName),
		jsonResult.GetPayload(),
	); err != nil {
		log.Printf("[NDIC] Failed to publish state for %s: %v", uid, err)
	}
}

func registerInstanceIfNeeded(client rabbitmq.Client, uid string, label string) {
	if !shouldRegisterInstance(uid) {
		return
	}

	message := sharedModel.SDInstanceRegistrationRequestISCMessage{
		EventTime:     time.Now().UTC(),
		Label:         label,
		SDInstanceUID: uid,
		SDTypeUID:     ndicSDTypeUID,
	}

	jsonMessage := sharedUtils.SerializeToJSON(message)
	if jsonMessage.IsFailure() {
		log.Printf("[NDIC] Failed to serialize SD instance registration for %s: %v", uid, jsonMessage.GetError())
		return
	}

	if err := client.PublishJSONMessage(
		sharedUtils.NewEmptyOptional[string](),
		sharedUtils.NewOptionalOf(sharedConstants.SDInstanceRegistrationRequestsQueueName),
		jsonMessage.GetPayload(),
	); err != nil {
		log.Printf("[NDIC] Failed to register SD instance %s: %v", uid, err)
		return
	}

	markInstanceRegistered(uid)
	log.Printf("[NDIC] Registered SD instance: %s", uid)
}
