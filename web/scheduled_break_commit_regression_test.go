// Copyright 2026 Freezy Arena contributors.
// Author: Justin Fischer <justin@justfishin99.xyz>

package web

import (
 "github.com/Team254/cheesy-arena/field"
 "github.com/Team254/cheesy-arena/game"
 "github.com/Team254/cheesy-arena/model"
 "github.com/Team254/cheesy-arena/tournament"
 "github.com/stretchr/testify/assert"
 "testing"
 "time"
)

// Reproduces clicking Commit & Post on a match with a scheduled break
// immediately before the next match. Discard Results must not be necessary.
func TestCommitPostTransitionsIntoPendingScheduledBreak(t *testing.T) {
 web := setupTestWeb(t)
 tournament.CreateTestAlliances(web.arena.Database, 8)
 assert.NoError(t, web.arena.CreatePlayoffTournament())
 assert.NoError(t, web.arena.CreatePlayoffMatches(time.Now()))
 matches, err := web.arena.Database.GetMatchesByType(model.Playoff, true)
 assert.NoError(t, err)
 if !assert.GreaterOrEqual(t, len(matches), 9) { return }
 for i := 0; i < 7; i++ {
   matches[i].Status = game.RedWonMatch
   assert.NoError(t, web.arena.Database.UpdateMatch(&matches[i]))
 }
 assert.NoError(t, web.arena.LoadMatch(&matches[7]))
 web.arena.MatchState = field.PostMatch
 err = web.commitPostAndLoadNextMatch()
 if !assert.NoError(t, err) { return }
 assert.Equal(t, 9, web.arena.CurrentMatch.TypeOrder)
 assert.Equal(t, field.PreMatch, web.arena.MatchState)
 if assert.NotNil(t, web.arena.PendingScheduledBreak()) {
    assert.Equal(t, 9, web.arena.PendingScheduledBreak().TypeOrderBefore)
 }
}
