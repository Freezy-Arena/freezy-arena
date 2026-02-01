// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)

package web

import (
	"testing"
	"time"

	"github.com/Team254/cheesy-arena/field"
	"github.com/Team254/cheesy-arena/game"
	"github.com/Team254/cheesy-arena/websocket"
	gorillawebsocket "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

func TestScoringPanel(t *testing.T) {
	web := setupTestWeb(t)

	recorder := web.getHttpResponse("/panels/scoring/invalidposition")
	assert.Equal(t, 500, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Invalid position")
	recorder = web.getHttpResponse("/panels/scoring/red_near")
	assert.Equal(t, 200, recorder.Code)
	recorder = web.getHttpResponse("/panels/scoring/red_far")
	assert.Equal(t, 200, recorder.Code)
	recorder = web.getHttpResponse("/panels/scoring/blue_near")
	assert.Equal(t, 200, recorder.Code)
	recorder = web.getHttpResponse("/panels/scoring/blue_far")
	assert.Equal(t, 200, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Scoring Panel - Untitled Event - Cheesy Arena")
}

func TestScoringPanelWebsocket(t *testing.T) {
	web := setupTestWeb(t)

	server, wsUrl := web.startTestServer()
	defer server.Close()
	_, _, err := gorillawebsocket.DefaultDialer.Dial(wsUrl+"/panels/scoring/blorpy/websocket", nil)
	assert.NotNil(t, err)
	redConn, _, err := gorillawebsocket.DefaultDialer.Dial(wsUrl+"/panels/scoring/red_near/websocket", nil)
	assert.Nil(t, err)
	defer redConn.Close()
	redWs := websocket.NewTestWebsocket(redConn)
	assert.Equal(t, 1, web.arena.ScoringPanelRegistry.GetNumPanels("red_near"))
	assert.Equal(t, 0, web.arena.ScoringPanelRegistry.GetNumPanels("blue_near"))
	blueConn, _, err := gorillawebsocket.DefaultDialer.Dial(wsUrl+"/panels/scoring/blue_near/websocket", nil)
	assert.Nil(t, err)
	defer blueConn.Close()
	blueWs := websocket.NewTestWebsocket(blueConn)
	assert.Equal(t, 1, web.arena.ScoringPanelRegistry.GetNumPanels("red_near"))
	assert.Equal(t, 1, web.arena.ScoringPanelRegistry.GetNumPanels("blue_near"))

	// Should get a few status updates right after connection.
	readWebsocketType(t, redWs, "resetLocalState")
	readWebsocketType(t, redWs, "matchLoad")
	readWebsocketType(t, redWs, "matchTime")
	readWebsocketType(t, redWs, "realtimeScore")
	readWebsocketType(t, blueWs, "resetLocalState")
	readWebsocketType(t, blueWs, "matchLoad")
	readWebsocketType(t, blueWs, "matchTime")
	readWebsocketType(t, blueWs, "realtimeScore")

	// Send some autonomous period scoring commands.
	assert.Equal(t, [3]bool{false, false, false}, web.arena.RedRealtimeScore.CurrentScore.LeaveStatuses)
	leaveData := struct {
		TeamPosition int
	}{}
	web.arena.MatchState = field.AutoPeriod
	leaveData.TeamPosition = 1
	redWs.Write("leave", leaveData)
	leaveData.TeamPosition = 3
	redWs.Write("leave", leaveData)
	for i := 0; i < 2; i++ {
		readWebsocketType(t, redWs, "realtimeScore")
		readWebsocketType(t, blueWs, "realtimeScore")
	}
	assert.Equal(t, [3]bool{true, false, true}, web.arena.RedRealtimeScore.CurrentScore.LeaveStatuses)
	redWs.Write("leave", leaveData)
	readWebsocketType(t, redWs, "realtimeScore")
	readWebsocketType(t, blueWs, "realtimeScore")
	assert.Equal(t, [3]bool{true, false, false}, web.arena.RedRealtimeScore.CurrentScore.LeaveStatuses)

	// Send some fuel scoring commands
	fuelData := struct {
		Adjustment int
	}{}
	assert.Equal(t, 0, web.arena.RedRealtimeScore.CurrentScore.Fuel)
	assert.Equal(t, 0, web.arena.BlueRealtimeScore.CurrentScore.Fuel)
	fuelData.Adjustment = 1
	blueWs.Write("fuel", fuelData)
	blueWs.Write("fuel", fuelData)
	blueWs.Write("fuel", fuelData)
	fuelData.Adjustment = -1
	blueWs.Write("fuel", fuelData)
	blueWs.Write("fuel", fuelData)
	fuelData.Adjustment = 1
	blueWs.Write("fuel", fuelData)
	for i := 0; i < 6; i++ {
		readWebsocketType(t, redWs, "realtimeScore")
		readWebsocketType(t, blueWs, "realtimeScore")
	}
	fuelData.Adjustment = 1
	redWs.Write("fuel", fuelData)
	redWs.Write("fuel", fuelData)
	redWs.Write("fuel", fuelData)
	fuelData.Adjustment = -1
	redWs.Write("fuel", fuelData)
	for i := 0; i < 4; i++ {
		readWebsocketType(t, redWs, "realtimeScore")
		readWebsocketType(t, blueWs, "realtimeScore")
	}
	assert.Equal(t, 2, web.arena.RedRealtimeScore.CurrentScore.Fuel)
	assert.Equal(t, 2, web.arena.BlueRealtimeScore.CurrentScore.Fuel)

	// Send some endgame scoring commands
	endgameData := struct {
		TeamPosition  int
		EndgameStatus int
	}{}
	assert.Equal(
		t,
		[3]game.EndgameStatus{game.EndgameNone, game.EndgameNone, game.EndgameNone},
		web.arena.RedRealtimeScore.CurrentScore.EndgameStatuses,
	)
	assert.Equal(
		t,
		[3]game.EndgameStatus{game.EndgameNone, game.EndgameNone, game.EndgameNone},
		web.arena.BlueRealtimeScore.CurrentScore.EndgameStatuses,
	)
	endgameData.TeamPosition = 1
	endgameData.EndgameStatus = 2
	redWs.Write("endgame", endgameData)
	endgameData.TeamPosition = 2
	endgameData.EndgameStatus = 3
	blueWs.Write("endgame", endgameData)
	endgameData.TeamPosition = 3
	endgameData.EndgameStatus = 1
	blueWs.Write("endgame", endgameData)
	endgameData.TeamPosition = 3
	endgameData.EndgameStatus = 1
	redWs.Write("endgame", endgameData)
	endgameData.TeamPosition = 3
	endgameData.EndgameStatus = 3
	redWs.Write("endgame", endgameData)
	endgameData.TeamPosition = 2
	endgameData.EndgameStatus = 0
	redWs.Write("endgame", endgameData)
	for i := 0; i < 6; i++ {
		readWebsocketType(t, redWs, "realtimeScore")
		readWebsocketType(t, blueWs, "realtimeScore")
	}
	assert.Equal(
		t,
		[3]game.EndgameStatus{game.EndgameShallowCage, game.EndgameNone, game.EndgameDeepCage},
		web.arena.RedRealtimeScore.CurrentScore.EndgameStatuses,
	)
	assert.Equal(
		t,
		[3]game.EndgameStatus{game.EndgameNone, game.EndgameDeepCage, game.EndgameParked},
		web.arena.BlueRealtimeScore.CurrentScore.EndgameStatuses,
	)

	// Test that some invalid commands do nothing and don't result in score change notifications.
	redWs.Write("invalid", nil)
	leaveData.TeamPosition = 0
	redWs.Write("leave", leaveData)
	endgameData.TeamPosition = 1
	endgameData.EndgameStatus = 4
	blueWs.Write("endgame", endgameData)

	// Test committing logic.
	redWs.Write("commitMatch", nil)
	readWebsocketType(t, redWs, "error")
	blueWs.Write("commitMatch", nil)
	readWebsocketType(t, blueWs, "error")
	assert.Equal(t, 0, web.arena.ScoringPanelRegistry.GetNumScoreCommitted("red_near"))
	assert.Equal(t, 0, web.arena.ScoringPanelRegistry.GetNumScoreCommitted("blue_near"))
	web.arena.MatchState = field.PostMatch
	redWs.Write("commitMatch", nil)
	blueWs.Write("commitMatch", nil)
	time.Sleep(time.Millisecond * 10) // Allow some time for the commands to be processed.
	assert.Equal(t, 1, web.arena.ScoringPanelRegistry.GetNumScoreCommitted("red_near"))
	assert.Equal(t, 1, web.arena.ScoringPanelRegistry.GetNumScoreCommitted("blue_near"))

	// Load another match to reset the results.
	web.arena.ResetMatch()
	web.arena.LoadTestMatch()
	readWebsocketType(t, redWs, "matchLoad")
	readWebsocketType(t, redWs, "realtimeScore")
	readWebsocketType(t, blueWs, "matchLoad")
	readWebsocketType(t, blueWs, "realtimeScore")
	assert.Equal(t, field.NewRealtimeScore(), web.arena.RedRealtimeScore)
	assert.Equal(t, field.NewRealtimeScore(), web.arena.BlueRealtimeScore)
	assert.Equal(t, 0, web.arena.ScoringPanelRegistry.GetNumScoreCommitted("red_near"))
	assert.Equal(t, 0, web.arena.ScoringPanelRegistry.GetNumScoreCommitted("blue_near"))
}
