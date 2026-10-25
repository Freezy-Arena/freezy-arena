package web

import (
	"net/url"
	"strconv"
	"testing"

	"github.com/Team254/cheesy-arena/game"
	"github.com/Team254/cheesy-arena/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPracticeSchedulingWithEmptyStations(t *testing.T) {
	web := setupTestWeb(t)
	require.NoError(t, web.arena.Database.CreateTeam(&model.Team{Id: 4728}))
	require.NoError(t, web.arena.Database.CreateTeam(&model.Team{Id: 254}))
	recorder := web.postHttpResponse("/panel/freezy/add_practice_match", "red1=4728&blue2=254&matchTime=2026-11-02T13:30")
	require.Equal(t, 303, recorder.Code)
	matches, err := web.arena.Database.GetMatchesByType(model.Practice, true)
	require.NoError(t, err)
	require.Len(t, matches, 1)
	assert.Equal(t, 4728, matches[0].Red1)
	assert.Zero(t, matches[0].Red2)
	assert.Equal(t, 254, matches[0].Blue2)
	assert.Equal(t, game.MatchScheduled, matches[0].Status)
	assert.Equal(t, 13, matches[0].Time.Hour())
	assert.Equal(t, 30, matches[0].Time.Minute())

	recorder = web.postHttpResponse("/panel/freezy/add_practice_match", "red1=4728&red2=4728&blue1=254")
	assert.Equal(t, 400, recorder.Code)
	recorder = web.postHttpResponse("/panel/freezy/add_practice_match", "red1=4728")
	assert.Equal(t, 400, recorder.Code)
}

func TestFillerKioskRequestAndOperatorAssignment(t *testing.T) {
	web := setupTestWeb(t)
	for _, id := range []int{4728, 254, 1114} {
		require.NoError(t, web.arena.Database.CreateTeam(&model.Team{Id: id}))
	}
	recorder := web.postHttpResponse("/panel/freezy/add_practice_match", "red1=4728&blue1=254")
	require.Equal(t, 303, recorder.Code)
	matches, err := web.arena.Database.GetMatchesByType(model.Practice, true)
	require.NoError(t, err)
	match := matches[0]

	recorder = web.kioskGetHttpResponse("/filler/kiosk")
	assert.Equal(t, 200, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Join Filler Queue")
	recorder = web.kioskPostHttpResponse("/filler/kiosk", url.Values{
		"teamId": {"1114"}, "matchId": {strconv.Itoa(match.Id)}, "station": {"R2"},
	}.Encode())
	require.Equal(t, 303, recorder.Code)
	entries, err := web.arena.Database.GetAllFillerQueueEntries()
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, model.FillerQueueAssigned, entries[0].Status)
	assert.Equal(t, "R2", entries[0].AssignedStation)

	recorder = web.postHttpResponse("/panel/freezy/filler/assign", url.Values{
		"entryId": {strconv.Itoa(entries[0].Id)}, "matchId": {strconv.Itoa(match.Id)}, "station": {"R2"},
	}.Encode())
	require.Equal(t, 409, recorder.Code) // Already assigned by the kiosk.
	updated, err := web.arena.Database.GetMatchById(match.Id)
	require.NoError(t, err)
	assert.Equal(t, 1114, updated.Red2)
}

func TestFillerQueueAdminRevisionUpdatesWhenKioskAssigns(t *testing.T) {
	web := setupTestWeb(t)
	require.NoError(t, web.arena.Database.CreateTeam(&model.Team{Id: 4728}))
	initial := web.getHttpResponse("/panel/freezy/filler/revision")
	require.Equal(t, 200, initial.Code)
	assert.Contains(t, initial.Header().Get("Content-Type"), "application/json")
	assert.Equal(t, "no-store", initial.Header().Get("Cache-Control"))
	page := web.getHttpResponse("/panel/freezy/add_practice_match")
	require.Equal(t, 200, page.Code)
	assert.Contains(t, page.Body.String(), "filler_queue_admin.js")
	assert.Contains(t, page.Body.String(), "Live updates enabled")
	assert.Contains(t, page.Body.String(), "Recent Assignments")

	require.Equal(t, 303, web.kioskPostHttpResponse("/filler/kiosk", "teamId=4728&station=ANY").Code)
	changed := web.getHttpResponse("/panel/freezy/filler/revision")
	require.Equal(t, 200, changed.Code)
	assert.NotEqual(t, initial.Body.String(), changed.Body.String())
	updatedPage := web.getHttpResponse("/panel/freezy/add_practice_match")
	assert.Contains(t, updatedPage.Body.String(), "Recent Assignments")
	assert.Contains(t, updatedPage.Body.String(), "4728")
	assert.Contains(t, updatedPage.Body.String(), "R1")
}

func TestPracticeEditCannotOverwriteConcurrentKioskAssignment(t *testing.T) {
	web := setupTestWeb(t)
	for _, id := range []int{4728, 254, 1114} {
		require.NoError(t, web.arena.Database.CreateTeam(&model.Team{Id: id}))
	}
	require.Equal(t, 303, web.postHttpResponse("/panel/freezy/add_practice_match", "red1=4728&blue1=254").Code)
	matches, err := web.arena.Database.GetMatchesByType(model.Practice, true)
	require.NoError(t, err)
	require.Len(t, matches, 1)
	oldRevision := matches[0].LineupRevision()
	require.Equal(t, 303, web.kioskPostHttpResponse("/filler/kiosk", "teamId=1114&station=ANY").Code)

	staleEdit := url.Values{
		"matchId": {strconv.Itoa(matches[0].Id)},
		"matchRevision": {oldRevision},
		"red1": {"4728"}, "blue1": {"254"},
	}
	response := web.postHttpResponse("/panel/freezy/edit_practice_match", staleEdit.Encode())
	assert.Equal(t, 409, response.Code)
	current, err := web.arena.Database.GetMatchById(matches[0].Id)
	require.NoError(t, err)
	assert.Equal(t, 1114, current.Red2) // Kiosk's slot was preserved.

	staleEdit.Set("matchRevision", current.LineupRevision())
	staleEdit.Set("red2", "1114")
	response = web.postHttpResponse("/panel/freezy/edit_practice_match", staleEdit.Encode())
	assert.Equal(t, 303, response.Code)
}
