package main

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/MichalBures-OG/bp-bures-RIoT-commons/src/rabbitmq"
	"github.com/MichalBures-OG/bp-bures-RIoT-commons/src/sharedConstants"
	"github.com/MichalBures-OG/bp-bures-RIoT-commons/src/sharedModel"
	"github.com/MichalBures-OG/bp-bures-RIoT-commons/src/sharedUtils"
)

const (
	wazeURL         = "https://www.waze.com/row-partnerhub-api/partners/16198912488/waze-feeds/9c8b4163-e3c2-436f-86b7-3db2058ce7a1?format=1"
	fetchDelay      = 2 * time.Minute
	startupJitter   = 500 * time.Millisecond
	wazeSDTypeUID   = "WAZE_JAM_LOCATION"
	wazeSDTypeLabel = "Waze Jam Location"
	unknownTagValue = "unknown"
)

var (
	sdInstances           = sharedUtils.NewSet[sharedModel.SDInstanceInfo]()
	sdInstancesMutex      sync.Mutex
	activeDevices         = make(map[string]deviceSnapshot)
	activeDevicesMu       sync.Mutex
	preprocessorStartedAt = time.Now().UTC()
)

type wazeFeed struct {
	Jams []map[string]interface{} `json:"jams"`
}

type lineCoordinate struct {
	X float64
	Y float64
}

type segmentReference struct {
	ID        int64
	FromNode  int64
	ToNode    int64
	IsForward bool
}

type deviceSnapshot struct {
	Label string
	Tags  map[string]string
}

type deviceAggregate struct {
	UID             string
	Label           string
	EventTime       time.Time
	Tags            map[string]string
	RawJams         []map[string]interface{}
	JamCount        int
	Delay           float64
	Length          float64
	Level           float64
	Speed           float64
	SpeedKPH        float64
	PubMillisLatest int64
}

func main() {
	log.SetOutput(os.Stderr)

	client := rabbitmq.NewClient()
	defer client.Dispose()

	registerSDType(client)
	time.Sleep(5 * time.Second)
	go checkForSetOfSDInstancesUpdates(client)

	for {
		fetchAndProcessWazeData(client)
		time.Sleep(fetchDelay)
	}
}

func registerSDType(client rabbitmq.Client) {
	parameters := []sharedModel.SDParameter{
		{Denotation: "country", Label: "Country", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleTag},
		{Denotation: "city", Label: "City", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleTag},
		{Denotation: "street", Label: "Street", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleTag},
		{Denotation: "roadType", Label: "Road Type", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleTag},
		{Denotation: "segmentId", Label: "Segment ID", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleTag},
		{Denotation: "fromNode", Label: "From Node", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleTag},
		{Denotation: "toNode", Label: "To Node", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleTag},
		{Denotation: "isForward", Label: "Is Forward", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleTag},
		{Denotation: "delay", Label: "Delay", Type: sharedModel.SDParameterTypeNumber, Role: sharedModel.SDParameterRoleField},
		{Denotation: "length", Label: "Length", Type: sharedModel.SDParameterTypeNumber, Role: sharedModel.SDParameterRoleField},
		{Denotation: "level", Label: "Level", Type: sharedModel.SDParameterTypeNumber, Role: sharedModel.SDParameterRoleField},
		{Denotation: "speed", Label: "Speed", Type: sharedModel.SDParameterTypeNumber, Role: sharedModel.SDParameterRoleField},
		{Denotation: "speedKPH", Label: "Speed KPH", Type: sharedModel.SDParameterTypeNumber, Role: sharedModel.SDParameterRoleField},
		{Denotation: "jamCount", Label: "Jam Count", Type: sharedModel.SDParameterTypeNumber, Role: sharedModel.SDParameterRoleField},
		{Denotation: "pubMillisLatest", Label: "Latest Publication Time", Type: sharedModel.SDParameterTypeNumber, Role: sharedModel.SDParameterRoleField},
		{Denotation: "rawJams", Label: "Raw Jams", Type: sharedModel.SDParameterTypeString, Role: sharedModel.SDParameterRoleField},
	}

	message := sharedModel.SDTypeRegistrationRequestISCMessage{
		SDTypeUID:  wazeSDTypeUID,
		Label:      wazeSDTypeLabel,
		Parameters: parameters,
	}

	jsonResult := sharedUtils.SerializeToJSON(message)
	if jsonResult.IsFailure() {
		log.Printf("[WAZE] Failed to serialize SDType registration: %v", jsonResult.GetError())
		return
	}

	if err := client.PublishJSONMessage(
		sharedUtils.NewEmptyOptional[string](),
		sharedUtils.NewOptionalOf(sharedConstants.SDTypeRegistrationRequestsQueueName),
		jsonResult.GetPayload(),
	); err != nil {
		log.Printf("[WAZE] Failed to register SDType: %v", err)
		return
	}

	log.Printf("[WAZE] SDType registration published: %s", wazeSDTypeUID)
}

func checkForSetOfSDInstancesUpdates(client rabbitmq.Client) {
	rabbitmq.ConsumeJSONMessages[sharedModel.SDInstanceConfigurationUpdateISCMessage](
		client,
		sharedConstants.SetOfSDInstancesUpdatesQueueName,
		func(messagePayload sharedModel.SDInstanceConfigurationUpdateISCMessage) error {
			sdInstancesMutex.Lock()
			sdInstances = sharedUtils.NewSetFromSlice(messagePayload)
			sdInstancesMutex.Unlock()
			return nil
		},
	)
}

func fetchAndProcessWazeData(client rabbitmq.Client) {
	resp, err := http.Get(wazeURL)
	if err != nil {
		log.Printf("[WAZE] Failed to fetch data: %v", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[WAZE] Failed to read response: %v", err)
		return
	}

	var feed wazeFeed
	if err := json.Unmarshal(body, &feed); err != nil {
		log.Printf("[WAZE] Failed to decode feed: %v", err)
		return
	}

	currentDevices := make(map[string]deviceSnapshot)
	aggregates := make(map[string]*deviceAggregate)

	for _, jam := range feed.Jams {
		for _, aggregate := range buildAggregatesFromJam(jam) {
			existing, exists := aggregates[aggregate.UID]
			if !exists {
				aggregates[aggregate.UID] = aggregate
				continue
			}

			mergeAggregate(existing, aggregate)
		}
	}

	for _, aggregate := range aggregates {
		if determineSDInstanceScenario(aggregate.UID) == "unknown" {
			registerSDInstance(client, aggregate.UID, aggregate.Label, aggregate.EventTime)
		}

		currentDevices[aggregate.UID] = deviceSnapshot{
			Label: aggregate.Label,
			Tags:  cloneTags(aggregate.Tags),
		}

		if !wasDeviceActive(aggregate.UID) && shouldPublishStartupZeroDelay(aggregate.EventTime) {
			publishState(client, aggregate.UID, buildStartupZeroDelayEventTime(aggregate.UID, aggregate.EventTime), aggregate.Label, buildZeroDelayParams(aggregate.Tags))
		}

		publishState(client, aggregate.UID, aggregate.EventTime, aggregate.Label, buildActiveParams(aggregate))
	}

	now := time.Now().UTC()

	activeDevicesMu.Lock()
	previousDevices := activeDevices
	activeDevices = currentDevices
	activeDevicesMu.Unlock()

	for uid, snapshot := range previousDevices {
		if _, stillActive := currentDevices[uid]; stillActive {
			continue
		}

		publishState(client, uid, now, snapshot.Label, buildZeroDelayParams(snapshot.Tags))
	}
}

func buildAggregatesFromJam(jam map[string]interface{}) []*deviceAggregate {
	segments := parseSegments(jam["segments"])
	if len(segments) == 0 {
		return nil
	}

	country := normalizeDisplayTag(jam["country"])
	city := normalizeDisplayTag(jam["city"])
	street := normalizeDisplayTag(jam["street"])
	roadType := normalizeDisplayTag(jam["roadType"])
	eventTime, pubMillis := extractEventTime(jam)
	rawJam := reduceRawJam(jam)
	aggregates := make([]*deviceAggregate, 0, len(segments))

	for _, segment := range segments {
		directionLabel := buildDirectionLabel(segment.IsForward)
		segmentID := strconv.FormatInt(segment.ID, 10)

		tags := map[string]string{
			"country":   country,
			"city":      city,
			"street":    street,
			"roadType":  roadType,
			"segmentId": segmentID,
			"fromNode":  safeInt64String(segment.FromNode),
			"toNode":    safeInt64String(segment.ToNode),
			"isForward": strconv.FormatBool(segment.IsForward),
		}

		aggregates = append(aggregates, &deviceAggregate{
			UID:             buildInstanceUID(segment.ID, segment.IsForward),
			Label:           buildInstanceLabel(city, street, segmentID, directionLabel),
			EventTime:       eventTime,
			Tags:            tags,
			RawJams:         []map[string]interface{}{rawJam},
			JamCount:        1,
			Delay:           normalizeAggregatedDelay(extractFloat(jam["delay"])),
			Length:          extractFloat(jam["length"]),
			Level:           extractFloat(jam["level"]),
			Speed:           extractFloat(jam["speed"]),
			SpeedKPH:        extractFloat(jam["speedKMH"]),
			PubMillisLatest: pubMillis,
		})
	}

	return aggregates
}

func mergeAggregate(target *deviceAggregate, addition *deviceAggregate) {
	target.RawJams = append(target.RawJams, addition.RawJams...)
	target.JamCount += addition.JamCount

	if isGreaterDelay(addition.Delay, target.Delay) {
		target.Delay = addition.Delay
	}
	target.Length += addition.Length
	if addition.Level > target.Level {
		target.Level = addition.Level
	}
	if target.Speed == 0 || (addition.Speed > 0 && addition.Speed < target.Speed) {
		target.Speed = addition.Speed
	}
	if target.SpeedKPH == 0 || (addition.SpeedKPH > 0 && addition.SpeedKPH < target.SpeedKPH) {
		target.SpeedKPH = addition.SpeedKPH
	}
	if addition.PubMillisLatest > target.PubMillisLatest {
		target.PubMillisLatest = addition.PubMillisLatest
		target.EventTime = addition.EventTime
	}

	sortRawJams(target.RawJams)
}

func buildActiveParams(aggregate *deviceAggregate) map[string]interface{} {
	params := tagsToParams(aggregate.Tags)
	params["delay"] = aggregate.Delay
	params["length"] = aggregate.Length
	params["level"] = aggregate.Level
	params["speed"] = aggregate.Speed
	params["speedKPH"] = aggregate.SpeedKPH
	params["jamCount"] = float64(aggregate.JamCount)
	params["pubMillisLatest"] = float64(aggregate.PubMillisLatest)

	sortRawJams(aggregate.RawJams)
	rawJamsJSON, err := json.Marshal(aggregate.RawJams)
	if err == nil {
		params["rawJams"] = string(rawJamsJSON)
	}

	return params
}

func buildZeroDelayParams(tags map[string]string) map[string]interface{} {
	params := tagsToParams(tags)
	params["delay"] = float64(0)
	return params
}

func buildStartupZeroDelayEventTime(uid string, activeEventTime time.Time) time.Time {
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(uid))

	maxJitterMillis := int64(startupJitter / time.Millisecond)
	if maxJitterMillis <= 0 {
		return activeEventTime.Add(-time.Millisecond)
	}

	// Keep the synthetic startup snapshot strictly before the real active event.
	offsetMillis := int64(hasher.Sum32()%uint32(maxJitterMillis)) + 1
	return activeEventTime.Add(-time.Duration(offsetMillis) * time.Millisecond)
}

func shouldPublishStartupZeroDelay(eventTime time.Time) bool {
	return !eventTime.Before(preprocessorStartedAt)
}

func publishState(client rabbitmq.Client, uid string, eventTime time.Time, label string, params map[string]interface{}) {
	message := sharedModel.KPIFulfillmentCheckRequestISCMessage{
		EventTime:     eventTime.UTC(),
		SDInstanceUID: uid,
		SDTypeUID:     wazeSDTypeUID,
		Parameters:    params,
	}

	jsonResult := sharedUtils.SerializeToJSON(message)
	if jsonResult.IsFailure() {
		log.Printf("[WAZE] Failed to serialize state for %s: %v", uid, jsonResult.GetError())
		return
	}

	if err := client.PublishJSONMessage(
		sharedUtils.NewEmptyOptional[string](),
		sharedUtils.NewOptionalOf(sharedConstants.KPIFulfillmentCheckRequestsQueueName),
		jsonResult.GetPayload(),
	); err != nil {
		log.Printf("[WAZE] Failed to publish state for %s: %v", uid, err)
		return
	}

	log.Printf("[WAZE] Published state | uid=%s label=%s delay=%v", uid, label, params["delay"])
}

func registerSDInstance(client rabbitmq.Client, uid string, label string, eventTime time.Time) {
	message := sharedModel.SDInstanceRegistrationRequestISCMessage{
		EventTime:     eventTime.UTC(),
		Label:         label,
		SDInstanceUID: uid,
		SDTypeUID:     wazeSDTypeUID,
	}

	jsonResult := sharedUtils.SerializeToJSON(message)
	if jsonResult.IsFailure() {
		log.Printf("[WAZE] Failed to serialize SDInstance registration for %s: %v", uid, jsonResult.GetError())
		return
	}

	if err := client.PublishJSONMessage(
		sharedUtils.NewEmptyOptional[string](),
		sharedUtils.NewOptionalOf(sharedConstants.SDInstanceRegistrationRequestsQueueName),
		jsonResult.GetPayload(),
	); err != nil {
		log.Printf("[WAZE] Failed to register instance %s: %v", uid, err)
		return
	}

	sdInstancesMutex.Lock()
	sdInstances.Add(sharedModel.SDInstanceInfo{
		SDInstanceUID:   uid,
		ConfirmedByUser: false,
	})
	sdInstancesMutex.Unlock()

	log.Printf("[WAZE] Registered instance | uid=%s label=%s", uid, label)
}

func determineSDInstanceScenario(uid string) string {
	sdInstancesMutex.Lock()
	defer sdInstancesMutex.Unlock()

	if sdInstances.Contains(sharedModel.SDInstanceInfo{SDInstanceUID: uid, ConfirmedByUser: true}) {
		return "confirmed"
	}
	if sdInstances.Contains(sharedModel.SDInstanceInfo{SDInstanceUID: uid, ConfirmedByUser: false}) {
		return "notYetConfirmed"
	}
	return "unknown"
}

func wasDeviceActive(uid string) bool {
	activeDevicesMu.Lock()
	defer activeDevicesMu.Unlock()

	_, exists := activeDevices[uid]
	return exists
}

func tagsToParams(tags map[string]string) map[string]interface{} {
	params := make(map[string]interface{}, len(tags)+1)
	keys := make([]string, 0, len(tags))
	for key := range tags {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		params[key] = tags[key]
	}
	return params
}

func cloneTags(src map[string]string) map[string]string {
	dst := make(map[string]string, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

func normalizeDisplayTag(value interface{}) string {
	raw := strings.TrimSpace(fmt.Sprintf("%v", value))
	raw = sanitizeText(raw)
	if raw == "" || raw == "<nil>" {
		return unknownTagValue
	}
	return raw
}

func sanitizeText(raw string) string {
	if !utf8.ValidString(raw) {
		raw = strings.ToValidUTF8(raw, "")
	}

	var builder strings.Builder
	for len(raw) > 0 {
		r, size := utf8.DecodeRuneInString(raw)
		raw = raw[size:]
		if r == utf8.RuneError && size == 1 {
			continue
		}
		if unicode.IsControl(r) && !unicode.IsSpace(r) {
			continue
		}
		builder.WriteRune(r)
	}

	return strings.TrimSpace(builder.String())
}

func buildInstanceUID(segmentID int64, isForward bool) string {
	suffix := "REV"
	if isForward {
		suffix = "FWD"
	}
	return fmt.Sprintf("WAZE_JAM_%d_%s", segmentID, suffix)
}

func buildInstanceLabel(city string, street string, segmentID string, directionLabel string) string {
	return fmt.Sprintf("%s, %s: %s %s",
		friendlyLabel(city),
		friendlyLabel(street),
		friendlyLabel(segmentID),
		friendlyLabel(directionLabel),
	)
}

func friendlyLabel(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || value == unknownTagValue {
		return "Unknown"
	}
	return value
}

func parseLine(value interface{}) []lineCoordinate {
	rawLine, ok := value.([]interface{})
	if !ok {
		return nil
	}

	result := make([]lineCoordinate, 0, len(rawLine))
	for _, point := range rawLine {
		pointMap, ok := point.(map[string]interface{})
		if !ok {
			continue
		}

		x, okX := extractMaybeFloat(pointMap["x"])
		y, okY := extractMaybeFloat(pointMap["y"])
		if !okX || !okY {
			continue
		}

		result = append(result, lineCoordinate{X: x, Y: y})
	}

	return result
}

func parseSegments(value interface{}) []segmentReference {
	rawSegments, ok := value.([]interface{})
	if !ok {
		return nil
	}

	result := make([]segmentReference, 0, len(rawSegments))
	for _, item := range rawSegments {
		segmentMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		id, okID := extractMaybeInt64(segmentMap["ID"])
		fromNode, okFrom := extractMaybeInt64(segmentMap["fromNode"])
		toNode, okTo := extractMaybeInt64(segmentMap["toNode"])
		isForward, okForward := extractMaybeBool(segmentMap["isForward"])
		if !okID || !okFrom || !okTo || !okForward {
			continue
		}

		result = append(result, segmentReference{
			ID:        id,
			FromNode:  fromNode,
			ToNode:    toNode,
			IsForward: isForward,
		})
	}

	sort.SliceStable(result, func(i, j int) bool {
		if result[i].ID != result[j].ID {
			return result[i].ID < result[j].ID
		}
		if result[i].IsForward != result[j].IsForward {
			return !result[i].IsForward && result[j].IsForward
		}
		if result[i].FromNode != result[j].FromNode {
			return result[i].FromNode < result[j].FromNode
		}
		return result[i].ToNode < result[j].ToNode
	})

	return result
}

func buildDirectionLabel(isForward bool) string {
	if isForward {
		return "Forward"
	}
	return "Reverse"
}

func normalizeAggregatedDelay(delay float64) float64 {
	return delay
}

func isGreaterDelay(left float64, right float64) bool {
	if left == -1 {
		return right != -1
	}
	if right == -1 {
		return false
	}
	return left > right
}

func reduceRawJam(jam map[string]interface{}) map[string]interface{} {
	reduced := make(map[string]interface{})
	for key, value := range jam {
		switch key {
		case "country", "city", "street", "roadType":
			continue
		default:
			reduced[key] = value
		}
	}
	return reduced
}

func extractEventTime(jam map[string]interface{}) (time.Time, int64) {
	pubMillis, ok := extractMaybeInt64(jam["pubMillis"])
	if !ok || pubMillis <= 0 {
		now := time.Now().UTC()
		return now, now.UnixMilli()
	}

	return time.UnixMilli(pubMillis).UTC(), pubMillis
}

func extractFloat(value interface{}) float64 {
	result, ok := extractMaybeFloat(value)
	if !ok {
		return 0
	}
	return result
}

func extractMaybeFloat(value interface{}) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case json.Number:
		parsed, err := typed.Float64()
		if err != nil {
			return 0, false
		}
		return parsed, true
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func extractMaybeInt64(value interface{}) (int64, bool) {
	switch typed := value.(type) {
	case int64:
		return typed, true
	case int:
		return int64(typed), true
	case float64:
		return int64(typed), true
	case json.Number:
		parsed, err := typed.Int64()
		if err != nil {
			return 0, false
		}
		return parsed, true
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func extractMaybeBool(value interface{}) (bool, bool) {
	switch typed := value.(type) {
	case bool:
		return typed, true
	case string:
		parsed, err := strconv.ParseBool(strings.TrimSpace(typed))
		if err != nil {
			return false, false
		}
		return parsed, true
	default:
		return false, false
	}
}

func safeInt64String(value int64) string {
	if value == 0 {
		return unknownTagValue
	}
	return strconv.FormatInt(value, 10)
}

func sortRawJams(items []map[string]interface{}) {
	sort.SliceStable(items, func(i, j int) bool {
		return rawJamSortKey(items[i]) < rawJamSortKey(items[j])
	})
}

func rawJamSortKey(item map[string]interface{}) string {
	pubMillis, _ := extractMaybeInt64(item["pubMillis"])
	uuid := strings.TrimSpace(fmt.Sprintf("%v", item["uuid"]))
	blocking := strings.TrimSpace(fmt.Sprintf("%v", item["blockingAlertUuid"]))
	return fmt.Sprintf("%020d|%s|%s", pubMillis, uuid, blocking)
}
