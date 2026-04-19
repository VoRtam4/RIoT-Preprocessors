package main

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/MichalBures-OG/bp-bures-RIoT-commons/src/rabbitmq"
	"github.com/gorilla/websocket"
)

var (
	unmatchedRecordsMu       sync.Mutex
	unmatchedRecordsLogged   = make(map[string]struct{})
	unmatchedRecordsLogLimit = 10
)

func runWebSocketLoop(client rabbitmq.Client, store *GTFSStore, config appConfig) {
	for {
		if err := consumeWebSocket(client, store, config); err != nil {
			log.Printf("[MHD] WebSocket loop failed: %v", err)
		}
		time.Sleep(config.ReconnectDelay)
	}
}

func consumeWebSocket(client rabbitmq.Client, store *GTFSStore, config appConfig) error {
	conn, _, err := websocket.DefaultDialer.Dial(config.WSURL, nil)
	if err != nil {
		return err
	}
	defer conn.Close()

	log.Printf("[MHD] WebSocket connected: %s", config.WSURL)
	if err := configureStreamFilter(conn); err != nil {
		return err
	}

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		processWebSocketMessage(client, store, config, message)
	}
}

func processWebSocketMessage(client rabbitmq.Client, store *GTFSStore, config appConfig, message []byte) {
	envelope, err := parseRawEnvelope(message)
	if err != nil {
		log.Printf("[MHD] Failed to parse WebSocket payload: %v", err)
		return
	}
	if isStreamControlMessage(envelope) {
		log.Printf("[MHD] Stream filter acknowledged")
		return
	}

	record := buildLiveRecord(envelope, message)
	match, ok := matchLiveRecord(store, record, time.Now().UTC(), config.MatchingWindow, config.GTFSLocation)
	if !ok {
		logUnmatchedRecord(record)
		return
	}

	processMatchedRecord(client, config, match, record)
}

func configureStreamFilter(conn *websocket.Conn) error {
	if err := conn.WriteJSON(map[string]interface{}{"filter": nil}); err != nil {
		return err
	}
	return nil
}

func isStreamControlMessage(envelope *rawEnvelope) bool {
	return envelope != nil && envelope.Attributes == nil && envelope.Filter != nil
}

func logUnmatchedRecord(record *liveRecord) {
	key := record.LineID + "/" + record.LiveRouteID
	if key == "/" {
		key = "missing-line-route"
	}

	unmatchedRecordsMu.Lock()
	defer unmatchedRecordsMu.Unlock()

	if len(unmatchedRecordsLogged) >= unmatchedRecordsLogLimit {
		return
	}
	if _, exists := unmatchedRecordsLogged[key]; exists {
		return
	}

	unmatchedRecordsLogged[key] = struct{}{}
	log.Printf("[MHD] Unmatched live record | lineid=%s routeid=%s vehicle=%s finalstopid=%s laststopid=%s lastupdate=%s",
		record.LineID,
		record.LiveRouteID,
		record.VehicleRuntimeID,
		record.FinalStopID,
		record.LastStopID,
		record.SourceTimestamp.Format(time.RFC3339Nano),
	)
}

func buildLiveRecord(envelope *rawEnvelope, message []byte) *liveRecord {
	record := &liveRecord{
		RawMessage: string(message),
		Attributes: envelope.Attributes,
	}

	if marshaled, err := json.Marshal(envelope); err == nil {
		record.RawMessage = string(marshaled)
	}

	record.GlobalID = extractString(lookupAttribute(envelope.Attributes, "globalid"))
	record.VehicleRuntimeID = extractString(lookupAttribute(envelope.Attributes, "id"))
	record.VehicleType = extractString(lookupAttribute(envelope.Attributes, "vtype"))
	record.LineType = extractString(lookupAttribute(envelope.Attributes, "ltype"))
	record.LineID = extractString(lookupAttribute(envelope.Attributes, "lineid"))
	record.LineName = extractString(lookupAttribute(envelope.Attributes, "linename"))
	record.LiveRouteID = extractString(lookupAttribute(envelope.Attributes, "routeid"))
	record.Course = extractString(lookupAttribute(envelope.Attributes, "course"))
	record.LowFloor = extractString(lookupAttribute(envelope.Attributes, "lf"))
	record.LastStopID = extractString(lookupAttribute(envelope.Attributes, "laststopid"))
	record.FinalStopID = extractString(lookupAttribute(envelope.Attributes, "finalstopid"))

	if timestamp, ok := extractUnixMillis(lookupAttribute(envelope.Attributes, "lastupdate", "timeupdated")); ok {
		record.SourceTimestamp = timestamp
	} else {
		record.SourceTimestamp = time.Now().UTC()
	}

	if value, ok := extractFloat(lookupAttribute(envelope.Attributes, "lat")); ok {
		record.GeometryLat = value
	}
	if value, ok := extractFloat(lookupAttribute(envelope.Attributes, "lng")); ok {
		record.GeometryLng = value
	}
	if value, ok := extractFloat(lookupAttribute(envelope.Attributes, "bearing")); ok {
		record.Bearing = value
	}
	if value, ok := extractFloat(lookupAttribute(envelope.Attributes, "delay")); ok {
		record.Delay = value
	}
	if value, ok := extractBool(lookupAttribute(envelope.Attributes, "isinactive")); ok {
		record.IsInactive = value
	}

	return record
}
