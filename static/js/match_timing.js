// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Shared client-side logic for interpreting match state and timing notifications.

// MatchType enum values.
const matchTypeTest = 0;
const matchTypePractice = 1;
const matchTypeQualification = 2;
const matchTypePlayoff = 3;

const matchStates = {
  0: "PRE_MATCH",
  1: "START_MATCH",
  2: "WARMUP_PERIOD",
  3: "AUTO_PERIOD",
  4: "PAUSE_PERIOD",
  5: "TRANSITION_PERIOD",
  6: "SHIFT1_PERIOD",
  7: "SHIFT2_PERIOD",
  8: "SHIFT3_PERIOD",
  9: "SHIFT4_PERIOD",
  10: "ENDGAME_PERIOD",
  11: "POST_MATCH",
  12: "TIMEOUT_ACTIVE",
  13: "POST_TIMEOUT"
};
let matchTiming;

// Handles a websocket message containing the length of each period in the match.
const handleMatchTiming = function (data) {
  matchTiming = data;
};

// Converts the raw match state and time into a human-readable state and per-period time. Calls the provided
// callback with the result.
const translateMatchTime = function (data, callback) {
  var matchStateText;
  switch (matchStates[data.MatchState]) {
    case "PRE_MATCH":
      matchStateText = "PRE-MATCH";
      break;
    case "START_MATCH":
    case "WARMUP_PERIOD":
      matchStateText = "WARMUP";
      break;
    case "AUTO_PERIOD":
      matchStateText = "AUTONOMOUS";
      break;
    case "PAUSE_PERIOD":
      matchStateText = "PAUSE";
      break;
    case "TRANSITION_PERIOD":
      matchStateText = "TRANSITION";
      break;
    case "SHIFT1_PERIOD":
      matchStateText = "SHIFT1";
      break;
    case "SHIFT2_PERIOD":
      matchStateText = "SHIFT2";
      break;
    case "SHIFT3_PERIOD":
      matchStateText = "SHIFT3";
      break;
    case "SHIFT4_PERIOD":
      matchStateText = "SHIFT4";
      break;
    case "ENDGAME_PERIOD":
      matchStateText = "END-GAME";
      break;
    case "POST_MATCH":
      matchStateText = "POST-MATCH";
      break;
    case "TIMEOUT_ACTIVE":
    case "POST_TIMEOUT":
      matchStateText = "TIMEOUT";
      break;
  }
  callback(matchStates[data.MatchState], matchStateText, getCountdown(data.MatchState, data.MatchTimeSec));
};

// Returns the per-period countdown for the given match state and overall time into the match.
const getCountdown = function (matchState, matchTimeSec) {
  switch (matchStates[matchState]) {
    case "PRE_MATCH":
    case "START_MATCH":
    case "WARMUP_PERIOD":
      return matchTiming.AutoDurationSec;
    case "AUTO_PERIOD":
      return matchTiming.WarmupDurationSec + matchTiming.AutoDurationSec - matchTimeSec;
    case "PAUSE_PERIOD":
      return matchTiming.WarmupDurationSec + matchTiming.AutoDurationSec + matchTiming.PauseDurationSec - matchTimeSec;
    case "TRANSITION_PERIOD":
      return matchTiming.WarmupDurationSec + matchTiming.AutoDurationSec + matchTiming.PauseDurationSec + matchTiming.TransitionShiftDurationSec - matchTimeSec;
    case "SHIFT1_PERIOD":
      return matchTiming.WarmupDurationSec + matchTiming.AutoDurationSec + matchTiming.PauseDurationSec + matchTiming.TransitionShiftDurationSec +
        matchTiming.AllianceShiftDurationSec*1 - matchTimeSec;
    case "SHIFT2_PERIOD":
      return matchTiming.WarmupDurationSec + matchTiming.AutoDurationSec + matchTiming.PauseDurationSec + matchTiming.TransitionShiftDurationSec +
        matchTiming.AllianceShiftDurationSec*2 - matchTimeSec;
    case "SHIFT3_PERIOD":
      return matchTiming.WarmupDurationSec + matchTiming.AutoDurationSec + matchTiming.PauseDurationSec + matchTiming.TransitionShiftDurationSec +
        matchTiming.AllianceShiftDurationSec*3 - matchTimeSec;
    case "SHIFT4_PERIOD":
      return matchTiming.WarmupDurationSec + matchTiming.AutoDurationSec + matchTiming.PauseDurationSec + matchTiming.TransitionShiftDurationSec +
        matchTiming.AllianceShiftDurationSec*4 - matchTimeSec;
    case "ENDGAME_PERIOD":
      return matchTiming.WarmupDurationSec + matchTiming.AutoDurationSec + matchTiming.PauseDurationSec + matchTiming.TransitionShiftDurationSec +
        matchTiming.AllianceShiftDurationSec*4 + matchTiming.EndGameDurationSec - matchTimeSec;
    case "TIMEOUT_ACTIVE":
      return matchTiming.TimeoutDurationSec - matchTimeSec;
    default:
      return 0;
  }
};

// Converts the given countdown in seconds to a string with a colon separator and leading zero padding.
const getCountdownString = function (countdownSec) {
  let countdownString = String(countdownSec % 60);
  if (countdownString.length === 1) {
    countdownString = "0" + countdownString;
  }
  return Math.floor(countdownSec / 60) + ":" + countdownString;
};
