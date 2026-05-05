package main

import (
	"time"
)

func matchLiveRecord(store *GTFSStore, record *liveRecord, now time.Time, matchingWindow time.Duration, location *time.Location) (*tripMatch, bool) {
	candidates := store.definitionsFor(record.LineID, record.LiveRouteID)
	if len(candidates) == 0 {
		return nil, false
	}

	var best *tripMatch
	for _, candidate := range candidates {
		for _, occurrence := range candidate.Occurrences {
			if record.SourceTimestamp.Before(occurrence.ScheduledStart.Add(-matchingWindow)) ||
				record.SourceTimestamp.After(occurrence.ScheduledEnd.Add(matchingWindow)) {
				continue
			}

			score := occurrence.ScheduledStart.Sub(record.SourceTimestamp)
			if score < 0 {
				score = -score
			}
			score -= matchingBonus(candidate, occurrence, record, location)
			if score < 0 {
				score = 0
			}

			if best == nil || score < best.Score {
				best = &tripMatch{
					Definition: candidate,
					Occurrence: occurrence,
					Score:      score,
				}
			}
		}
	}

	return best, best != nil
}

func matchingBonus(candidate *tripDefinition, occurrence tripOccurrence, record *liveRecord, location *time.Location) time.Duration {
	var bonus time.Duration

	if sameServiceDate(occurrence.ServiceDate, record.SourceTimestamp, location) {
		bonus += 30 * time.Minute
	}
	if record.ServiceID != "" && record.ServiceID == occurrence.ServiceID {
		bonus += 45 * time.Minute
	}
	if record.ObservedDepartureValid {
		drift := occurrence.ScheduledStart.Sub(record.ObservedDepartureTime)
		if drift < 0 {
			drift = -drift
		}
		switch {
		case drift <= 2*time.Minute:
			bonus += 60 * time.Minute
		case drift <= 5*time.Minute:
			bonus += 40 * time.Minute
		case drift <= 10*time.Minute:
			bonus += 20 * time.Minute
		}
	}
	if record.FinalStopID != "" && record.FinalStopID == candidate.ToStopID {
		bonus += 20 * time.Minute
	}
	if record.LastStopID != "" {
		if containsStopID(candidate.StopIDs, record.LastStopID) {
			bonus += 10 * time.Minute
		}
		if record.LastStopID == candidate.FromStopID || record.LastStopID == candidate.ToStopID {
			bonus += 5 * time.Minute
		}
	}
	if activeInstanceMatches(candidate.UID, record.VehicleRuntimeID, record.SourceTimestamp) {
		bonus += 25 * time.Minute
	}

	return bonus
}

func sameServiceDate(left time.Time, right time.Time, location *time.Location) bool {
	left = left.In(location)
	right = right.In(location)
	return left.Year() == right.Year() && left.YearDay() == right.YearDay()
}

func containsStopID(stopIDs []string, target string) bool {
	for _, stopID := range stopIDs {
		if stopID == target {
			return true
		}
	}
	return false
}

func activeInstanceMatches(uid string, vehicleID string, sourceTime time.Time) bool {
	if vehicleID == "" {
		return false
	}

	instanceStatesMutex.Lock()
	defer instanceStatesMutex.Unlock()

	state, exists := instanceStates[uid]
	if !exists || !state.CurrentlyActive {
		return false
	}
	if state.LastVehicleID != vehicleID {
		return false
	}
	if sourceTime.Before(state.LastSourceTime) {
		return false
	}
	return true
}
