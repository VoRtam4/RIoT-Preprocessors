package main

import (
	"encoding/json"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/MichalBures-OG/bp-bures-RIoT-commons/src/rabbitmq"
	"github.com/MichalBures-OG/bp-bures-RIoT-commons/src/sharedConstants"
	"github.com/MichalBures-OG/bp-bures-RIoT-commons/src/sharedModel"
	"github.com/MichalBures-OG/bp-bures-RIoT-commons/src/sharedUtils"
)

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

	if segment != nil {
		tags["segment_index"] = strconv.Itoa(segment.Index)
		tags["segment_from_stop_id"] = segment.From.ID
		tags["segment_to_stop_id"] = segment.To.ID
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
		params["segment_progress"] = segment.Progress
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

func publishState(client rabbitmq.Client, uid string, eventTime time.Time, params map[string]interface{}) {
	message := sharedModel.KPIFulfillmentCheckRequestISCMessage{
		EventTime:     eventTime.UTC(),
		SDInstanceUID: uid,
		SDTypeUID:     mhdSDTypeUID,
		Parameters:    params,
	}

	jsonResult := sharedUtils.SerializeToJSON(message)
	if jsonResult.IsFailure() {
		log.Printf("[MHD] Failed to serialize state for %s: %v", uid, jsonResult.GetError())
		return
	}

	if err := client.PublishJSONMessage(
		sharedUtils.NewEmptyOptional[string](),
		sharedUtils.NewOptionalOf(sharedConstants.KPIFulfillmentCheckRequestsQueueName),
		jsonResult.GetPayload(),
	); err != nil {
		log.Printf("[MHD] Failed to publish state for %s: %v", uid, err)
		return
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
		SDTypeUID:     mhdSDTypeUID,
	}

	jsonMessage := sharedUtils.SerializeToJSON(message)
	if jsonMessage.IsFailure() {
		log.Printf("[MHD] Failed to serialize SD instance registration for %s: %v", uid, jsonMessage.GetError())
		return
	}

	if err := client.PublishJSONMessage(
		sharedUtils.NewEmptyOptional[string](),
		sharedUtils.NewOptionalOf(sharedConstants.SDInstanceRegistrationRequestsQueueName),
		jsonMessage.GetPayload(),
	); err != nil {
		log.Printf("[MHD] Failed to register SD instance %s: %v", uid, err)
		return
	}

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
