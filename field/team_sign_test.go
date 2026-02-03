// Copyright 2024 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)

package field

import (
	"testing"

	"github.com/Team254/cheesy-arena/game"
	"github.com/Team254/cheesy-arena/model"
	"github.com/stretchr/testify/assert"
)

func TestTeamSign_GenerateInMatchRearText(t *testing.T) {
	arena := setupTestArena(t)
	arena.RedRealtimeScore.CurrentScore = *game.TestScore1()
	arena.BlueRealtimeScore.CurrentScore = *game.TestScore2()

	assert.Equal(t, "01:23 R013-B046", generateInMatchTeamRearText(arena, true, "01:23"))
	assert.Equal(t, "01:23 B046-R013", generateInMatchTeamRearText(arena, false, "01:23"))
	assert.Equal(t, "Fuel: 7", generateInMatchTimerRearText(arena, true))
	assert.Equal(t, "Fuel: 9", generateInMatchTimerRearText(arena, false))

	arena.BlueRealtimeScore.CurrentScore.Fuel = 15
	assert.Equal(t, "00:59 R013-B052", generateInMatchTeamRearText(arena, true, "00:59"))
	assert.Equal(t, "00:59 B052-R013", generateInMatchTeamRearText(arena, false, "00:59"))
	assert.Equal(t, "Fuel: 7", generateInMatchTimerRearText(arena, true))
	assert.Equal(t, "Fuel: 15", generateInMatchTimerRearText(arena, false))

	// Check that formatting is correct for playoff matches.
	arena.CurrentMatch.Type = model.Playoff
	assert.Equal(t, "00:45 R013-B052", generateInMatchTeamRearText(arena, true, "00:45"))
	assert.Equal(t, "00:45 B052-R013", generateInMatchTeamRearText(arena, false, "00:45"))
}

func TestTeamSign_Timer(t *testing.T) {
	arena := setupTestArena(t)
	sign := TeamSign{isTimer: true}

	// Should do nothing if no address is set.
	sign.update(arena, nil, true, "12:34", "Rear Text")
	assert.Equal(t, [128]byte{}, sign.packetData)

	sign.SetId(1)
	sign.update(arena, nil, true, "12:34", "Rear Text")
	assert.Equal(t, "12:34", sign.frontText)
}
