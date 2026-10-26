// Copyright 2026 Freezy Arena contributors.
// Author: Justin Fischer <justin@justfishin99.xyz>

package web

import (
	"github.com/Team254/cheesy-arena/field"
	"github.com/Team254/cheesy-arena/game"
	"github.com/Team254/cheesy-arena/model"
	"github.com/Team254/cheesy-arena/websocket"
	gorillawebsocket "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMatchPlayScheduledBreakConfirmation(t *testing.T) {
	web := setupTestWeb(t)
	recorder := web.getHttpResponse("/match_play")
	assert.Equal(t, 200, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Confirm Field Clear")
	assert.Contains(t, recorder.Body.String(), "all robots have left")

	first := &model.Match{Type: model.Playoff, TypeOrder: 1, LongName: "Match 1", Status: game.RedWonMatch}
	next := &model.Match{Type: model.Playoff, TypeOrder: 2, LongName: "Match 2", Status: game.MatchScheduled}
	assert.NoError(t, web.arena.Database.CreateMatch(first))
	assert.NoError(t, web.arena.Database.CreateMatch(next))
	pause := &model.ScheduledBreak{
		MatchType: model.Playoff, TypeOrderBefore: 2, DurationSec: 120,
		Description: "Awards Break",
	}
	assert.NoError(t, web.arena.Database.CreateScheduledBreak(pause))
	assert.NoError(t, web.arena.LoadMatch(first))
	assert.NoError(t, web.arena.LoadNextMatch(true))

	server, wsUrl := web.startTestServer()
	defer server.Close()
	conn, _, err := gorillawebsocket.DefaultDialer.Dial(wsUrl+"/match_play/websocket", nil)
	assert.NoError(t, err)
	if conn == nil {
		return
	}
	defer conn.Close()
	ws := websocket.NewTestWebsocket(conn)
	initial := readWebsocketMultiple(t, ws, 10)
	matchLoad := initial["matchLoad"].(map[string]any)
	pending := matchLoad["PendingScheduledBreak"].(map[string]any)
	assert.Equal(t, "Awards Break", pending["Description"])
	assert.Equal(t, float64(120), pending["DurationSec"])

	ws.Write("confirmScheduledBreak", map[string]any{"matchId": float64(first.Id)})
	assert.Contains(t, readWebsocketError(t, ws), "no scheduled break")
	assert.Equal(t, field.PreMatch, web.arena.MatchState)

	ws.Write("confirmScheduledBreak", map[string]any{"matchId": float64(next.Id)})
	received := readWebsocketTypes(t, ws, 5, "matchTiming", "matchLoad", "allianceStationDisplayMode", "arenaStatus")
	assert.Nil(t, received["matchLoad"].(map[string]any)["PendingScheduledBreak"])
	assert.Equal(t, field.TimeoutActive, web.arena.MatchState)
	assert.Nil(t, web.arena.PendingScheduledBreak())

	ws.Write("confirmScheduledBreak", map[string]any{"matchId": float64(next.Id)})
	assert.Contains(t, readWebsocketError(t, ws), "no scheduled break")
}
