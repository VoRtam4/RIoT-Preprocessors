package main

import (
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/MichalBures-OG/bp-bures-RIoT-commons/src/rabbitmq"
	"github.com/MichalBures-OG/bp-bures-RIoT-commons/src/sharedConstants"
	"github.com/MichalBures-OG/bp-bures-RIoT-commons/src/sharedModel"
	"github.com/MichalBures-OG/bp-bures-RIoT-commons/src/sharedUtils"
)

const publishBatchLimit = 500

func buildTripTags(definition *tripDefinition, occurrence tripOccurrence, record *liveRecord, segment *segmentMatch) map[string]string {
	serviceDaysJSON, err := json.Marshal(definition.ServiceDays)
	if err != nil {
		serviceDaysJSON = []byte("[]")
	}

	tags := map[string]string{
		"lineid":         firstNonEmpty(record.LineID, definition.LineID),
		"routeid":        firstNonEmpty(record.LiveRouteID, definition.LiveRouteID),
		"service_date":   isoDateKey(occurrence.ServiceDate),
		"trip_id":        firstNonEmpty(occurrence.TripID, definition.TripID),
		"route_id":       definition.RouteID,
		"direction_id":   definition.DirectionID,
		"departure_time": definition.DepartureTime,
		"from_stop_id":   definition.FromStopID,
		"to_stop_id":     definition.ToStopID,
		"finalstopid":    firstNonEmpty(record.FinalStopID, definition.ToStopID),
		"vtype":          record.VehicleType,
		"serviceDays":    string(serviceDaysJSON),
	}

	return tags
}

func buildActiveParams(tags map[string]string, record *liveRecord, segment *segmentMatch) map[string]interface{} {
	params := tagsToParams(tags)
	params["globalid"] = record.GlobalID
	params["id"] = record.VehicleRuntimeID
	params["linename"] = record.LineName
	params["course"] = record.Course
	params["ltype"] = record.LineType
	params["lf"] = record.LowFloor
	params["laststopid"] = record.LastStopID
	params["lastupdate"] = float64(record.SourceTimestamp.UnixMilli())
	params["lat"] = record.GeometryLat
	params["lng"] = record.GeometryLng
	params["bearing"] = record.Bearing
	params["delay"] = record.Delay
	if segment != nil {
		params["segment_from_stop_id"] = segment.From.ID
		params["segment_to_stop_id"] = segment.To.ID
	}
	params["isinactive"] = false
	return params
}

func buildInactiveParams(tags map[string]string) map[string]interface{} {
	params := tagsToParams(tags)
	params["isinactive"] = true
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
		SDTypeUID:     mhdSDTypeUID,
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
		log.Printf("[MHD] Failed to publish state tuple: %v", err)
	}
}

func buildSDInstanceRegistrationMessage(uid string, label string) sharedModel.SDInstanceRegistrationRequestISCMessage {
	return sharedModel.SDInstanceRegistrationRequestISCMessage{
		EventTime:     time.Now().UTC(),
		Label:         label,
		SDInstanceUID: uid,
		SDTypeUID:     mhdSDTypeUID,
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
		log.Printf("[MHD] Failed to publish SD instance registration tuple: %v", err)
	}
}

func registerInstanceIfNeeded(client rabbitmq.Client, uid string, label string) {
	if !shouldRegisterInstance(uid) {
		return
	}

	publishSDInstanceRegistrations(client, []sharedModel.SDInstanceRegistrationRequestISCMessage{buildSDInstanceRegistrationMessage(uid, label)})

	markInstanceRegistered(uid)
	log.Printf("[MHD] Registered SD instance: %s", uid)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
