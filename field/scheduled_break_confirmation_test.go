// Copyright 2026 Freezy Arena contributors.
// Author: Justin Fischer <justin@justfishin99.xyz>

package field

import (
	"github.com/Team254/cheesy-arena/game"
	"github.com/Team254/cheesy-arena/model"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestScheduledBreakRequiresFieldClearanceConfirmation(t *testing.T) {
	arena := setupTestArena(t)
	first := &model.Match{Type: model.Playoff, TypeOrder: 1, LongName: "Match 1", Status: game.RedWonMatch}
	next := &model.Match{Type: model.Playoff, TypeOrder: 2, LongName: "Match 2", Status: game.MatchScheduled}
	assert.NoError(t, arena.Database.CreateMatch(first))
	assert.NoError(t, arena.Database.CreateMatch(next))
	pause := &model.ScheduledBreak{
		MatchType: model.Playoff, TypeOrderBefore: 2, DurationSec: 180,
		Description: "Field Clearance Break",
	}
	assert.NoError(t, arena.Database.CreateScheduledBreak(pause))
	assert.NoError(t, arena.LoadMatch(first))
	assert.NoError(t, arena.LoadNextMatch(true))

	if !assert.NotNil(t, arena.PendingScheduledBreak()) {
		return
	}
	assert.Equal(t, pause.Id, arena.PendingScheduledBreak().Id)
	assert.Equal(t, PreMatch, arena.MatchState)
	assert.Contains(t, arena.checkCanStartMatch().Error(), "field-clear confirmation")
	assert.ErrorContains(t, arena.StartAdHocTimeout("Bypass", "", 30), "confirm field clearance")
	assert.ErrorContains(t, arena.LoadTestMatch(), "cannot switch matches")
	assert.ErrorContains(t, arena.ConfirmScheduledBreakStart(first.Id), "no scheduled break")
	assert.ErrorContains(t, arena.ConfirmScheduledBreakStart(0), "no scheduled break")
	assert.Equal(t, PreMatch, arena.MatchState)

	before := time.Now()
	assert.NoError(t, arena.ConfirmScheduledBreakStart(next.Id))
	assert.Equal(t, TimeoutActive, arena.MatchState)
	assert.WithinDuration(t, before, arena.MatchStartTime, time.Second)
	assert.Equal(t, 180, game.MatchTiming.TimeoutDurationSec)
	assert.Nil(t, arena.PendingScheduledBreak())
	assert.Equal(t, "Field Clearance Break", arena.breakDescription)
	assert.Equal(t, "Match 2", arena.breakNextMatchName)
	assert.ErrorContains(t, arena.ConfirmScheduledBreakStart(next.Id), "no scheduled break")
}

func TestNoScheduledBreakDoesNotRequireConfirmation(t *testing.T) {
	arena := setupTestArena(t)
	first := &model.Match{Type: model.Playoff, TypeOrder: 1, LongName: "Match 1", Status: game.RedWonMatch}
	next := &model.Match{Type: model.Playoff, TypeOrder: 2, LongName: "Match 2", Status: game.MatchScheduled}
	assert.NoError(t, arena.Database.CreateMatch(first))
	assert.NoError(t, arena.Database.CreateMatch(next))
	assert.NoError(t, arena.LoadMatch(first))
	assert.NoError(t, arena.LoadNextMatch(true))
	assert.Nil(t, arena.PendingScheduledBreak())
	assert.Equal(t, PreMatch, arena.MatchState)
}
