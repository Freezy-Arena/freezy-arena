// Copyright 2018 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Web handlers for the field monitor display showing robot connection status.

package web

import (
	"github.com/Team254/cheesy-arena/game"
	//"github.com/Team254/cheesy-arena/model"
	"encoding/json"
	//"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/Team254/cheesy-arena/field"
)

// RequestPayload represents the structure of the incoming POST data.
type RequestPayload struct {
	Channel int  `json:"channel"`
	State   bool `json:"state"`
}

// Renders the field monitor display.
func (web *Web) eStopStatePostHandler(w http.ResponseWriter, r *http.Request) {
	// Ensure the request is a POST request.
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Parse the request body.
	var payload []RequestPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
    	http.Error(w, "Invalid request payload", http.StatusBadRequest)
    	return
	}

	for _, item := range payload {
    	web.arena.Plc.SetAlternateIOStopState(item.Channel, item.State)
	}

	// Respond with success.
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("eStop state updated successfully."))

}

func (web *Web) getAllPlcCoilsGetHandler(w http.ResponseWriter, r *http.Request) {
	// Ensure the request is a GET request.
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Get the current state of all PLC coils.
    coilsArray := web.arena.Plc.GetAllCoils()
    coilsArrayNames := web.arena.Plc.GetCoilNames()

	// Build a map pairing coil names with their values.
    coilsMap := make(map[string]bool)
    for i, name := range coilsArrayNames {
        if i < len(coilsArray) {
            coilsMap[name] = coilsArray[i]
        }
    }
	
	// Marshal the response payload.
	response, err := json.Marshal(coilsMap)
	if err != nil {
		http.Error(w, "Failed to marshal PLC Coils state", http.StatusInternalServerError)
		return
	}

	// Send the response.
	w.Write(response)
}

// Handles the request to start the match.
func (web *Web) startMatchPostHandler(w http.ResponseWriter, r *http.Request) {
	// Ensure the request is a POST request.
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Start the match.
	web.arena.StartMatch()

	// Respond with success.
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Field stack light state updated successfully."))
}

type fieldStackLight struct {
	Red    bool `json:"redStackLight"`
	Blue   bool `json:"blueStackLight"`
	Orange bool `json:"orangeStackLight"`
	Green  bool `json:"greenStackLight"`
}

func (web *Web) fieldStackLightGetHandler(w http.ResponseWriter, r *http.Request) {
	// Ensure the request is a GET request.
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Track that the score table module is calling
	web.arena.Esp32.UpdateScoreTableLastSeen()

	// Get the current state of the field stack light.
	var stackLight fieldStackLight
	stackLight.Red, stackLight.Blue, stackLight.Orange, stackLight.Green = web.arena.Plc.GetFieldStackLight()

	// Marshal the response payload.
	response, err := json.Marshal(stackLight)
	if err != nil {
		http.Error(w, "Failed to marshal eStop state", http.StatusInternalServerError)
		return
	}

	// Send the response.
	w.Write(response)
}

type lightState struct {
	Color string `json:"color"`
	Blink bool `json:"blink"`
}
// Structure representing one light fixture
// Each lightState represents one light in the stack.
type teamStackLight struct {
	LightStates [2]lightState `json:"lightStates"`
}
// Structure that represents all of the team stack lights
type allStackLights struct {
	Red [3]teamStackLight `json:"red"`
	Blue [3]teamStackLight `json:"blue"`
}
func (web *Web) teamStackLightGetHandler(w http.ResponseWriter, r *http.Request) {
		// Ensure the request is a GET request.
		// See the team_sign.go method: generateTeamNumberTexts for the template
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Track which estops module is calling via the alliance query parameter
	alliance := strings.ToLower(r.URL.Query().Get("alliance"))
	switch alliance {
	case "red", "r":
		web.arena.Esp32.UpdateRedEstopsLastSeen()
	case "blue", "b":
		web.arena.Esp32.UpdateBlueEstopsLastSeen()
	}

	var stackLights allStackLights
	// stackLights.Red = [3]teamStackLight
	// stackLights.Blue = [3]teamStackLight

	for team, allianceStation := range web.arena.AllianceStations { 
		var teamStackLights = &stackLights.Blue
		var allianceColor = "blue"
		if team[0] == 'R'	{
			teamStackLights = &stackLights.Red
			allianceColor = "red"
		}
		dsN,_ := strconv.Atoi(string(team[1]))
		teamStackLight := &teamStackLights[dsN-1]
		//  The lights are as follows:
		//  L2: Blue/Red
		//     off: Connection est. to robot
		//     solid: Robot enabled
		//     flash: no connection to robot or bypassed
		//  L1: Amber
		//     off: Estop not pressed/disabled
		//     solid: Estop pressed/enabled
		//     flash: Astop pressed/enabled during autonomous period


		// Light/Layer 1 - Stop States
		if allianceStation.EStop {
			teamStackLight.LightStates[0] = lightState{Color: "orangered", Blink: false}
		} else if allianceStation.AStop && web.arena.MatchState == field.AutoPeriod {
			teamStackLight.LightStates[0] = lightState{Color: "orangered", Blink: true}
		} else {
			teamStackLight.LightStates[0] = lightState{Color: "black", Blink: false}
		}

		// Light/Layer 2 - Robot States
		// Blink with any problem 
		// Solid during the match if all is good.
		// Off off-match if all is good.
		var ok = true;
		if allianceStation.Bypass {
			ok = false
			// This is always false for some reason
		// } else if !allianceStation.Ethernet {
		// 	ok = false
		} else if allianceStation.DsConn == nil {
			ok = false
		} else if allianceStation.DsConn.WrongStation != "" {
			ok = false
		} else if !allianceStation.DsConn.RadioLinked {
			ok = false
		} else if !allianceStation.DsConn.RioLinked {
			ok = false
		} else if !allianceStation.DsConn.RobotLinked {
			ok = false
		}

		if ok {
			if web.arena.MatchState == field.AutoPeriod || web.arena.MatchState == field.PausePeriod || web.arena.MatchState == field.TransitionShift ||
				web.arena.MatchState == field.Shift1 || web.arena.MatchState == field.Shift2 || web.arena.MatchState == field.Shift3 || web.arena.MatchState == field.Shift4 ||
				web.arena.MatchState == field.EndGame {
				// Robot enabled during match
				teamStackLight.LightStates[1] = lightState{Color: allianceColor, Blink: false}
			} else {
				// Robot connected outside of the match
				teamStackLight.LightStates[1] = lightState{Color: "black", Blink: false}
			}
		} else {
			teamStackLight.LightStates[1] = lightState{Color: allianceColor, Blink: true}		
		}
	}

	// Marshal the response payload.
	response, err := json.Marshal(stackLights)
	if err != nil {
		http.Error(w, "Failed to marshal team stacklights state", http.StatusInternalServerError)
		return
	}

	// Send the response.
	w.Write(response)
}

type hubStates struct {
	Red lightState `json:"red"`
	Blue lightState `json:"blue"`
} 
// Provides a single API for a hub to retrieve it's state which is:
// Alliance Color Solid:   HUB Active
// Alliance color Pulsing: HUB Deactivation Warning
// Purple:                 Field is safe for staff
// Green:                  Field is safe for all
// Off
func (web *Web) teamHubStateGetHandler(w http.ResponseWriter, r *http.Request) {
		// Ensure the request is a GET request.
		// See the team_sign.go method: generateTeamNumberTexts for the template
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Track which hub module is calling via the alliance query parameter
	alliance := strings.ToLower(r.URL.Query().Get("alliance"))
	switch alliance {
	case "red", "r":
		web.arena.Esp32.UpdateRedHubLastSeen()
	case "blue", "b":
		web.arena.Esp32.UpdateBlueHubLastSeen()
	}

	var hubStates hubStates

	// State during match
	matchTimeSec := web.arena.MatchTimeSec()
	switch web.arena.MatchState {
	case field.AutoPeriod, field.PausePeriod, field.TransitionShift, field.Shift1, field.Shift2, field.Shift3, field.Shift4, field.EndGame:
		// Determine if we're within 3 seconds of the current state ending (for blink warning)
		var stateEndSec float64
		switch web.arena.MatchState {
		case field.PausePeriod:
			stateEndSec = game.GetDurationToPauseEnd().Seconds()
		case field.TransitionShift:
			stateEndSec = game.GetDurationToShift1Start().Seconds()
		case field.Shift1:
			stateEndSec = game.GetDurationToShiftEnd(1).Seconds()
		case field.Shift2:
			stateEndSec = game.GetDurationToShiftEnd(2).Seconds()
		case field.Shift3:
			stateEndSec = game.GetDurationToShiftEnd(3).Seconds()
		case field.Shift4:
			stateEndSec = game.GetDurationToShiftEnd(4).Seconds()
		case field.EndGame:
			stateEndSec = game.GetDurationToTeleopEnd().Seconds()
		}
		timeUntilStateEnd := stateEndSec - matchTimeSec
		blinkWarning := timeUntilStateEnd > 0 && timeUntilStateEnd <= 3

		// Determine which hubs will be active in the next state
		var nextStateRedActive, nextStateBlueActive bool
		switch web.arena.MatchState {
		case field.PausePeriod:
			// Next state is TransitionShift - both hubs active (no blink needed)
			nextStateRedActive = true
			nextStateBlueActive = true
		case field.TransitionShift:
			// Next state is Shift1 - use pre-calculated FirstShiftHubState (calculated at end of Auto)
			nextStateRedActive = web.arena.FirstShiftHubState&field.RedAllianceHubBit != 0
			nextStateBlueActive = web.arena.FirstShiftHubState&field.BlueAllianceHubBit != 0
		case field.Shift1, field.Shift2, field.Shift3:
			// Next state has opposite hub active
			nextStateRedActive = web.arena.HubsActive&field.RedAllianceHubBit == 0
			nextStateBlueActive = web.arena.HubsActive&field.BlueAllianceHubBit == 0
		case field.Shift4:
			// Next state is EndGame - both hubs active (no blink needed)
			nextStateRedActive = true
			nextStateBlueActive = true
		case field.EndGame:
			// Next state is PostMatch - no hubs active
			nextStateRedActive = false
			nextStateBlueActive = false
		}

		// Red
		hubStates.Red.Color = "black"
		if web.arena.HubsActive&field.RedAllianceHubBit != 0 {
			hubStates.Red.Color = "red"
			// Blink if hub is active now but will become inactive in next state
			hubStates.Red.Blink = blinkWarning && !nextStateRedActive
		}

		// Blue
		hubStates.Blue.Color = "black"
		if web.arena.HubsActive&field.BlueAllianceHubBit != 0 {
			hubStates.Blue.Color = "blue"
			// Blink if hub is active now but will become inactive in next state
			hubStates.Blue.Blink = blinkWarning && !nextStateBlueActive
		}
		case field.PostMatch, field.PreMatch:
			if web.arena.FieldVolunteers {			
				hubStates.Red.Color = "green"
				hubStates.Blue.Color = "green"
			} else {
				hubStates.Red.Color = "purple"
				hubStates.Blue.Color = "purple"
			}
			hubStates.Red.Blink = false
			hubStates.Blue.Blink = false
		default:
			hubStates.Red.Color = "black"
			hubStates.Red.Blink = false
			hubStates.Blue.Color = "black"
			hubStates.Blue.Blink = false

	}

	// Marshal the response payload.
	response, err := json.Marshal(hubStates)
	if err != nil {
		http.Error(w, "Failed to marshal hub state", http.StatusInternalServerError)
		return
	}

	// Send the response.
	w.Write(response)
}

// HubBatteryPayload represents the structure of the incoming POST data for hub battery status.
type HubBatteryPayload struct {
	Voltage float64 `json:"voltage"`
	Percent float64 `json:"percent"`
}

// POST /api/freezy/hub_status
// Updates the battery status for a hub (red or blue alliance).
func (web *Web) teamHubStatusPostHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Determine which hub is reporting via the alliance query parameter
	alliance := strings.ToLower(r.URL.Query().Get("alliance"))
	if alliance != "red" && alliance != "r" && alliance != "blue" && alliance != "b" {
		http.Error(w, "Missing or invalid alliance query parameter; must be 'red' or 'blue'", http.StatusBadRequest)
		return
	}

	// Parse the request body
	var payload HubBatteryPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Update the battery status and last seen timestamp for the appropriate hub
	switch alliance {
	case "red", "r":
		web.arena.Esp32.SetRedHubBattery(payload.Voltage, payload.Percent)
		web.arena.Esp32.UpdateRedHubLastSeen()
	case "blue", "b":
		web.arena.Esp32.SetBlueHubBattery(payload.Voltage, payload.Percent)
		web.arena.Esp32.UpdateBlueHubLastSeen()
	}

	// Notify arena status subscribers of the update
	web.arena.ArenaStatusNotifier.Notify()

	// Respond with success
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hub battery status updated successfully."))
}

type incrementElementPayload struct {
    Alliance string `json:"alliance"`
    Element  string `json:"element"`
}

// POST /freezy/alternateio/increment
// Increments the specified element counter on the chosen alliance realtime score.
func (web *Web) incrementElementPostHandler(w http.ResponseWriter, r *http.Request) {

	//log.Printf("incrementElementPostHandler: received request")
    if r.Method != http.MethodPost {
        http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
        return
    }
	// Parse the request body.
    var p incrementElementPayload
    if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
        http.Error(w, "Invalid request payload", http.StatusBadRequest)
        return
    }

    // Select alliance realtime score
    var scorePtr *field.RealtimeScore
    switch strings.ToLower(p.Alliance) {
    case "red", "r":
        scorePtr = web.arena.RedRealtimeScore
    case "blue", "b":
        scorePtr = web.arena.BlueRealtimeScore
    default:
        http.Error(w, "Unknown alliance; must be 'red' or 'blue'", http.StatusBadRequest)
        return
    }

    if scorePtr == nil {
        http.Error(w, "Realtime score not initialized for alliance", http.StatusInternalServerError)
        return
    }

    // Increment the requested element counter. Add cases as needed.
    switch p.Element {
    case "ProcessorAlgae":
        scorePtr.CurrentScore.ProcessorAlgae++
		web.arena.RealtimeScoreNotifier.Notify()
        //writeJsonOK(w, map[string]interface{}{"ok": true, "element": p.Element, "value": scorePtr.CurrentScore.ProcessorAlgae})
        return
    case "Barge":
        scorePtr.CurrentScore.BargeAlgae++
        web.arena.RealtimeScoreNotifier.Notify()
        //writeJsonOK(w, map[string]interface{}{"ok": true, "element": p.Element, "value": scorePtr.CurrentScore.BargeAlgae})
		return
    default:
        http.Error(w, "Unknown element", http.StatusBadRequest)
        return
    }
}
