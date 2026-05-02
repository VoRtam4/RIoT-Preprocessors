package main

import (
	"log"
	"sync"
	"time"

	"github.com/MichalBures-OG/bp-bures-RIoT-commons/src/rabbitmq"
	"github.com/MichalBures-OG/bp-bures-RIoT-commons/src/sharedConstants"
	"github.com/MichalBures-OG/bp-bures-RIoT-commons/src/sharedModel"
	"github.com/MichalBures-OG/bp-bures-RIoT-commons/src/sharedUtils"
)

var (
	sdInstances           = sharedUtils.NewSet[sharedModel.SDInstanceInfo]()
	sdInstancesMutex      sync.Mutex
	instanceStates        = make(map[string]*runtimeInstanceState)
	instanceStatesMutex   sync.Mutex
	preprocessorStartedAt = time.Now().UTC()
)

func checkForSetOfSDInstancesUpdates(client rabbitmq.Client) {
	err := rabbitmq.ConsumeJSONMessages[sharedModel.SDInstanceConfigurationUpdateISCMessage](
		client, sharedConstants.SetOfSDInstancesUpdatesQueueName,
		func(messagePayload sharedModel.SDInstanceConfigurationUpdateISCMessage) error {
			updatedSDInstances := sharedUtils.NewSetFromSlice(messagePayload)
			sdInstancesMutex.Lock()
			sdInstances = updatedSDInstances
			sdInstancesMutex.Unlock()
			return nil
		})
	if err != nil {
		log.Printf("[MHD] Failed to consume messages from '%s'", sharedConstants.SetOfSDInstancesUpdatesQueueName)
	}
}

func shouldRegisterInstance(uid string) bool {
	sdInstancesMutex.Lock()
	defer sdInstancesMutex.Unlock()

	if sdInstances.Contains(sharedModel.SDInstanceInfo{SDInstanceUID: uid, ConfirmedByUser: true}) {
		return false
	}
	if sdInstances.Contains(sharedModel.SDInstanceInfo{SDInstanceUID: uid, ConfirmedByUser: false}) {
		return false
	}
	return true
}

func markInstanceRegistered(uid string) {
	sdInstancesMutex.Lock()
	sdInstances.Add(sharedModel.SDInstanceInfo{
		SDInstanceUID:   uid,
		ConfirmedByUser: false,
	})
	sdInstancesMutex.Unlock()
}

func processMatchedRecord(client rabbitmq.Client, config appConfig, match *tripMatch, record *liveRecord) {
	segment, _ := buildSegmentMatch(match.Definition, record)
	tags := buildTripTags(match.Definition, match.Occurrence, record, segment)

	registerInstanceIfNeeded(client, match.Definition.UID, match.Definition.Label)

	instanceStatesMutex.Lock()
	state, exists := instanceStates[match.Definition.UID]
	if !exists {
		state = &runtimeInstanceState{
			UID:   match.Definition.UID,
			Label: match.Definition.Label,
			Tags:  cloneTags(tags),
		}
		instanceStates[match.Definition.UID] = state
	}
	wasActive := state.CurrentlyActive
	needsSyntheticStart := !state.SeenSinceStart &&
		time.Since(preprocessorStartedAt) > config.StartupGracePeriod
	now := time.Now().UTC()
	shouldPublishActive := !wasActive || state.LastActivePublishAt.IsZero() || now.Sub(state.LastActivePublishAt) >= config.ActivePublishInterval
	state.Tags = cloneTags(tags)
	state.SeenSinceStart = true
	state.CurrentlyActive = true
	state.LastSourceTime = record.SourceTimestamp
	state.LastOccurrenceAt = match.Occurrence.ScheduledStart
	state.LastVehicleID = record.VehicleRuntimeID
	state.CloseAt = match.Occurrence.ScheduledEnd.Add(config.TripEndReserve)
	if shouldPublishActive {
		state.LastActivePublishAt = now
	}
	instanceStatesMutex.Unlock()

	if needsSyntheticStart {
		publishStates(client, []sharedModel.KPIFulfillmentCheckRequestISCMessage{
			buildStateMessage(match.Definition.UID, jitterTime(preprocessorStartedAt, config.SyntheticJitter), buildInactiveParams(tags)),
		})
	}

	if shouldPublishActive {
		publishStates(client, []sharedModel.KPIFulfillmentCheckRequestISCMessage{
			buildStateMessage(match.Definition.UID, record.SourceTimestamp, buildActiveParams(tags, record, segment)),
		})
	}
}

func closeExpiredInstances(client rabbitmq.Client, config appConfig, now time.Time) {
	type closingPayload struct {
		UID  string
		Tags map[string]string
		Time time.Time
	}

	toClose := make([]closingPayload, 0)

	instanceStatesMutex.Lock()
	for _, state := range instanceStates {
		if !state.CurrentlyActive {
			continue
		}
		shouldCloseByTripEnd := !state.CloseAt.IsZero() && !state.CloseAt.After(now)
		if !shouldCloseByTripEnd {
			continue
		}

		state.CurrentlyActive = false
		state.LastActivePublishAt = time.Time{}
		toClose = append(toClose, closingPayload{
			UID:  state.UID,
			Tags: cloneTags(state.Tags),
			Time: jitterTime(state.CloseAt, config.SyntheticJitter),
		})
	}
	instanceStatesMutex.Unlock()

	for _, item := range toClose {
		publishStates(client, []sharedModel.KPIFulfillmentCheckRequestISCMessage{
			buildStateMessage(item.UID, item.Time, buildInactiveParams(item.Tags)),
		})
	}
}
