package model

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/Team254/cheesy-arena/game"
	"go.etcd.io/bbolt"
)

const fillerQueueGeneratedMarker = "Filler queue auto-generated"
const fillerQueueMatchSpacing = 15 * time.Minute

var fillerQueueStations = [...]string{"R1", "R2", "R3", "B1", "B2", "B3"}

// IsFillerQueueGenerated identifies matches created by kiosk demand.
func (match *Match) IsFillerQueueGenerated() bool {
	return match.Type == Practice && match.NameDetail == fillerQueueGeneratedMarker
}

// IsFillerQueueDraft identifies an automatically generated, not-yet-playable match.
func (match *Match) IsFillerQueueDraft() bool {
	return match.IsFillerQueueGenerated() && match.Status == game.MatchHidden
}

func (match *Match) HasBothAlliances() bool {
	return (match.Red1 != 0 || match.Red2 != 0 || match.Red3 != 0) &&
		(match.Blue1 != 0 || match.Blue2 != 0 || match.Blue3 != 0)
}

// CanReserveFillerQueueSlot accounts for vacant stations and pending kiosk requests.
// Specific-station signups reserve their preferred position; "any" signups reserve
// remaining capacity without prematurely assigning a driver station.
func CanReserveFillerQueueSlot(match *Match, station string, waiting []FillerQueueEntry) bool {
	if match.Type != Practice ||
		(match.Status != game.MatchScheduled && !match.IsFillerQueueDraft()) ||
		!match.StartedAt.IsZero() {
		return false
	}
	claimed := make(map[string]bool)
	anyCount := 0
	for _, entry := range waiting {
		if entry.Status != FillerQueueWaiting || entry.MatchId != match.Id {
			continue
		}
		if entry.Station == "" || entry.Station == "ANY" {
			anyCount++
		} else {
			claimed[entry.Station] = true
		}
	}
	free := 0
	for _, id := range fillerQueueStations {
		assigned, err := match.MatchStationTeam(id)
		if err != nil {
			return false
		}
		if assigned == 0 && !claimed[id] {
			free++
		}
	}
	if station != "" && station != "ANY" {
		assigned, err := match.MatchStationTeam(station)
		if err != nil || assigned != 0 || claimed[station] {
			return false
		}
	}
	return free > anyCount
}

// CreateFillerQueueEntryWithExclusion selects a practice match with capacity and
// creates a hidden new one if none exists. Both the match and request are written
// in one transaction to prevent simultaneous kiosks from overbooking a station.
// excludedMatchId is the match currently loaded on the field.
func (database *Database) CreateFillerQueueEntryWithExclusion(entry *FillerQueueEntry, excludedMatchId int) error {
	if entry.TeamId <= 0 {
		return fmt.Errorf("select a valid team")
	}
	if entry.MatchId < 0 {
		return fmt.Errorf("invalid match")
	}
	if entry.Station != "" && entry.Station != "ANY" {
		if _, err := (&Match{}).MatchStationTeam(entry.Station); err != nil {
			return err
		}
	}
	return database.bolt.Update(func(tx *bbolt.Tx) error {
		if tx.Bucket(database.teamTable.bucketKey).Get(idToKey(entry.TeamId)) == nil {
			return fmt.Errorf("team %d is not registered", entry.TeamId)
		}
		queueBucket := tx.Bucket(database.fillerQueueTable.bucketKey)
		matchBucket := tx.Bucket(database.matchTable.bucketKey)
		waiting := make([]FillerQueueEntry, 0)
		if err := queueBucket.ForEach(func(_, data []byte) error {
			var existing FillerQueueEntry
			if err := json.Unmarshal(data, &existing); err != nil {
				return err
			}
			if existing.Status == FillerQueueWaiting {
				if existing.TeamId == entry.TeamId {
					return fmt.Errorf("team %d already has a pending request", entry.TeamId)
				}
				waiting = append(waiting, existing)
			}
			return nil
		}); err != nil {
			return err
		}
		matches := make([]Match, 0)
		highestOrder := 0
		nextTime := time.Now().Add(fillerQueueMatchSpacing)
		if err := matchBucket.ForEach(func(_, data []byte) error {
			var match Match
			if err := json.Unmarshal(data, &match); err != nil {
				return err
			}
			if match.Type == Practice {
				matches = append(matches, match)
				if match.TypeOrder > highestOrder {
					highestOrder = match.TypeOrder
				}
				if !match.Time.IsZero() && !match.Time.Before(nextTime) {
					nextTime = match.Time.Add(fillerQueueMatchSpacing)
				}
			}
			return nil
		}); err != nil {
			return err
		}
		sort.Slice(matches, func(i, j int) bool {
			if matches[i].TypeOrder != matches[j].TypeOrder {
				return matches[i].TypeOrder < matches[j].TypeOrder
			}
			return matches[i].Id < matches[j].Id
		})

		var selected *Match
		for i := range matches {
			match := &matches[i]
			if entry.MatchId != 0 && match.Id != entry.MatchId {
				continue
			}
			if match.Id == excludedMatchId {
				if entry.MatchId != 0 {
					return fmt.Errorf("selected match is already loaded on the field")
				}
				continue
			}
			if match.HasTeam(entry.TeamId) {
				if entry.MatchId != 0 {
					return fmt.Errorf("team is already scheduled in that match")
				}
				continue
			}
			if !CanReserveFillerQueueSlot(match, entry.Station, waiting) {
				if entry.MatchId != 0 {
					return fmt.Errorf("selected match has no available spot matching that preference")
				}
				continue
			}
			selected = match
			break
		}
		if selected == nil {
			if entry.MatchId != 0 {
				return fmt.Errorf("requested match is unavailable")
			}
			order := highestOrder + 1
			newMatch := Match{
				Type: Practice,
				TypeOrder: order,
				Time: nextTime,
				LongName: fmt.Sprintf("Practice %d", order),
				ShortName: fmt.Sprintf("P%d", order),
				NameDetail: fillerQueueGeneratedMarker,
				Status: game.MatchHidden,
				TbaMatchKey: TbaMatchKey{CompLevel: "p", MatchNumber: order},
			}
			id, err := matchBucket.NextSequence()
			if err != nil {
				return err
			}
			newMatch.Id = int(id)
			data, err := json.Marshal(&newMatch)
			if err != nil {
				return err
			}
			if err := matchBucket.Put(idToKey(newMatch.Id), data); err != nil {
				return err
			}
			selected = &newMatch
			entry.CreatedNewMatch = true
		}
		id, err := queueBucket.NextSequence()
		if err != nil {
			return err
		}
		entry.Id = int(id)
		entry.MatchId = selected.Id
		entry.Status = FillerQueueWaiting
		entry.CreatedAt = time.Now()
		entry.UpdatedAt = entry.CreatedAt
		data, err := json.Marshal(entry)
		if err != nil {
			return err
		}
		return queueBucket.Put(idToKey(entry.Id), data)
	})
}
