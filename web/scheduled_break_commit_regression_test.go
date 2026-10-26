// Copyright 2026 Freezy Arena contributors.
// Author: Justin Fischer <justin@justfishin99.xyz>

package web

import (
	"github.com/Team254/cheesy-arena/field"
	"github.com/Team254/cheesy-arena/model"
	"github.com/Team254/cheesy-arena/tournament"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

// A normal Commit & Post must load the next match and arm its scheduled
// break without requiring Discard Results, even when confirmation is enabled.
func TestCommitPostTransitionsIntoPendingScheduledBreak(t *testing.T) {
	web := setupTestWeb(t)
	tournament.CreateTestAlliances(web.arena.Database, 8)
	assert.NoError(t, web.arena.CreatePlayoffTournament())
	assert.NoError(t, web.arena.CreatePlayoffMatches(time.Now()))
	matches, err := web.arena.Database.GetMatchesByType(model.Playoff, true)
	assert.NoError(t, err)
	if !assert.GreaterOrEqual(t, len(matches), 2) {
		return
	}

	// Create a break immediately after the opening playoff match so that
	// the regression exercises a real tournament without fake prior scores.
	pause := &model.ScheduledBreak{
		MatchType: model.Playoff, TypeOrderBefore: matches[1].TypeOrder,
		DurationSec: 300, Description: "Field Break",
	}
	assert.NoError(t, web.arena.Database.CreateScheduledBreak(pause))
	assert.NoError(t, web.arena.LoadMatch(&matches[0]))
	web.arena.MatchState = field.PostMatch

	err = web.commitPostAndLoadNextMatch()
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, matches[1].Id, web.arena.CurrentMatch.Id)
	assert.Equal(t, field.PreMatch, web.arena.MatchState)
	if assert.NotNil(t, web.arena.PendingScheduledBreak()) {
		assert.Equal(t, pause.Id, web.arena.PendingScheduledBreak().Id)
	}
	assert.ErrorContains(t, web.arena.StartMatch(), "scheduled break")
	assert.NoError(t, web.arena.ConfirmScheduledBreakStart(matches[1].Id))
	assert.Equal(t, field.TimeoutActive, web.arena.MatchState)
}

// A playoff match with an unresolved alliance slot must not panic during
// Commit & Post, even when it has no assigned red or blue playoff alliance.
func TestCommitPostWithUnassignedPlayoffAlliances(t *testing.T) {
	web := setupTestWeb(t)
	tournament.CreateTestAlliances(web.arena.Database, 8)
	assert.NoError(t, web.arena.CreatePlayoffTournament())
	assert.NoError(t, web.arena.CreatePlayoffMatches(time.Now()))
	match, err := web.arena.Database.GetMatchByTypeOrder(model.Playoff, 1)
	assert.NoError(t, err)
	if !assert.NotNil(t, match) {
		return
	}
	match.PlayoffRedAlliance = 0
	match.PlayoffBlueAlliance = 0
	assert.NoError(t, web.arena.Database.UpdateMatch(match))
	assert.NoError(t, web.arena.LoadMatch(match))
	web.arena.MatchState = field.PostMatch

	// The old UpdateAllianceFromMatch dereferenced nil when the ID was zero.
	assert.NoError(t, web.commitPostAndLoadNextMatch())
	assert.Equal(t, 2, web.arena.CurrentMatch.TypeOrder)
	assert.Equal(t, field.PreMatch, web.arena.MatchState)
}
