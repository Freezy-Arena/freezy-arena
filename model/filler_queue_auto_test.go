package model

import (
	"sync"
	"testing"

	"github.com/Team254/cheesy-arena/game"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Demand-driven kiosk requests must never create an empty, playable match.
func TestFillerQueueAutomaticallyCreatesAndExtendsDraftMatches(t *testing.T) {
	db := setupTestDb(t)
	defer db.Close()
	for team := 1; team <= 8; team++ {
		require.NoError(t, db.CreateTeam(&Team{Id: team}))
	}
	entries := make([]*FillerQueueEntry, 7)
	for index := range entries {
		entry := &FillerQueueEntry{TeamId: index + 1, Station: "ANY"}
		require.NoError(t, db.CreateFillerQueueEntry(entry))
		entries[index] = entry
	}
	first := entries[0].MatchId
	assert.True(t, entries[0].CreatedNewMatch)
	for i := 1; i < 6; i++ {
		assert.Equal(t, first, entries[i].MatchId)
		assert.False(t, entries[i].CreatedNewMatch)
	}
	assert.NotEqual(t, first, entries[6].MatchId)
	assert.True(t, entries[6].CreatedNewMatch)

	matches, err := db.GetMatchesByType(Practice, true)
	require.NoError(t, err)
	require.Len(t, matches, 2)
	assert.Equal(t, 1, matches[0].TypeOrder)
	assert.Equal(t, 2, matches[1].TypeOrder)
	assert.True(t, matches[0].HasBothAlliances())
	assert.Equal(t, game.MatchScheduled, matches[0].Status)
	assert.Equal(t, game.MatchHidden, matches[1].Status)
	assert.True(t, matches[1].IsFillerQueueDraft())
	assert.False(t, matches[1].HasBothAlliances())
	for _, entry := range entries {
		assert.Equal(t, FillerQueueAssigned, entry.Status)
		assert.NotEmpty(t, entry.AssignedStation)
	}
	assert.True(t, matches[1].Time.After(matches[0].Time))
}

func TestFillerQueueDraftBecomesPlayableOnlyWithBothAlliances(t *testing.T) {
	db := setupTestDb(t)
	defer db.Close()
	require.NoError(t, db.CreateTeam(&Team{Id: 4728}))
	require.NoError(t, db.CreateTeam(&Team{Id: 254}))

	red := &FillerQueueEntry{TeamId: 4728, Station: "R1"}
	require.NoError(t, db.CreateFillerQueueEntry(red))
	match, err := db.GetMatchById(red.MatchId)
	require.NoError(t, err)
	assert.True(t, match.IsFillerQueueDraft()) // One alliance alone is not runnable.

	blue := &FillerQueueEntry{TeamId: 254, Station: "B3"}
	require.NoError(t, db.CreateFillerQueueEntry(blue))
	require.Equal(t, red.MatchId, blue.MatchId)
	assert.Equal(t, "B3", blue.AssignedStation)
	require.Error(t, db.AssignFillerQueueEntry(red.Id, red.MatchId, "R1")) // Already assigned automatically.
	match, err = db.GetMatchById(red.MatchId)
	require.NoError(t, err)
	assert.Equal(t, game.MatchScheduled, match.Status)
	assert.True(t, match.HasBothAlliances())
	assert.Equal(t, 4728, match.Red1)
	assert.Equal(t, 254, match.Blue3)
}

func TestFillerQueueRespectsPreferredStationsAndExistingMatch(t *testing.T) {
	db := setupTestDb(t)
	defer db.Close()
	for _, team := range []int{4728, 254, 1114, 1678} {
		require.NoError(t, db.CreateTeam(&Team{Id: team}))
	}
	first := &FillerQueueEntry{TeamId: 4728, Station: "R1"}
	require.NoError(t, db.CreateFillerQueueEntry(first))
	second := &FillerQueueEntry{TeamId: 254, Station: "R1"}
	require.NoError(t, db.CreateFillerQueueEntry(second))
	assert.NotEqual(t, first.MatchId, second.MatchId)
	require.Error(t, db.CreateFillerQueueEntry(&FillerQueueEntry{TeamId: 1114, Station: "R1", MatchId: first.MatchId}))
	require.Error(t, db.AssignFillerQueueEntry(first.Id, second.MatchId, "R1"))
	// A specifically reserved R1 must not be consumed by a different waiting request.
	other := &FillerQueueEntry{TeamId: 1114, Station: "ANY", MatchId: first.MatchId}
	require.NoError(t, db.CreateFillerQueueEntry(other))
	require.Error(t, db.AssignFillerQueueEntry(other.Id, first.MatchId, "R1"))

	// Skip a match already loaded on the field when auto-selecting a slot.
	third := &FillerQueueEntry{TeamId: 1678, Station: "B1"}
	require.NoError(t, db.CreateFillerQueueEntryWithExclusion(third, first.MatchId))
	assert.NotEqual(t, first.MatchId, third.MatchId)
}

func TestFillerQueueMultipleKiosksDoNotOverbook(t *testing.T) {
	db := setupTestDb(t)
	defer db.Close()
	for team := 1; team <= 8; team++ {
		require.NoError(t, db.CreateTeam(&Team{Id: team}))
	}
	entries := make([]*FillerQueueEntry, 8)
	errs := make([]error, 8)
	var wg sync.WaitGroup
	for i := range entries {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			entry := &FillerQueueEntry{TeamId: i + 1, Station: "ANY"}
			errs[i] = db.CreateFillerQueueEntry(entry)
			entries[i] = entry
		}()
	}
	wg.Wait()
	for _, err := range errs {
		require.NoError(t, err)
	}
	count := make(map[int]int)
	for _, entry := range entries {
		count[entry.MatchId]++
	}
	assert.Len(t, count, 2)
	for _, n := range count {
		assert.LessOrEqual(t, n, 6)
	}
}

func TestFillerQueueAnyStationBalancesAlliances(t *testing.T) {
	db := setupTestDb(t)
	defer db.Close()
	for team := 1; team <= 6; team++ {
		require.NoError(t, db.CreateTeam(&Team{Id: team}))
	}
	want := []string{"R1", "B1", "R2", "B2", "R3", "B3"}
	for i, expectedStation := range want {
		entry := &FillerQueueEntry{TeamId: i + 1, Station: "ANY"}
		require.NoError(t, db.CreateFillerQueueEntry(entry))
		assert.Equal(t, expectedStation, entry.AssignedStation)
		assert.Equal(t, FillerQueueAssigned, entry.Status)
	}
	matches, err := db.GetMatchesByType(Practice, true)
	require.NoError(t, err)
	require.Len(t, matches, 1)
	assert.Equal(t, game.MatchScheduled, matches[0].Status)
	assert.True(t, matches[0].HasBothAlliances())
}

func TestFillerQueueAutoAssignRespectsWaitingReservations(t *testing.T) {
	db := setupTestDb(t)
	defer db.Close()
	for _, team := range []int{4728, 254, 1114} {
		require.NoError(t, db.CreateTeam(&Team{Id: team}))
	}
	match := &Match{Type: Practice, TypeOrder: 1, ShortName: "P1", Status: game.MatchScheduled}
	require.NoError(t, db.CreateMatch(match))
	// A legacy waiting request has reserved R1, and must not be displaced.
	require.NoError(t, db.fillerQueueTable.create(&FillerQueueEntry{
		TeamId: 4728, MatchId: match.Id, Station: "R1", Status: FillerQueueWaiting,
	}))
	auto := &FillerQueueEntry{TeamId: 254, MatchId: match.Id, Station: "ANY"}
	require.NoError(t, db.CreateFillerQueueEntry(auto))
	assert.Equal(t, "R2", auto.AssignedStation)
	updated, err := db.GetMatchById(match.Id)
	require.NoError(t, err)
	assert.Zero(t, updated.Red1)
	assert.Equal(t, 254, updated.Red2)
	require.Error(t, db.CreateFillerQueueEntry(&FillerQueueEntry{
		TeamId: 1114, MatchId: match.Id, Station: "R1",
	}))
}
