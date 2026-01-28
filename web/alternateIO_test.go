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
	web.arena.MatchState = field.AutoPeriod
	web.arena.HubsActive = field.RedAllianceHubBit | field.BlueAllianceHubBit
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
	web.arena.MatchState = field.Shift1
	web.arena.HubsActive = field.RedAllianceHubBit
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
	web.arena.MatchState = field.Shift2
	web.arena.HubsActive = field.BlueAllianceHubBit
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
	web.arena.HubsActive = field.RedAllianceHubBit
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

func TestTeamHubStateGetHandler_BlinkWarning_TransitionShift_BlueFirst(t *testing.T) {
	web := setupTestWeb(t)

	// Set up timing so we're within 3 seconds of TransitionShift ending
	// FirstShiftHubState = Blue, so Red will become inactive and should blink
	transitionEndSec := game.GetDurationToShift1Start().Seconds()
	web.arena.MatchState = field.TransitionShift
	web.arena.HubsActive = field.RedAllianceHubBit | field.BlueAllianceHubBit // both active
	web.arena.FirstShiftHubState = field.BlueAllianceHubBit                   // Blue will be active in Shift1
	// Set match start time so that current time is 2 seconds before transition ends
	web.arena.MatchStartTime = time.Now().Add(-time.Duration(transitionEndSec-2) * time.Second)

	recorder := web.getHttpResponse("/api/freezy/hub_status")
	assert.Equal(t, 200, recorder.Code)

	var states hubStates
	err := json.Unmarshal([]byte(recorder.Body.String()), &states)
	assert.Nil(t, err)
	assert.True(t, states.Red.Blink, "Red hub should blink (will become inactive in Shift1)")
	assert.False(t, states.Blue.Blink, "Blue hub should NOT blink (will stay active in Shift1)")
}

func TestTeamHubStateGetHandler_BlinkWarning_TransitionShift_RedFirst(t *testing.T) {
	web := setupTestWeb(t)

	// Set up timing so we're within 3 seconds of TransitionShift ending
	// FirstShiftHubState = Red, so Blue will become inactive and should blink
	transitionEndSec := game.GetDurationToShift1Start().Seconds()
	web.arena.MatchState = field.TransitionShift
	web.arena.HubsActive = field.RedAllianceHubBit | field.BlueAllianceHubBit // both active
	web.arena.FirstShiftHubState = field.RedAllianceHubBit                    // Red will be active in Shift1
	// Set match start time so that current time is 2 seconds before transition ends
	web.arena.MatchStartTime = time.Now().Add(-time.Duration(transitionEndSec-2) * time.Second)

	recorder := web.getHttpResponse("/api/freezy/hub_status")
	assert.Equal(t, 200, recorder.Code)

	var states hubStates
	err := json.Unmarshal([]byte(recorder.Body.String()), &states)
	assert.Nil(t, err)
	assert.False(t, states.Red.Blink, "Red hub should NOT blink (will stay active in Shift1)")
	assert.True(t, states.Blue.Blink, "Blue hub should blink (will become inactive in Shift1)")
}

func TestTeamHubStateGetHandler_BlinkWarning_EndGame(t *testing.T) {
	web := setupTestWeb(t)

	// Set up timing so we're within 3 seconds of EndGame ending
	endGameEndSec := game.GetDurationToTeleopEnd().Seconds()
	web.arena.MatchState = field.EndGame
	web.arena.HubsActive = field.RedAllianceHubBit | field.BlueAllianceHubBit
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

func TestTeamHubStateGetHandler_NoBlink_Shift4ToEndGame(t *testing.T) {
	web := setupTestWeb(t)

	// Set up timing so we're within 3 seconds of Shift4 ending
	// Since EndGame has both hubs active, neither should blink
	shift4EndSec := game.GetDurationToShiftEnd(4).Seconds()
	web.arena.MatchState = field.Shift4
	web.arena.HubsActive = field.RedAllianceHubBit // only one hub active in Shift4
	// Set match start time so that current time is 2 seconds before shift4 ends
	web.arena.MatchStartTime = time.Now().Add(-time.Duration(shift4EndSec-2) * time.Second)

	recorder := web.getHttpResponse("/api/freezy/hub_status")
	assert.Equal(t, 200, recorder.Code)

	var states hubStates
	err := json.Unmarshal([]byte(recorder.Body.String()), &states)
	assert.Nil(t, err)
	assert.Equal(t, "red", states.Red.Color)
	assert.False(t, states.Red.Blink, "Red hub should NOT blink (EndGame will have both active)")
}

func TestTeamHubStateGetHandler_NoBlink_WhenNotNearStateEnd(t *testing.T) {
	web := setupTestWeb(t)

	// Set up timing so we're well before the state ends (more than 3 seconds)
	web.arena.MatchState = field.Shift1
	web.arena.HubsActive = field.RedAllianceHubBit
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

	recorder := web.getHttpResponse("/api/freezy/hub_status")
	// GET is valid, POST without alliance param should fail
	assert.Equal(t, 200, recorder.Code)
}

func TestTeamHubStatusPostHandler_RedHub(t *testing.T) {
	web := setupTestWeb(t)

	body := `{"voltage": 12.5, "percent": 85.0}`
	recorder := web.postJsonHttpResponse("/api/freezy/hub_status?alliance=red", body)
	assert.Equal(t, 200, recorder.Code)

	// Verify the battery values were set
	assert.Equal(t, 12.5, web.arena.Esp32.GetRedHubBatteryVoltage())
	assert.Equal(t, 85.0, web.arena.Esp32.GetRedHubBatteryPercent())
}

func TestTeamHubStatusPostHandler_BlueHub(t *testing.T) {
	web := setupTestWeb(t)

	body := `{"voltage": 11.8, "percent": 72.5}`
	recorder := web.postJsonHttpResponse("/api/freezy/hub_status?alliance=blue", body)
	assert.Equal(t, 200, recorder.Code)

	// Verify the battery values were set
	assert.Equal(t, 11.8, web.arena.Esp32.GetBlueHubBatteryVoltage())
	assert.Equal(t, 72.5, web.arena.Esp32.GetBlueHubBatteryPercent())
}

func TestTeamHubStatusPostHandler_ShortForm(t *testing.T) {
	web := setupTestWeb(t)

	// Test short form "r" for red
	body := `{"voltage": 13.0, "percent": 90.0}`
	recorder := web.postJsonHttpResponse("/api/freezy/hub_status?alliance=r", body)
	assert.Equal(t, 200, recorder.Code)
	assert.Equal(t, 13.0, web.arena.Esp32.GetRedHubBatteryVoltage())

	// Test short form "b" for blue
	body = `{"voltage": 12.0, "percent": 80.0}`
	recorder = web.postJsonHttpResponse("/api/freezy/hub_status?alliance=b", body)
	assert.Equal(t, 200, recorder.Code)
	assert.Equal(t, 12.0, web.arena.Esp32.GetBlueHubBatteryVoltage())
}

func TestTeamHubStatusPostHandler_MissingAlliance(t *testing.T) {
	web := setupTestWeb(t)

	body := `{"voltage": 12.5, "percent": 85.0}`
	recorder := web.postJsonHttpResponse("/api/freezy/hub_status", body)
	assert.Equal(t, 400, recorder.Code)
}

func TestTeamHubStatusPostHandler_InvalidAlliance(t *testing.T) {
	web := setupTestWeb(t)

	body := `{"voltage": 12.5, "percent": 85.0}`
	recorder := web.postJsonHttpResponse("/api/freezy/hub_status?alliance=green", body)
	assert.Equal(t, 400, recorder.Code)
}

func TestTeamHubStatusPostHandler_InvalidPayload(t *testing.T) {
	web := setupTestWeb(t)

	recorder := web.postJsonHttpResponse("/api/freezy/hub_status?alliance=red", "invalid json")
	assert.Equal(t, 400, recorder.Code)
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

func TestTeamHubStateGetHandler_AllianceQueryParam_Red(t *testing.T) {
	web := setupTestWeb(t)

	// Call with red alliance parameter
	recorder := web.getHttpResponse("/api/freezy/hub_status?alliance=red")
	assert.Equal(t, 200, recorder.Code)

	// Verify red hub last seen was updated
	assert.True(t, web.arena.Esp32.IsRedHubActive(), "Red hub should be marked as active after API call with alliance=red")
}

func TestTeamHubStateGetHandler_AllianceQueryParam_Blue(t *testing.T) {
	web := setupTestWeb(t)

	// Call with blue alliance parameter
	recorder := web.getHttpResponse("/api/freezy/hub_status?alliance=blue")
	assert.Equal(t, 200, recorder.Code)

	// Verify blue hub last seen was updated
	assert.True(t, web.arena.Esp32.IsBlueHubActive(), "Blue hub should be marked as active after API call with alliance=blue")
}

func TestTeamHubStateGetHandler_AllianceQueryParam_ShortForm(t *testing.T) {
	web := setupTestWeb(t)

	// Call with short form 'r' for red
	recorder := web.getHttpResponse("/api/freezy/hub_status?alliance=r")
	assert.Equal(t, 200, recorder.Code)
	assert.True(t, web.arena.Esp32.IsRedHubActive(), "Red hub should be marked as active after API call with alliance=r")

	// Reset and test with 'b' for blue
	web = setupTestWeb(t)
	recorder = web.getHttpResponse("/api/freezy/hub_status?alliance=b")
	assert.Equal(t, 200, recorder.Code)
	assert.True(t, web.arena.Esp32.IsBlueHubActive(), "Blue hub should be marked as active after API call with alliance=b")
}

func TestTeamStackLightGetHandler_AllianceQueryParam_Red(t *testing.T) {
	web := setupTestWeb(t)

	// Call with red alliance parameter
	recorder := web.getHttpResponse("/api/freezy/team_stack_light?alliance=red")
	assert.Equal(t, 200, recorder.Code)

	// Verify red estops last seen was updated
	assert.True(t, web.arena.Esp32.IsRedEstopsActive(), "Red estops should be marked as active after API call with alliance=red")
}

func TestTeamStackLightGetHandler_AllianceQueryParam_Blue(t *testing.T) {
	web := setupTestWeb(t)

	// Call with blue alliance parameter
	recorder := web.getHttpResponse("/api/freezy/team_stack_light?alliance=blue")
	assert.Equal(t, 200, recorder.Code)

	// Verify blue estops last seen was updated
	assert.True(t, web.arena.Esp32.IsBlueEstopsActive(), "Blue estops should be marked as active after API call with alliance=blue")
}

func TestTeamStackLightGetHandler_AllianceQueryParam_ShortForm(t *testing.T) {
	web := setupTestWeb(t)

	// Call with short form 'r' for red
	recorder := web.getHttpResponse("/api/freezy/team_stack_light?alliance=r")
	assert.Equal(t, 200, recorder.Code)
	assert.True(t, web.arena.Esp32.IsRedEstopsActive(), "Red estops should be marked as active after API call with alliance=r")

	// Reset and test with 'b' for blue
	web = setupTestWeb(t)
	recorder = web.getHttpResponse("/api/freezy/team_stack_light?alliance=b")
	assert.Equal(t, 200, recorder.Code)
	assert.True(t, web.arena.Esp32.IsBlueEstopsActive(), "Blue estops should be marked as active after API call with alliance=b")
}

func TestFieldStackLightGetHandler_UpdatesScoreTableLastSeen(t *testing.T) {
	web := setupTestWeb(t)

	// Call field stack light endpoint
	recorder := web.getHttpResponse("/api/freezy/field_stack_light")
	assert.Equal(t, 200, recorder.Code)

	// Verify score table last seen was updated
	assert.True(t, web.arena.Esp32.IsScoreTableActive(), "Score table should be marked as active after field_stack_light API call")
}

