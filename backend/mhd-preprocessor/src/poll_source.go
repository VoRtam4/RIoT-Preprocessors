package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/MichalBures-OG/bp-bures-RIoT-commons/src/rabbitmq"
)

type pollingFeatureResponse struct {
	Features []pollingFeature `json:"features"`
	Error    any              `json:"error"`
}

type pollingFeature struct {
	Attributes map[string]interface{} `json:"attributes"`
	Geometry   map[string]interface{} `json:"geometry"`
}

type pollingCursor struct {
	mu                    sync.Mutex
	anchorTime            time.Time
	lastSeenTimeByVehicle map[string]time.Time
}

var pollCursor = pollingCursor{
	lastSeenTimeByVehicle: map[string]time.Time{},
}

func runPrimarySourceLoop(client rabbitmq.Client, store *GTFSStore, config appConfig) {
	setSourceMode("wss")
	err := consumeAnyWebSocket(client, store, config, config.WSNoDataFallback)
	log.Printf("[MHD] Switching to polling fallback: %v", err)
	setSourceMode("poll")
	runPollingFallbackLoop(client, store, config)
}

func runPollingFallbackLoop(client rabbitmq.Client, store *GTFSStore, config appConfig) {
	pollDelay := config.PollingInterval
	if pollDelay <= 0 {
		pollDelay = time.Second
	}

	wsRetryDelay := config.WSPollingRetryDelay
	if wsRetryDelay <= 0 {
		wsRetryDelay = pollDelay
	}

	nextWebSocketAttempt := time.Now().UTC().Add(wsRetryDelay)

	for {
		if err := pollOnce(client, store, config); err != nil {
			log.Printf("[MHD] Polling request failed: %v", err)
		}

		now := time.Now().UTC()
		if !now.Before(nextWebSocketAttempt) {
			log.Printf("[MHD] Probing WebSocket recovery")
			err := consumeAnyWebSocket(client, store, config, config.WSPollingProbeTimeout)
			log.Printf("[MHD] WebSocket probe ended, continuing polling fallback: %v", err)
			setSourceMode("poll")
			nextWebSocketAttempt = time.Now().UTC().Add(wsRetryDelay)
		}

		time.Sleep(pollDelay)
	}
}

func pollOnce(client rabbitmq.Client, store *GTFSStore, config appConfig) error {
	anchor, err := ensurePollingAnchor(config)
	if err != nil {
		return err
	}

	records, observedMaxTime, freshnessCutoff, err := fetchRecentPollingRecords(config, anchor)
	if err != nil {
		return err
	}
	if !observedMaxTime.IsZero() && observedMaxTime.Before(freshnessCutoff) {
		log.Printf("[MHD] Polling data is stale: latest=%s cutoff=%s", observedMaxTime.Format(time.RFC3339), freshnessCutoff.Format(time.RFC3339))
	}

	latestByVehicle := reduceLatestByVehicle(records)
	ordered := make([]*liveRecord, 0, len(latestByVehicle))
	for _, record := range latestByVehicle {
		ordered = append(ordered, record)
	}

	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].SourceTimestamp.Equal(ordered[j].SourceTimestamp) {
			return pollingVehicleKey(ordered[i]) < pollingVehicleKey(ordered[j])
		}
		return ordered[i].SourceTimestamp.Before(ordered[j].SourceTimestamp)
	})

	processed := 0
	for _, record := range ordered {
		if !shouldProcessPolledVehicleRecord(record) {
			continue
		}
		processed++
		processLiveRecord(client, store, config, record)
	}

	updatePollingAnchor(observedMaxTime)
	log.Printf("[MHD] Polling fetched %d fresh rows, reduced to %d vehicle snapshots, processed %d updates", len(records), len(ordered), processed)
	return nil
}

func ensurePollingAnchor(config appConfig) (time.Time, error) {
	pollCursor.mu.Lock()
	current := pollCursor.anchorTime
	pollCursor.mu.Unlock()

	if !current.IsZero() {
		return current, nil
	}

	latest, err := fetchLatestPollingTimestamp(config)
	if err != nil {
		return time.Time{}, err
	}

	updatePollingAnchor(latest)
	return latest, nil
}

func updatePollingAnchor(candidate time.Time) {
	if candidate.IsZero() {
		return
	}

	pollCursor.mu.Lock()
	if candidate.After(pollCursor.anchorTime) {
		pollCursor.anchorTime = candidate
	}
	pollCursor.mu.Unlock()
}

func fetchLatestPollingTimestamp(config appConfig) (time.Time, error) {
	requestURL, err := buildLatestPollingTimestampURL(config)
	if err != nil {
		return time.Time{}, err
	}

	responseBody, err := fetchPollingBody(config, requestURL)
	if err != nil {
		return time.Time{}, err
	}

	var payload struct {
		Features []struct {
			Attributes map[string]interface{} `json:"attributes"`
		} `json:"features"`
		Error any `json:"error"`
	}

	if err := json.Unmarshal(responseBody, &payload); err != nil {
		return time.Time{}, err
	}
	if payload.Error != nil {
		return time.Time{}, fmt.Errorf("polling max timestamp query returned ArcGIS error: %v", payload.Error)
	}
	if len(payload.Features) == 0 {
		return time.Time{}, fmt.Errorf("polling max timestamp query returned no features")
	}

	value := lookupAttribute(payload.Features[0].Attributes, "maxTimeUpdated")
	latest, ok := extractUnixMillis(value)
	if !ok {
		return time.Time{}, fmt.Errorf("polling max timestamp query returned invalid timestamp: %v", value)
	}

	return latest, nil
}

func fetchRecentPollingRecords(config appConfig, anchor time.Time) ([]*liveRecord, time.Time, time.Time, error) {
	freshnessCutoff := time.Now().UTC().Add(-2 * config.PollingInterval)
	requestURL, err := buildPollingURL(config, anchor, freshnessCutoff)
	if err != nil {
		return nil, time.Time{}, time.Time{}, err
	}

	responseBody, err := fetchPollingBody(config, requestURL)
	if err != nil {
		return nil, time.Time{}, time.Time{}, err
	}

	var payload pollingFeatureResponse
	if err := json.Unmarshal(responseBody, &payload); err != nil {
		return nil, time.Time{}, time.Time{}, err
	}

	if payload.Error != nil {
		return nil, time.Time{}, time.Time{}, fmt.Errorf("polling endpoint returned ArcGIS error: %v", payload.Error)
	}

	records := make([]*liveRecord, 0, len(payload.Features))
	var observedMax time.Time
	for _, feature := range payload.Features {
		rawMessage, _ := json.Marshal(feature)
		record := buildLiveRecordFromPayload(feature.Attributes, feature.Geometry, rawMessage)
		if record.SourceTimestamp.After(observedMax) {
			observedMax = record.SourceTimestamp
		}
		if record.SourceTimestamp.Before(freshnessCutoff) {
			continue
		}
		records = append(records, record)
	}

	return records, observedMax, freshnessCutoff, nil
}

func fetchPollingBody(config appConfig, requestURL string) ([]byte, error) {
	client := &http.Client{Timeout: config.PollingHTTPTimeout}
	response, err := client.Get(requestURL)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return nil, fmt.Errorf("polling endpoint returned %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}

	return io.ReadAll(response.Body)
}

func buildLatestPollingTimestampURL(config appConfig) (string, error) {
	parsed, err := url.Parse(config.PollingURL)
	if err != nil {
		return "", err
	}

	query := parsed.Query()
	query.Set("f", "json")
	query.Set("where", "1=1")
	query.Set("outStatistics", `[{"statisticType":"max","onStatisticField":"TimeUpdated","outStatisticFieldName":"maxTimeUpdated"}]`)
	query.Set("returnGeometry", "false")
	parsed.RawQuery = query.Encode()

	return parsed.String(), nil
}

func buildPollingURL(config appConfig, anchor time.Time, freshnessCutoff time.Time) (string, error) {
	parsed, err := url.Parse(config.PollingURL)
	if err != nil {
		return "", err
	}

	windowStart := anchor.Add(-2 * config.PollingInterval)
	if windowStart.IsZero() {
		windowStart = anchor
	}
	if windowStart.Before(freshnessCutoff) {
		windowStart = freshnessCutoff
	}

	query := parsed.Query()
	query.Set("f", "json")
	query.Set("where", fmt.Sprintf("TimeUpdated >= %d", windowStart.UnixMilli()))
	query.Set("outFields", "ID,IDB,IDC,VType,LType,Lat,Lng,Bearing,LineID,LineName,RouteID,Course,LF,Delay,LastStopID,FinalStopID,IsInactive,TimeUpdated,objectid,globalid")
	query.Set("returnGeometry", "true")
	query.Set("outSR", "4326")
	query.Set("orderByFields", "TimeUpdated DESC")
	query.Set("resultRecordCount", strconv.Itoa(config.PollingResultLimit))
	parsed.RawQuery = query.Encode()

	return parsed.String(), nil
}

func reduceLatestByVehicle(records []*liveRecord) map[string]*liveRecord {
	result := make(map[string]*liveRecord, len(records))

	for _, record := range records {
		key := pollingVehicleKey(record)
		if key == "" {
			continue
		}

		current, exists := result[key]
		if !exists || record.SourceTimestamp.After(current.SourceTimestamp) {
			result[key] = record
		}
	}

	return result
}

func pollingVehicleKey(record *liveRecord) string {
	return strings.TrimSpace(firstNonEmpty(record.VehicleRuntimeID, record.GlobalID, record.ObjectID))
}

func shouldProcessPolledVehicleRecord(record *liveRecord) bool {
	key := pollingVehicleKey(record)
	if key == "" {
		return false
	}

	pollCursor.mu.Lock()
	defer pollCursor.mu.Unlock()

	lastSeen, exists := pollCursor.lastSeenTimeByVehicle[key]
	if exists && !record.SourceTimestamp.After(lastSeen) {
		return false
	}

	pollCursor.lastSeenTimeByVehicle[key] = record.SourceTimestamp
	return true
}
