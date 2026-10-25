package model

import (
	"testing"

	"github.com/Team254/cheesy-arena/game"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFillerQueueAssignmentAndPersistence(t *testing.T) {
	db := setupTestDb(t)
	defer db.Close()
	require.NoError(t, db.CreateTeam(&Team{Id: 4728}))
	require.NoError(t, db.CreateTeam(&Team{Id: 254}))
	match := &Match{Type: Practice, TypeOrder: 1, Status: game.MatchScheduled, Red1: 254, Blue1: 4728, ShortName: "P1"}
	require.NoError(t, db.CreateMatch(match))
	entry := &FillerQueueEntry{TeamId: 254, MatchId: match.Id, Station: "R2"}
	require.Error(t, db.CreateFillerQueueEntry(entry)) // Already scheduled.
	require.NoError(t, db.CreateTeam(&Team{Id: 1114}))
	entry = &FillerQueueEntry{TeamId: 1114, MatchId: match.Id, Station: "R2"}
	require.NoError(t, db.CreateFillerQueueEntry(entry))
	assert.Equal(t, FillerQueueAssigned, entry.Status)
	assert.Equal(t, "R2", entry.AssignedStation)
	// The team is already assigned; it cannot request another position in P1.
	require.Error(t, db.CreateFillerQueueEntry(&FillerQueueEntry{TeamId: 1114, MatchId: match.Id}))
	require.Error(t, db.AssignFillerQueueEntry(entry.Id, match.Id, "R1")) // Already assigned.
	require.Error(t, db.AssignFillerQueueEntry(entry.Id, match.Id, "B2")) // Already assigned.
	updated, err := db.GetMatchById(match.Id)
	require.NoError(t, err)
	assert.Equal(t, 1114, updated.Red2)
	entries, err := db.GetAllFillerQueueEntries()
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, FillerQueueAssigned, entries[0].Status)
	require.Error(t, db.AssignFillerQueueEntry(entry.Id, match.Id, "R2"))
}

func TestFillerQueueCancellation(t *testing.T) {
	db := setupTestDb(t)
	defer db.Close()
	require.NoError(t, db.CreateTeam(&Team{Id: 4728}))
	entry := &FillerQueueEntry{TeamId: 4728, Station: "ANY", Status: FillerQueueWaiting}
	require.NoError(t, db.fillerQueueTable.create(entry)) // Existing manual-review request.
	require.NoError(t, db.CancelFillerQueueEntry(entry.Id))
	require.Error(t, db.CancelFillerQueueEntry(entry.Id))
	require.NoError(t, db.CreateFillerQueueEntry(&FillerQueueEntry{TeamId: 4728}))
}

func TestMatchStationTeam(t *testing.T) {
	match := &Match{Red1: 4728, Blue3: 254}
	id, err := match.MatchStationTeam("R1")
	require.NoError(t, err)
	assert.Equal(t, 4728, id)
	require.NoError(t, match.SetMatchStationTeam("B2", 1114))
	assert.True(t, match.HasTeam(1114))
	assert.False(t, match.HasTeam(0))
	_, err = match.MatchStationTeam("R4")
	require.Error(t, err)
}
