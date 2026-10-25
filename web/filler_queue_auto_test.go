package web

import (
	"strings"
	"testing"

	"github.com/Team254/cheesy-arena/game"
	"github.com/Team254/cheesy-arena/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKioskCreatesMatchWhenNoPracticeScheduleExists(t *testing.T) {
	web := setupTestWeb(t)
	for _, team := range []int{4728, 254} {
		require.NoError(t, web.arena.Database.CreateTeam(&model.Team{Id: team}))
	}
	page := web.kioskGetHttpResponse("/filler/kiosk")
	require.Equal(t, 200, page.Code)
	assert.Contains(t, page.Body.String(), "automatically create if needed")

	first := web.kioskPostHttpResponse("/filler/kiosk", "teamId=4728&station=R1")
	require.Equal(t, 303, first.Code)
	assert.Contains(t, first.Header().Get("Location"), "P1")
	matches, err := web.arena.Database.GetMatchesByType(model.Practice, true)
	require.NoError(t, err)
	require.Len(t, matches, 1)
	assert.True(t, matches[0].IsFillerQueueDraft())
	assert.Equal(t, game.MatchHidden, matches[0].Status)

	second := web.kioskPostHttpResponse("/filler/kiosk", "teamId=254&station=B1")
	require.Equal(t, 303, second.Code)
	matches, err = web.arena.Database.GetMatchesByType(model.Practice, true)
	require.NoError(t, err)
	require.Len(t, matches, 1) // Reuse the same draft rather than opening P2.
	assert.Equal(t, game.MatchScheduled, matches[0].Status)
	assert.True(t, matches[0].HasBothAlliances())

	entries, err := web.arena.Database.GetAllFillerQueueEntries()
	require.NoError(t, err)
	require.Len(t, entries, 2)
	for _, entry := range entries {
		assert.Equal(t, matches[0].Id, entry.MatchId)
		assert.Equal(t, model.FillerQueueAssigned, entry.Status)
		assert.NotEmpty(t, entry.AssignedStation)
	}
	operatorPage := web.getHttpResponse("/panel/freezy/add_practice_match")
	require.Equal(t, 200, operatorPage.Code)
	assert.Contains(t, operatorPage.Body.String(), "Recent Assignments")
	assert.Contains(t, operatorPage.Body.String(), "P1")
	ready, err := web.arena.Database.GetMatchById(matches[0].Id)
	require.NoError(t, err)
	assert.Equal(t, game.MatchScheduled, ready.Status)
	assert.Equal(t, 4728, ready.Red1)
	assert.Equal(t, 254, ready.Blue1)
}

func TestKioskAutoCreatesNextMatchWhenRequestedStationIsTaken(t *testing.T) {
	web := setupTestWeb(t)
	for _, team := range []int{4728, 1114} {
		require.NoError(t, web.arena.Database.CreateTeam(&model.Team{Id: team}))
	}
	require.Equal(t, 303, web.kioskPostHttpResponse("/filler/kiosk", "teamId=4728&station=R1").Code)
	require.Equal(t, 303, web.kioskPostHttpResponse("/filler/kiosk", "teamId=1114&station=R1").Code)
	matches, err := web.arena.Database.GetMatchesByType(model.Practice, true)
	require.NoError(t, err)
	require.Len(t, matches, 2)
	assert.Equal(t, "P1", matches[0].ShortName)
	assert.Equal(t, "P2", matches[1].ShortName)

	// Choosing a full or reserved specific station must not silently switch matches.
	recorder := web.kioskPostHttpResponse("/filler/kiosk", "teamId=1114&matchId=1&station=R1")
	assert.Equal(t, 409, recorder.Code)
	assert.True(t, strings.Contains(recorder.Body.String(), "already has a pending request") ||
		strings.Contains(recorder.Body.String(), "no available spot"))
}

func TestKioskAutoFillsExistingOpenMatchWithoutPreference(t *testing.T) {
	web := setupTestWeb(t)
	for _, id := range []int{4728, 254, 1114} {
		require.NoError(t, web.arena.Database.CreateTeam(&model.Team{Id: id}))
	}
	require.Equal(t, 303, web.postHttpResponse("/panel/freezy/add_practice_match", "red1=4728&blue1=254").Code)
	response := web.kioskPostHttpResponse("/filler/kiosk", "teamId=1114&station=ANY")
	require.Equal(t, 303, response.Code)
	assert.Contains(t, response.Header().Get("Location"), "R2")
	matches, err := web.arena.Database.GetMatchesByType(model.Practice, true)
	require.NoError(t, err)
	require.Len(t, matches, 1) // No new match because P1 has capacity.
	assert.Equal(t, 1114, matches[0].Red2)
	entries, err := web.arena.Database.GetAllFillerQueueEntries()
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, model.FillerQueueAssigned, entries[0].Status)
	assert.Equal(t, "R2", entries[0].AssignedStation)
}
