// Copyright 2018 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)

package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Team254/cheesy-arena/field"
	"github.com/Team254/cheesy-arena/game"
	"github.com/stretchr/testify/assert"
)

func (web *Web) postJsonHttpResponse(path string, body string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	web.newHandler().ServeHTTP(recorder, req)
	return recorder
}

func TestFieldStackLightGetHandler(t *testing.T) {
	web := setupTestWeb(t)

	recorder := web.getHttpResponse("/api/freezy/field_stack_light")
	assert.Equal(t, 200, recorder.Code)

	var stackLight fieldStackLight
	err := json.Unmarshal([]byte(recorder.Body.String()), &stackLight)
	assert.Nil(t, err)
}

func TestFieldStackLightGetHandler_InvalidMethod(t *testing.T) {
	web := setupTestWeb(t)

	recorder := web.postHttpResponse("/api/freezy/field_stack_light", "")
	assert.Equal(t, 405, recorder.Code)
}

func TestTeamStackLightGetHandler(t *testing.T) {
	web := setupTestWeb(t)

	recorder := web.getHttpResponse("/api/freezy/team_stack_light")
	assert.Equal(t, 200, recorder.Code)

	var stackLights allStackLights
	err := json.Unmarshal([]byte(recorder.Body.String()), &stackLights)
	assert.Nil(t, err)
}

func TestTeamStackLightGetHandler_InvalidMethod(t *testing.T) {
	web := setupTestWeb(t)

	recorder := web.postHttpResponse("/api/freezy/team_stack_light", "")
	assert.Equal(t, 405, recorder.Code)
}

func TestTeamHubStateGetHandler_PreMatch(t *testing.T) {
	web := setupTestWeb(t)

	// PreMatch state without FieldVolunteers
	web.arena.MatchState = field.PreMatch
	web.arena.FieldVolunteers = false

	recorder := web.getHttpResponse("/api/freezy/hub_status")
	assert.Equal(t, 200, recorder.Code)

	var states hubStates
	err := json.Unmarshal([]byte(recorder.Body.String()), &states)
	assert.Nil(t, err)
	assert.Equal(t, "purple", states.Red.Color)
	assert.Equal(t, "purple", states.Blue.Color)
	assert.False(t, states.Red.Blink)
	assert.False(t, states.Blue.Blink)
}

func TestTeamHubStateGetHandler_PreMatchWithVolunteers(t *testing.T) {
	web := setupTestWeb(t)

	// PreMatch state with FieldVolunteers
	web.arena.MatchState = field.PreMatch
	web.arena.FieldVolunteers = true

	recorder := web.getHttpResponse("/api/freezy/hub_status")
	assert.Equal(t, 200, recorder.Code)

	var states hubStates
	err := json.Unmarshal([]byte(recorder.Body.String()), &states)
	assert.Nil(t, err)
	assert.Equal(t, "green", states.Red.Color)
	assert.Equal(t, "green", states.Blue.Color)
	assert.False(t, states.Red.Blink)
	assert.False(t, states.Blue.Blink)
}

func TestTeamHubStateGetHandler_DuringMatch_BothHubsActive(t *testing.T) {
	web := setupTestWeb(t)

	// During match with both hubs active
	// Note: The handler checks (1<<1) for red and (1<<2) for blue, so we use those values
	web.arena.MatchState = field.AutoPeriod
	web.arena.HubsActive = (1 << 1) | (1 << 2) // Red (bit 1) and Blue (bit 2)
	web.arena.MatchStartTime = time.Now()

	recorder := web.getHttpResponse("/api/freezy/hub_status")
	assert.Equal(t, 200, recorder.Code)

	var states hubStates
	err := json.Unmarshal([]byte(recorder.Body.String()), &states)
	assert.Nil(t, err)
	assert.Equal(t, "red", states.Red.Color)
	assert.Equal(t, "blue", states.Blue.Color)
	assert.False(t, states.Red.Blink)
	assert.False(t, states.Blue.Blink)
}

func TestTeamHubStateGetHandler_DuringMatch_OnlyRedActive(t *testing.T) {
	web := setupTestWeb(t)

	// During match with only red hub active
	// Note: The handler checks (1<<1) for red, so we use that value
	web.arena.MatchState = field.Shift1
	web.arena.HubsActive = (1 << 1) // Red (bit 1)
	web.arena.MatchStartTime = time.Now()

	recorder := web.getHttpResponse("/api/freezy/hub_status")
	assert.Equal(t, 200, recorder.Code)

	var states hubStates
	err := json.Unmarshal([]byte(recorder.Body.String()), &states)
	assert.Nil(t, err)
	assert.Equal(t, "red", states.Red.Color)
	assert.Equal(t, "black", states.Blue.Color)
}

func TestTeamHubStateGetHandler_DuringMatch_OnlyBlueActive(t *testing.T) {
	web := setupTestWeb(t)

	// During match with only blue hub active
	// Note: The handler checks (1<<2) for blue, so we use that value
	web.arena.MatchState = field.Shift2
	web.arena.HubsActive = (1 << 2) // Blue (bit 2)
	web.arena.MatchStartTime = time.Now()

	recorder := web.getHttpResponse("/api/freezy/hub_status")
	assert.Equal(t, 200, recorder.Code)

	var states hubStates
	err := json.Unmarshal([]byte(recorder.Body.String()), &states)
	assert.Nil(t, err)
	assert.Equal(t, "black", states.Red.Color)
	assert.Equal(t, "blue", states.Blue.Color)
}

func TestTeamHubStateGetHandler_BlinkWarning_Shift1(t *testing.T) {
	web := setupTestWeb(t)

	// Set up timing so we're within 3 seconds of Shift1 ending
	// Shift1 ends at: WarmupDurationSec + AutoDurationSec + TransitionShiftDurationSec + AllianceShiftDurationSec
	shift1EndSec := game.GetDurationToShiftEnd(1).Seconds()
	web.arena.MatchState = field.Shift1
	web.arena.HubsActive = (1 << 1) // Red (bit 1)
	// Set match start time so that current time is 2 seconds before shift1 ends
	web.arena.MatchStartTime = time.Now().Add(-time.Duration(shift1EndSec-2) * time.Second)

	recorder := web.getHttpResponse("/api/freezy/hub_status")
	assert.Equal(t, 200, recorder.Code)

	var states hubStates
	err := json.Unmarshal([]byte(recorder.Body.String()), &states)
	assert.Nil(t, err)
	assert.Equal(t, "red", states.Red.Color)
	assert.True(t, states.Red.Blink, "Red hub should blink when within 3 seconds of state end")
}

func TestTeamHubStateGetHandler_BlinkWarning_TransitionShift(t *testing.T) {
	web := setupTestWeb(t)

	// Set up timing so we're within 3 seconds of TransitionShift ending
	transitionEndSec := game.GetDurationToShift1Start().Seconds()
	web.arena.MatchState = field.TransitionShift
	web.arena.HubsActive = (1 << 1) | (1 << 2) // Red (bit 1) and Blue (bit 2)
	// Set match start time so that current time is 2 seconds before transition ends
	web.arena.MatchStartTime = time.Now().Add(-time.Duration(transitionEndSec-2) * time.Second)

	recorder := web.getHttpResponse("/api/freezy/hub_status")
	assert.Equal(t, 200, recorder.Code)

	var states hubStates
	err := json.Unmarshal([]byte(recorder.Body.String()), &states)
	assert.Nil(t, err)
	assert.True(t, states.Red.Blink, "Red hub should blink when within 3 seconds of TransitionShift end")
	assert.True(t, states.Blue.Blink, "Blue hub should blink when within 3 seconds of TransitionShift end")
}

func TestTeamHubStateGetHandler_BlinkWarning_EndGame(t *testing.T) {
	web := setupTestWeb(t)

	// Set up timing so we're within 3 seconds of EndGame ending
	endGameEndSec := game.GetDurationToTeleopEnd().Seconds()
	web.arena.MatchState = field.EndGame
	web.arena.HubsActive = (1 << 1) | (1 << 2) // Red (bit 1) and Blue (bit 2)
	// Set match start time so that current time is 2 seconds before endgame ends
	web.arena.MatchStartTime = time.Now().Add(-time.Duration(endGameEndSec-2) * time.Second)

	recorder := web.getHttpResponse("/api/freezy/hub_status")
	assert.Equal(t, 200, recorder.Code)

	var states hubStates
	err := json.Unmarshal([]byte(recorder.Body.String()), &states)
	assert.Nil(t, err)
	assert.True(t, states.Red.Blink, "Red hub should blink when within 3 seconds of EndGame end")
	assert.True(t, states.Blue.Blink, "Blue hub should blink when within 3 seconds of EndGame end")
}

func TestTeamHubStateGetHandler_NoBlink_WhenNotNearStateEnd(t *testing.T) {
	web := setupTestWeb(t)

	// Set up timing so we're well before the state ends (more than 3 seconds)
	web.arena.MatchState = field.Shift1
	web.arena.HubsActive = (1 << 1) // Red (bit 1)
	// Set match start time to just after Shift1 starts (10 seconds into shift, well before end)
	shift1StartSec := game.GetDurationToShift1Start().Seconds()
	web.arena.MatchStartTime = time.Now().Add(-time.Duration(shift1StartSec+10) * time.Second)

	recorder := web.getHttpResponse("/api/freezy/hub_status")
	assert.Equal(t, 200, recorder.Code)

	var states hubStates
	err := json.Unmarshal([]byte(recorder.Body.String()), &states)
	assert.Nil(t, err)
	assert.Equal(t, "red", states.Red.Color)
	assert.False(t, states.Red.Blink, "Red hub should not blink when more than 3 seconds from state end")
}

func TestTeamHubStateGetHandler_InvalidMethod(t *testing.T) {
	web := setupTestWeb(t)

	recorder := web.postHttpResponse("/api/freezy/hub_status", "")
	assert.Equal(t, 405, recorder.Code)
}

func TestGetAllPlcCoilsGetHandler(t *testing.T) {
	web := setupTestWeb(t)

	recorder := web.getHttpResponse("/api/freezy/alternateIO/PLC_Coils")
	assert.Equal(t, 200, recorder.Code)

	var coilsMap map[string]bool
	err := json.Unmarshal([]byte(recorder.Body.String()), &coilsMap)
	assert.Nil(t, err)
}

func TestGetAllPlcCoilsGetHandler_InvalidMethod(t *testing.T) {
	web := setupTestWeb(t)

	recorder := web.postHttpResponse("/api/freezy/alternateIO/PLC_Coils", "")
	assert.Equal(t, 405, recorder.Code)
}

func TestEStopStatePostHandler(t *testing.T) {
	web := setupTestWeb(t)

	body := `[{"channel": 1, "state": true}]`
	recorder := web.postJsonHttpResponse("/api/freezy/eStopState", body)
	assert.Equal(t, 200, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "eStop state updated successfully")
}

func TestEStopStatePostHandler_MultipleChannels(t *testing.T) {
	web := setupTestWeb(t)

	body := `[{"channel": 1, "state": true}, {"channel": 2, "state": false}]`
	recorder := web.postJsonHttpResponse("/api/freezy/eStopState", body)
	assert.Equal(t, 200, recorder.Code)
}

func TestEStopStatePostHandler_InvalidPayload(t *testing.T) {
	web := setupTestWeb(t)

	recorder := web.postJsonHttpResponse("/api/freezy/eStopState", "invalid json")
	assert.Equal(t, 400, recorder.Code)
}

func TestIncrementElementPostHandler_ProcessorAlgae(t *testing.T) {
	web := setupTestWeb(t)

	initialCount := web.arena.RedRealtimeScore.CurrentScore.ProcessorAlgae
	body := `{"alliance": "red", "element": "ProcessorAlgae"}`
	recorder := web.postJsonHttpResponse("/freezy/alternateio/increment", body)
	assert.Equal(t, 200, recorder.Code)
	assert.Equal(t, initialCount+1, web.arena.RedRealtimeScore.CurrentScore.ProcessorAlgae)
}

func TestIncrementElementPostHandler_Barge(t *testing.T) {
	web := setupTestWeb(t)

	initialCount := web.arena.BlueRealtimeScore.CurrentScore.BargeAlgae
	body := `{"alliance": "blue", "element": "Barge"}`
	recorder := web.postJsonHttpResponse("/freezy/alternateio/increment", body)
	assert.Equal(t, 200, recorder.Code)
	assert.Equal(t, initialCount+1, web.arena.BlueRealtimeScore.CurrentScore.BargeAlgae)
}

func TestIncrementElementPostHandler_InvalidAlliance(t *testing.T) {
	web := setupTestWeb(t)

	body := `{"alliance": "green", "element": "ProcessorAlgae"}`
	recorder := web.postJsonHttpResponse("/freezy/alternateio/increment", body)
	assert.Equal(t, 400, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Unknown alliance")
}

func TestIncrementElementPostHandler_InvalidElement(t *testing.T) {
	web := setupTestWeb(t)

	body := `{"alliance": "red", "element": "InvalidElement"}`
	recorder := web.postJsonHttpResponse("/freezy/alternateio/increment", body)
	assert.Equal(t, 400, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Unknown element")
}

func TestIncrementElementPostHandler_AllianceVariants(t *testing.T) {
	web := setupTestWeb(t)

	// Test "r" variant for red
	initialCount := web.arena.RedRealtimeScore.CurrentScore.ProcessorAlgae
	body := `{"alliance": "r", "element": "ProcessorAlgae"}`
	recorder := web.postJsonHttpResponse("/freezy/alternateio/increment", body)
	assert.Equal(t, 200, recorder.Code)
	assert.Equal(t, initialCount+1, web.arena.RedRealtimeScore.CurrentScore.ProcessorAlgae)

	// Test "b" variant for blue
	initialCount = web.arena.BlueRealtimeScore.CurrentScore.ProcessorAlgae
	body = `{"alliance": "b", "element": "ProcessorAlgae"}`
	recorder = web.postJsonHttpResponse("/freezy/alternateio/increment", body)
	assert.Equal(t, 200, recorder.Code)
	assert.Equal(t, initialCount+1, web.arena.BlueRealtimeScore.CurrentScore.ProcessorAlgae)
}

func TestIncrementElementPostHandler_InvalidPayload(t *testing.T) {
	web := setupTestWeb(t)

	recorder := web.postJsonHttpResponse("/freezy/alternateio/increment", "invalid json")
	assert.Equal(t, 400, recorder.Code)
}

func TestStartMatchPostHandler(t *testing.T) {
	web := setupTestWeb(t)

	recorder := web.postJsonHttpResponse("/api/freezy/startMatch", "")
	assert.Equal(t, 200, recorder.Code)
}

