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

// LineupRevision identifies the exact schedule state an operator opened in
// the edit dialog. It is used to reject stale edits after a kiosk assignment.
func (match *Match) LineupRevision() string {
	return fmt.Sprintf("%d:%d:%d:%d:%d:%d:%d:%d",
		match.Red1, match.Red2, match.Red3,
		match.Blue1, match.Blue2, match.Blue3,
		match.Time.UnixNano(), match.Status)
}

// UpdatePracticeMatchIfUnchanged performs an optimistic, atomic update so a
// stale operator form cannot erase a kiosk's new driver station assignment.
func (database *Database) UpdatePracticeMatchIfUnchanged(match *Match, revision string) error {
	if revision == "" {
		return fmt.Errorf("missing match revision; refresh and try again")
	}
	return database.bolt.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(database.matchTable.bucketKey)
		stored := bucket.Get(idToKey(match.Id))
		if stored == nil {
			return fmt.Errorf("practice match no longer exists")
		}
		var current Match
		if err := json.Unmarshal(stored, &current); err != nil {
			return err
		}
		if current.Type != Practice || (current.Status != game.MatchScheduled && !current.IsFillerQueueDraft()) ||
			!current.StartedAt.IsZero() {
			return fmt.Errorf("match is no longer editable")
		}
		if current.LineupRevision() != revision {
			return fmt.Errorf("match was changed by a kiosk or another operator; refresh before saving")
		}
		data, err := json.Marshal(match)
		if err != nil {
			return err
		}
		return bucket.Put(idToKey(match.Id), data)
	})
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

// selectFillerQueueStation chooses an available station while preserving any
// outstanding kiosk reservations. For 'any', fill the less-populated alliance
// first so two signups produce a playable match rather than a one-sided one.
func selectFillerQueueStation(match *Match, preferred string, waiting []FillerQueueEntry) (string, bool) {
	if !CanReserveFillerQueueSlot(match, preferred, waiting) {
		return "", false
	}
	if preferred != "" && preferred != "ANY" {
		return preferred, true
	}

	redCount, blueCount := 0, 0
	for _, station := range fillerQueueStations {
		team, _ := match.MatchStationTeam(station)
		if team != 0 {
			if station[0] == 'R' {
				redCount++
			} else {
				blueCount++
			}
		}
	}
	stationOrder := []string{"R1", "R2", "R3", "B1", "B2", "B3"}
	if redCount > blueCount {
		stationOrder = []string{"B1", "B2", "B3", "R1", "R2", "R3"}
	}
	reserved := make(map[string]bool)
	for _, entry := range waiting {
		if entry.Status == FillerQueueWaiting && entry.MatchId == match.Id &&
			entry.Station != "" && entry.Station != "ANY" {
			reserved[entry.Station] = true
		}
	}
	for _, station := range stationOrder {
		team, _ := match.MatchStationTeam(station)
		if team == 0 && !reserved[station] {
			return station, true
		}
	}
	return "", false
}

// CreateFillerQueueEntryWithExclusion selects a practice match with capacity and
// creates a hidden new one if none exists. When assignment requires no operator
// intervention, a station is selected and populated immediately, atomically with
// the kiosk record. This prevents simultaneous kiosks from overbooking a station.
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
		// Automatically place the team if the match is not loaded or started
		// and its requested station has capacity. New matches remain hidden
		// until both alliances have a robot.
		assignedStation, ok := selectFillerQueueStation(selected, entry.Station, waiting)
		if !ok {
			return fmt.Errorf("no free station available for this request")
		}
		if selected.HasTeam(entry.TeamId) {
			return fmt.Errorf("team %d is already assigned to %s", entry.TeamId, selected.ShortName)
		}
		if err := selected.SetMatchStationTeam(assignedStation, entry.TeamId); err != nil {
			return err
		}
		if selected.IsFillerQueueDraft() && selected.HasBothAlliances() {
			selected.Status = game.MatchScheduled
		}
		updatedMatch, err := json.Marshal(selected)
		if err != nil {
			return err
		}
		if err := matchBucket.Put(idToKey(selected.Id), updatedMatch); err != nil {
			return err
		}

		id, err := queueBucket.NextSequence()
		if err != nil {
			return err
		}
		entry.Id = int(id)
		entry.MatchId = selected.Id
		entry.Status = FillerQueueAssigned
		entry.AssignedMatchId = selected.Id
		entry.AssignedStation = assignedStation
		entry.CreatedAt = time.Now()
		entry.UpdatedAt = entry.CreatedAt
		data, err := json.Marshal(entry)
		if err != nil {
			return err
		}
		return queueBucket.Put(idToKey(entry.Id), data)
	})
}
