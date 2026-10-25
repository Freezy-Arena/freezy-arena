// Copyright 2026 Freezy Arena contributors.
// Author: Justin Fischer <justin@justfishin99.xyz>

package web

import (
	"fmt"
	"github.com/Team254/cheesy-arena/model"
	"github.com/Team254/cheesy-arena/tournament"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/url"
	"testing"
	"time"
)

func breakPostRows(items ...model.PlayoffBreakSetting) string {
	values := url.Values{"action": {"save"}}
	for _, item := range items {
		values.Add("afterMatchOrder", fmt.Sprint(item.AfterMatchOrder))
		values.Add("durationMinutes", fmt.Sprint(item.DurationSec/60))
		values.Add("description", item.Description)
	}
	return values.Encode()
}

func TestSetupBreaksShowsBracketDefaults(t *testing.T) {
	web := setupTestWeb(t)
	response := web.getHttpResponse("/setup/breaks")
	assert.Equal(t, http.StatusOK, response.Code)
	assert.Contains(t, response.Body.String(), "Break After Match")
	assert.Contains(t, response.Body.String(), "Restore Defaults")
	assert.Contains(t, response.Body.String(), "Add Break")
	assert.Contains(t, response.Body.String(), "value=\"8\" selected")
	assert.Contains(t, response.Body.String(), "Field Break")
	assert.Contains(t, response.Body.String(), "Awards Break")
	assert.Contains(t, response.Body.String(), "value=\"5\"")
	assert.Equal(t, 6, len(web.arena.PlayoffTournament.ConfiguredBreaks()))
}

func TestSetupBreaksBeforeTournamentCreation(t *testing.T) {
	web := setupTestWeb(t)
	custom := []model.PlayoffBreakSetting{
		{AfterMatchOrder: 2, DurationSec: 300, Description: "Robot Inspection"},
		{AfterMatchOrder: 8, DurationSec: 1200, Description: "Lunch Break"},
	}
	response := web.postHttpResponse("/setup/breaks", breakPostRows(custom...))
	assert.Equal(t, http.StatusSeeOther, response.Code)
	assert.NotEmpty(t, web.arena.EventSettings.PlayoffBreaksJSON)

	// Saved break locations are reloaded when the application restarts.
	assert.NoError(t, web.arena.CreatePlayoffTournament())
	assert.Equal(t, custom, web.arena.PlayoffTournament.ConfiguredBreaks())
	response = web.getHttpResponse("/setup/breaks")
	assert.Contains(t, response.Body.String(), "Robot Inspection")
	assert.Contains(t, response.Body.String(), "Lunch Break")

	tournament.CreateTestAlliances(web.arena.Database, 8)
	start := time.Unix(5000, 0)
	assert.NoError(t, web.arena.CreatePlayoffMatches(start))
	breaks, err := web.arena.Database.GetScheduledBreaksByMatchType(model.Playoff)
	assert.NoError(t, err)
	if assert.Len(t, breaks, 2) {
		assert.Equal(t, 3, breaks[0].TypeOrderBefore)
		assert.Equal(t, 300, breaks[0].DurationSec)
		assert.Equal(t, "Robot Inspection", breaks[0].Description)
		assert.Equal(t, 9, breaks[1].TypeOrderBefore)
		assert.Equal(t, 1200, breaks[1].DurationSec)
		assert.Equal(t, "Lunch Break", breaks[1].Description)
		match3, err := web.arena.Database.GetMatchByTypeOrder(model.Playoff, 3)
		assert.NoError(t, err)
		assert.Equal(t, 300*time.Second, match3.Time.Sub(breaks[0].Time))
	}
}

func TestSetupBreaksReschedulesExistingMatches(t *testing.T) {
	web := setupTestWeb(t)
	start := time.Date(2026, 10, 25, 10, 0, 0, 0, time.Local)
	matches := []*model.Match{
		{Type: model.Playoff, TypeOrder: 1, LongName: "Match 1", Time: start},
		{Type: model.Playoff, TypeOrder: 2, LongName: "Match 2", Time: start.Add(15 * time.Minute)},
		{Type: model.Playoff, TypeOrder: 3, LongName: "Match 3", Time: start.Add(20 * time.Minute)},
		{Type: model.Playoff, TypeOrder: 4, LongName: "Match 4", Time: start.Add(35 * time.Minute)},
	}
	for _, match := range matches {
		assert.NoError(t, web.arena.Database.CreateMatch(match))
	}
	break1 := &model.ScheduledBreak{MatchType: model.Playoff, TypeOrderBefore: 2, Time: start.Add(5 * time.Minute), DurationSec: 600, Description: "Break 1"}
	break2 := &model.ScheduledBreak{MatchType: model.Playoff, TypeOrderBefore: 4, Time: start.Add(25 * time.Minute), DurationSec: 600, Description: "Break 2"}
	assert.NoError(t, web.arena.Database.CreateScheduledBreak(break1))
	assert.NoError(t, web.arena.Database.CreateScheduledBreak(break2))

	// Move the first break from after Match 1 to after Match 2, and
	// shorten the one after Match 3.
	response := web.postHttpResponse("/setup/breaks", breakPostRows(
		model.PlayoffBreakSetting{AfterMatchOrder: 2, DurationSec: 1200, Description: "After M2"},
		model.PlayoffBreakSetting{AfterMatchOrder: 3, DurationSec: 300, Description: "Awards"},
	))
	assert.Equal(t, http.StatusSeeOther, response.Code)
	for index, want := range []time.Duration{0, 5 * time.Minute, 30 * time.Minute, 40 * time.Minute} {
		got, err := web.arena.Database.GetMatchById(matches[index].Id)
		assert.NoError(t, err)
		assert.Equal(t, start.Add(want), got.Time)
	}
	breaks, err := web.arena.Database.GetScheduledBreaksByMatchType(model.Playoff)
	assert.NoError(t, err)
	if assert.Len(t, breaks, 2) {
		assert.Equal(t, 3, breaks[0].TypeOrderBefore)
		assert.Equal(t, start.Add(10*time.Minute), breaks[0].Time)
		assert.Equal(t, "After M2", breaks[0].Description)
		assert.Equal(t, 4, breaks[1].TypeOrderBefore)
		assert.Equal(t, start.Add(35*time.Minute), breaks[1].Time)
		assert.Equal(t, "Awards", breaks[1].Description)
	}
}

func TestSetupBreaksRejectsDuplicatesAndStartedMatches(t *testing.T) {
	web := setupTestWeb(t)
	duplicate := breakPostRows(
		model.PlayoffBreakSetting{AfterMatchOrder: 8, DurationSec: 300, Description: "One"},
		model.PlayoffBreakSetting{AfterMatchOrder: 8, DurationSec: 600, Description: "Two"},
	)
	response := web.postHttpResponse("/setup/breaks", duplicate)
	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Contains(t, response.Body.String(), "more than one break")
	assert.Empty(t, web.arena.EventSettings.PlayoffBreaksJSON)

	start := time.Now().Add(-time.Hour)
	previous := &model.Match{Type: model.Playoff, TypeOrder: 1, LongName: "Match 1", Time: start, StartedAt: start}
	next := &model.Match{Type: model.Playoff, TypeOrder: 2, LongName: "Match 2", Time: start.Add(15*time.Minute), StartedAt: start.Add(15*time.Minute)}
	assert.NoError(t, web.arena.Database.CreateMatch(previous))
	assert.NoError(t, web.arena.Database.CreateMatch(next))
	pause := &model.ScheduledBreak{MatchType: model.Playoff, TypeOrderBefore: 2, Time: start.Add(5*time.Minute), DurationSec: 600, Description: "Original"}
	assert.NoError(t, web.arena.Database.CreateScheduledBreak(pause))
	response = web.postHttpResponse("/setup/breaks", breakPostRows(
		model.PlayoffBreakSetting{AfterMatchOrder: 1, DurationSec: 900, Description: "Changed"},
	))
	assert.Equal(t, http.StatusBadRequest, response.Code)
	got, err := web.arena.Database.GetScheduledBreakById(pause.Id)
	assert.NoError(t, err)
	assert.Equal(t, "Original", got.Description)
	assert.Equal(t, 600, got.DurationSec)
}

func TestSetupBreaksRestoreDefaultsAndAllowNoBreaks(t *testing.T) {
	web := setupTestWeb(t)
	response := web.postHttpResponse("/setup/breaks", breakPostRows())
	assert.Equal(t, http.StatusSeeOther, response.Code)
	assert.Empty(t, web.arena.PlayoffTournament.ConfiguredBreaks())
	assert.NotEmpty(t, web.arena.EventSettings.PlayoffBreaksJSON)

	response = web.postHttpResponse("/setup/breaks", "action=defaults")
	assert.Equal(t, http.StatusSeeOther, response.Code)
	assert.Equal(t, 6, len(web.arena.PlayoffTournament.ConfiguredBreaks()))
	assert.Equal(t, "", web.arena.EventSettings.PlayoffBreaksJSON)
}

func TestSetupBreaksRejectsInvalidPositionsAndDurations(t *testing.T) {
	web := setupTestWeb(t)
	response := web.postHttpResponse("/setup/breaks", breakPostRows(
		model.PlayoffBreakSetting{AfterMatchOrder: 999, DurationSec: 600, Description: "Bad"},
	))
	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Contains(t, response.Body.String(), "not a valid break position")

	post := url.Values{
		"afterMatchOrder": {"8"}, "durationMinutes": {"0"}, "description": {"Bad"},
	}
	response = web.postHttpResponse("/setup/breaks", post.Encode())
	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Contains(t, response.Body.String(), "duration")
}
