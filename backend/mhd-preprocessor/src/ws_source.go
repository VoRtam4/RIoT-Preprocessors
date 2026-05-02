package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/MichalBures-OG/bp-bures-RIoT-commons/src/rabbitmq"
	"github.com/gorilla/websocket"
)

var (
	unmatchedRecordsMu       sync.Mutex
	unmatchedRecordsLogged   = make(map[string]struct{})
	unmatchedRecordsLogLimit = 10
	livePayloadLogMu         sync.Mutex
	livePayloadsLogged       = 0
	livePayloadLogLimit      = 5
	errWebSocketIdle         = errors.New("websocket idle timeout")
)

const (
	wsWriteWait      = 10 * time.Second
	wsPongWait       = 75 * time.Second
	wsPingInterval   = 25 * time.Second
	wsMaxMessageSize = 1 << 20
)

func consumeAnyWebSocket(client rabbitmq.Client, store *GTFSStore, config appConfig, idleTimeout time.Duration) error {
	var lastErr error

	for _, wsURL := range config.WSURLs {
		log.Printf("[MHD] Attempting WebSocket connection: %s", wsURL)
		if err := consumeWebSocket(client, store, config, wsURL, idleTimeout); err != nil {
			log.Printf("[MHD] WebSocket endpoint failed: %s | %v", wsURL, err)
			lastErr = err
			continue
		}
	}

	if lastErr == nil {
		lastErr = errors.New("no configured websocket endpoints")
	}

	return lastErr
}

func consumeWebSocket(client rabbitmq.Client, store *GTFSStore, config appConfig, wsURL string, idleTimeout time.Duration) error {
	header := buildWebSocketHeaders(wsURL)
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		return err
	}
	defer conn.Close()

	conn.SetReadLimit(wsMaxMessageSize)
	_ = conn.SetReadDeadline(time.Now().Add(wsPongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(wsPongWait))
	})

	log.Printf("[MHD] WebSocket connected: %s", wsURL)

	done := make(chan struct{})
	defer close(done)
	go keepWebSocketAlive(conn, done)

	activity := make(chan struct{}, 1)
	idleSignal := make(chan struct{}, 1)
	go closeQuietStream(conn, done, activity, idleSignal, idleTimeout, wsURL)

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			select {
			case <-idleSignal:
				return errWebSocketIdle
			default:
			}
			return err
		}
		_ = conn.SetReadDeadline(time.Now().Add(wsPongWait))
		processWebSocketMessage(client, store, config, activity, message)
	}
}

func buildWebSocketHeaders(wsURL string) http.Header {
	header := http.Header{}

	parsed, err := url.Parse(wsURL)
	if err != nil {
		return header
	}

	switch parsed.Hostname() {
	case "gis.brno.cz":
		header.Set("Origin", "https://gis.brno.cz")
	}

	return header
}

func processWebSocketMessage(client rabbitmq.Client, store *GTFSStore, config appConfig, activity chan<- struct{}, message []byte) {
	envelope, err := parseRawEnvelope(message)
	if err != nil {
		log.Printf("[MHD] Failed to parse WebSocket payload: %v", err)
		return
	}
	select {
	case activity <- struct{}{}:
	default:
	}
	logSampleLivePayload(message)

	processLiveRecord(client, store, config, buildLiveRecord(envelope, message))
}

func processLiveRecord(client rabbitmq.Client, store *GTFSStore, config appConfig, record *liveRecord) {
	match, ok := matchLiveRecord(store, record, time.Now().UTC(), config.MatchingWindow, config.GTFSLocation)
	if !ok {
		logUnmatchedRecord(record)
		return
	}

	processMatchedRecord(client, config, match, record)
}

func closeQuietStream(conn *websocket.Conn, done <-chan struct{}, activity <-chan struct{}, idleSignal chan<- struct{}, timeout time.Duration, wsURL string) {
	if timeout < 0 {
		return
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case <-done:
			return
		case <-activity:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(timeout)
		case <-timer.C:
			log.Printf("[MHD] No live payload received for %s on %s, reconnecting", timeout, wsURL)
			select {
			case idleSignal <- struct{}{}:
			default:
			}
			_ = conn.Close()
			return
		}
	}
}

func keepWebSocketAlive(conn *websocket.Conn, done <-chan struct{}) {
	ticker := time.NewTicker(wsPingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			_ = conn.SetWriteDeadline(time.Now().Add(wsWriteWait))
			if err := conn.WriteMessage(websocket.PingMessage, []byte("mhd-keepalive")); err != nil {
				log.Printf("[MHD] WebSocket ping failed: %v", err)
				return
			}
		}
	}
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

func logSampleLivePayload(message []byte) {
	livePayloadLogMu.Lock()
	defer livePayloadLogMu.Unlock()

	if livePayloadsLogged >= livePayloadLogLimit {
		return
	}

	livePayloadsLogged++
	log.Printf("[MHD] Sample live payload %d: %s", livePayloadsLogged, string(message))
}

func runWebSocketLoop(client rabbitmq.Client, store *GTFSStore, config appConfig) {
	for {
		if err := consumeAnyWebSocket(client, store, config, config.NoDataReconnect); err != nil {
			log.Printf("[MHD] WebSocket loop ended: %v", err)
		}
		time.Sleep(config.ReconnectDelay)
	}
}

func buildLiveRecord(envelope *rawEnvelope, message []byte) *liveRecord {
	return buildLiveRecordFromPayload(envelope.Attributes, envelope.Geometry, message)
}

func buildLiveRecordFromPayload(attributes map[string]interface{}, geometry map[string]interface{}, rawMessage []byte) *liveRecord {
	record := &liveRecord{
		RawMessage: string(rawMessage),
		Attributes: attributes,
	}

	if marshaled, err := json.Marshal(map[string]interface{}{
		"attributes": attributes,
		"geometry":   geometry,
	}); err == nil {
		record.RawMessage = string(marshaled)
	}

	record.ObjectID = extractString(lookupAttribute(attributes, "objectid"))
	record.GlobalID = extractString(lookupAttribute(attributes, "globalid"))
	record.VehicleRuntimeID = extractString(lookupAttribute(attributes, "id"))
	record.VehicleType = extractString(lookupAttribute(attributes, "vtype"))
	record.LineType = extractString(lookupAttribute(attributes, "ltype"))
	record.LineID = extractString(lookupAttribute(attributes, "lineid"))
	record.LineName = extractString(lookupAttribute(attributes, "linename"))
	record.LiveRouteID = extractString(lookupAttribute(attributes, "routeid"))
	record.Course = extractString(lookupAttribute(attributes, "course"))
	record.LowFloor = extractString(lookupAttribute(attributes, "lf"))
	record.LastStopID = extractString(lookupAttribute(attributes, "laststopid"))
	record.FinalStopID = extractString(lookupAttribute(attributes, "finalstopid"))

	if timestamp, ok := extractUnixMillis(lookupAttribute(attributes, "lastupdate", "timeupdated")); ok {
		record.SourceTimestamp = timestamp
	} else {
		record.SourceTimestamp = time.Now().UTC()
	}

	if value, ok := extractFloat(lookupAttribute(attributes, "lat")); ok {
		record.GeometryLat = value
	}
	if value, ok := extractFloat(lookupAttribute(attributes, "lng")); ok {
		record.GeometryLng = value
	}
	if value, ok := extractFloat(lookupAttribute(geometry, "y")); ok {
		record.GeometryLat = value
	}
	if value, ok := extractFloat(lookupAttribute(geometry, "x")); ok {
		record.GeometryLng = value
	}
	if value, ok := extractFloat(lookupAttribute(attributes, "bearing")); ok {
		record.Bearing = value
	}
	if value, ok := extractFloat(lookupAttribute(attributes, "delay")); ok {
		record.Delay = value
	}
	if value, ok := extractBool(lookupAttribute(attributes, "isinactive")); ok {
		record.IsInactive = value
	}

	return record
}
