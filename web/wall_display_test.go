// Copyright 2023 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)

package web

import (
	"github.com/Team254/cheesy-arena/websocket"
	gorillawebsocket "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestWallDisplay(t *testing.T) {
	web := setupTestWeb(t)

	recorder := web.getHttpResponse("/displays/wall")
	assert.Equal(t, 302, recorder.Code)
	assert.Contains(t, recorder.Header().Get("Location"), "displayId=100")
	assert.Contains(t, recorder.Header().Get("Location"), "background=%23000")
	assert.Contains(t, recorder.Header().Get("Location"), "message=")
	assert.Contains(t, recorder.Header().Get("Location"), "reversed=false")
	assert.Contains(t, recorder.Header().Get("Location"), "topSpacingPx=0")
	assert.Contains(t, recorder.Header().Get("Location"), "zoomFactor=1")
	assert.Contains(t, recorder.Header().Get("Location"), "overlayLocation=bottom")

	recorder = web.getHttpResponse(
		"/displays/wall?displayId=1&background=%23fff&message=hello+there&reversed=true&topSpacingPx=10&zoomFactor=2&overlayLocation=top",
	)
	assert.Equal(t, 200, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Wall Display - Untitled Event - Cheesy Arena")
	assert.Contains(t, recorder.Body.String(), "id=\"audienceInformation\"")
	assert.Contains(t, recorder.Body.String(), "id=\"wallBracket\"")
	assert.Contains(t, recorder.Body.String(), "audience_information.js")
	assert.Contains(t, recorder.Body.String(), "audience_information.css")
}

func TestWallDisplayWebsocket(t *testing.T) {
	web := setupTestWeb(t)

	server, wsUrl := web.startTestServer()
	defer server.Close()
	conn, _, err := gorillawebsocket.DefaultDialer.Dial(wsUrl+"/displays/wall/websocket?displayId=1", nil)
	assert.Nil(t, err)
	defer conn.Close()
	ws := websocket.NewTestWebsocket(conn)

	// Should get a few status updates right after connection.
	readWebsocketType(t, ws, "displayConfiguration")
	readWebsocketType(t, ws, "matchTiming")
	readWebsocketType(t, ws, "audienceDisplayMode")
	readWebsocketType(t, ws, "matchLoad")
	readWebsocketType(t, ws, "matchTime")
	readWebsocketType(t, ws, "realtimeScore")
	// Wall displays now receive score-posted updates to refresh a visible bracket.
	readWebsocketType(t, ws, "scorePosted")

	// Run through a match cycle.
	web.arena.MatchLoadNotifier.Notify()
	readWebsocketType(t, ws, "matchLoad")
	web.arena.AllianceStations["R1"].Bypass = true
	web.arena.AllianceStations["R2"].Bypass = true
	web.arena.AllianceStations["R3"].Bypass = true
	web.arena.AllianceStations["B1"].Bypass = true
	web.arena.AllianceStations["B2"].Bypass = true
	web.arena.AllianceStations["B3"].Bypass = true
	web.arena.StartMatch()
	web.arena.Update()
	web.arena.Update()
	messages := readWebsocketMultiple(t, ws, 3)
	screen, ok := messages["audienceDisplayMode"]
	if assert.True(t, ok) {
		assert.Equal(t, "match", screen)
	}
	_, ok = messages["matchTime"]
	assert.True(t, ok)
	web.arena.RealtimeScoreNotifier.Notify()
	readWebsocketType(t, ws, "realtimeScore")
}

func TestWallDisplayInformationModes(t *testing.T) {
	web := setupTestWeb(t)

	server, wsUrl := web.startTestServer()
	defer server.Close()
	conn, _, err := gorillawebsocket.DefaultDialer.Dial(wsUrl+"/displays/wall/websocket?displayId=2", nil)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	defer conn.Close()
	ws := websocket.NewTestWebsocket(conn)

	// Drain the initial state, including the scorePosted notifier.
	for _, name := range []string{
		"displayConfiguration", "matchTiming", "audienceDisplayMode", "matchLoad",
		"matchTime", "realtimeScore", "scorePosted",
	} {
		readWebsocketType(t, ws, name)
	}

	// Wall screens must receive exactly the same mode events as audience screens.
	for _, mode := range []string{"rankingsFull", "rankingsBar", "bracket", "bracketBar", "match"} {
		web.arena.SetAudienceDisplayMode(mode)
		assert.Equal(t, mode, readWebsocketType(t, ws, "audienceDisplayMode"))
	}
}
