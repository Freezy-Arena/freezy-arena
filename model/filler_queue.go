package model

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/Team254/cheesy-arena/game"
	"go.etcd.io/bbolt"
)

// FillerQueueEntry records a kiosk signup independently of the official match schedule.
// A kiosk can request a match, station, both, or neither. Only an operator can assign it.
type FillerQueueEntry struct {
	Id              int `db:"id"`
	TeamId          int
	MatchId         int
	Station         string
	Status          string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	AssignedMatchId int
	AssignedStation string
	CreatedNewMatch bool
}

const (
	FillerQueueWaiting   = "waiting"
	FillerQueueAssigned  = "assigned"
	FillerQueueCancelled = "cancelled"
)

// MatchStationTeam returns the team assigned to a station, with zero representing a vacancy.
func (match *Match) MatchStationTeam(station string) (int, error) {
	switch station {
	case "R1":
		return match.Red1, nil
	case "R2":
		return match.Red2, nil
	case "R3":
		return match.Red3, nil
	case "B1":
		return match.Blue1, nil
	case "B2":
		return match.Blue2, nil
	case "B3":
		return match.Blue3, nil
	}
	return 0, fmt.Errorf("invalid driver station %q", station)
}

func (match *Match) SetMatchStationTeam(station string, teamId int) error {
	switch station {
	case "R1":
		match.Red1 = teamId
	case "R2":
		match.Red2 = teamId
	case "R3":
		match.Red3 = teamId
	case "B1":
		match.Blue1 = teamId
	case "B2":
		match.Blue2 = teamId
	case "B3":
		match.Blue3 = teamId
	default:
		return fmt.Errorf("invalid driver station %q", station)
	}
	return nil
}

func (match *Match) HasTeam(teamId int) bool {
	if teamId == 0 {
		return false
	}
	return match.Red1 == teamId || match.Red2 == teamId || match.Red3 == teamId ||
		match.Blue1 == teamId || match.Blue2 == teamId || match.Blue3 == teamId
}

func (database *Database) GetAllFillerQueueEntries() ([]FillerQueueEntry, error) {
	entries, err := database.fillerQueueTable.getAll()
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Id < entries[j].Id
	})
	return entries, nil
}

// CreateFillerQueueEntry reserves an open spot or creates a hidden practice match
// when there is no existing match with capacity.
func (database *Database) CreateFillerQueueEntry(entry *FillerQueueEntry) error {
	return database.CreateFillerQueueEntryWithExclusion(entry, 0)
}

// AssignFillerQueueEntry changes both the schedule and request in one transaction.
// It cannot overwrite a filled station or place the same team twice in a match.
func (database *Database) AssignFillerQueueEntry(entryId, matchId int, station string) error {
	if matchId <= 0 {
		return fmt.Errorf("choose a practice match")
	}
	if _, err := (&Match{}).MatchStationTeam(station); err != nil {
		return err
	}
	return database.bolt.Update(func(tx *bbolt.Tx) error {
		queueBucket := tx.Bucket(database.fillerQueueTable.bucketKey)
		matchBucket := tx.Bucket(database.matchTable.bucketKey)
		entryBytes := queueBucket.Get(idToKey(entryId))
		if entryBytes == nil {
			return fmt.Errorf("queue request not found")
		}
		var entry FillerQueueEntry
		if err := json.Unmarshal(entryBytes, &entry); err != nil {
			return err
		}
		if entry.Status != FillerQueueWaiting {
			return fmt.Errorf("request is no longer waiting")
		}
		if entry.MatchId != 0 && entry.MatchId != matchId {
			return fmt.Errorf("request is for a different match")
		}
		if entry.Station != "" && entry.Station != "ANY" && entry.Station != station {
			return fmt.Errorf("request is for a different driver station")
		}
		matchBytes := matchBucket.Get(idToKey(matchId))
		if matchBytes == nil {
			return fmt.Errorf("match no longer exists")
		}
		var match Match
		if err := json.Unmarshal(matchBytes, &match); err != nil {
			return err
		}
		if match.Type != Practice || (match.Status != game.MatchScheduled && !match.IsFillerQueueDraft()) || !match.StartedAt.IsZero() {
			return fmt.Errorf("only unplayed practice matches can be filled")
		}
		assigned, err := match.MatchStationTeam(station)
		if err != nil {
			return err
		}
		if assigned != 0 {
			return fmt.Errorf("%s is already occupied in %s", station, match.ShortName)
		}
		if match.HasTeam(entry.TeamId) {
			return fmt.Errorf("team %d is already assigned to %s", entry.TeamId, match.ShortName)
		}
		// Respect other pending requests when choosing this team’s station.
		otherRequests := make([]FillerQueueEntry, 0)
		if err := queueBucket.ForEach(func(_, data []byte) error {
			var other FillerQueueEntry
			if err := json.Unmarshal(data, &other); err != nil {
				return err
			}
			if other.Id != entry.Id && other.Status == FillerQueueWaiting {
				otherRequests = append(otherRequests, other)
			}
			return nil
		}); err != nil {
			return err
		}
		if !CanReserveFillerQueueSlot(&match, station, otherRequests) {
			return fmt.Errorf("station is reserved by another pending request")
		}
		if err := match.SetMatchStationTeam(station, entry.TeamId); err != nil {
			return err
		}
		if match.IsFillerQueueDraft() && match.HasBothAlliances() {
			match.Status = game.MatchScheduled
		}
		matchData, err := json.Marshal(&match)
		if err != nil {
			return err
		}
		entry.Status = FillerQueueAssigned
		entry.AssignedMatchId = matchId
		entry.AssignedStation = station
		entry.UpdatedAt = time.Now()
		entryData, err := json.Marshal(&entry)
		if err != nil {
			return err
		}
		if err := matchBucket.Put(idToKey(match.Id), matchData); err != nil {
			return err
		}
		return queueBucket.Put(idToKey(entry.Id), entryData)
	})
}

func (database *Database) CancelFillerQueueEntry(entryId int) error {
	return database.bolt.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(database.fillerQueueTable.bucketKey)
		value := bucket.Get(idToKey(entryId))
		if value == nil {
			return fmt.Errorf("queue request not found")
		}
		var entry FillerQueueEntry
		if err := json.Unmarshal(value, &entry); err != nil {
			return err
		}
		if entry.Status != FillerQueueWaiting {
			return fmt.Errorf("only waiting requests can be cancelled")
		}
		entry.Status = FillerQueueCancelled
		entry.UpdatedAt = time.Now()
		data, err := json.Marshal(entry)
		if err != nil {
			return err
		}
		return bucket.Put(idToKey(entry.Id), data)
	})
}
