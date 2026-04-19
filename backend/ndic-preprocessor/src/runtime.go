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
		log.Printf("[NDIC] Failed to consume messages from '%s'", sharedConstants.SetOfSDInstancesUpdatesQueueName)
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

func processFetchResult(client rabbitmq.Client, config appConfig, fetch *parsedFetch) {
	currentUIDs := make(map[string]struct{}, len(fetch.Snapshots))

	for sourceID, snapshot := range fetch.Snapshots {
		uid := ndicInstanceUID(sourceID)
		currentUIDs[uid] = struct{}{}
		tags := buildTags(snapshot)

		registerInstanceIfNeeded(client, uid, uid)

		instanceStatesMutex.Lock()
		state, exists := instanceStates[uid]
		if !exists {
			state = &runtimeInstanceState{
				UID:   uid,
				Label: uid,
				Tags:  cloneTags(tags),
			}
			instanceStates[uid] = state
		}
		state.Tags = cloneTags(tags)
		needsSyntheticStart := !state.SeenSinceStart &&
			fetch.PublicationTime.After(preprocessorStartedAt) &&
			time.Since(preprocessorStartedAt) > config.StartupGracePeriod
		state.SeenSinceStart = true
		state.CurrentlyActive = true
		state.LastEventTime = fetch.PublicationTime
		instanceStatesMutex.Unlock()

		if needsSyntheticStart {
			publishState(client, uid, jitterTime(preprocessorStartedAt, config.SyntheticJitter), buildInactiveParams(tags))
		}

		publishState(client, uid, fetch.PublicationTime, buildActiveParams(tags, snapshot, fetch.PublicationTime))
	}

	closeMissingInstances(client, config, fetch.PublicationTime, currentUIDs)
}

func closeMissingInstances(client rabbitmq.Client, config appConfig, eventTime time.Time, currentUIDs map[string]struct{}) {
	type closingPayload struct {
		UID  string
		Tags map[string]string
		Time time.Time
	}

	toClose := make([]closingPayload, 0)

	instanceStatesMutex.Lock()
	for uid, state := range instanceStates {
		if !state.CurrentlyActive {
			continue
		}
		if _, exists := currentUIDs[uid]; exists {
			continue
		}

		state.CurrentlyActive = false
		state.LastEventTime = eventTime
		toClose = append(toClose, closingPayload{
			UID:  uid,
			Tags: cloneTags(state.Tags),
			Time: jitterTime(eventTime, config.SyntheticJitter),
		})
	}
	instanceStatesMutex.Unlock()

	for _, item := range toClose {
		publishState(client, item.UID, item.Time, buildInactiveParams(item.Tags))
	}
}

func ndicInstanceUID(sourceID string) string {
	return ndicSDTypeUID + "_" + sourceID
}
