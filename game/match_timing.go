// Copyright 2017 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Game-specific period timing.

package game

import "time"

const (
	TeleopGracePeriodSec = 3
)

var MatchTiming = struct {
	WarmupDurationSec           int
	AutoDurationSec             int
	TransitionShiftDurationSec  int 
	AllianceShiftDurationSec    int
	EndGameDurationSec 					int
	TimeoutDurationSec          int
}{0, 20, 10, 25, 30, 0}

func GetDurationToAutoEnd() time.Duration {
	return time.Duration(MatchTiming.WarmupDurationSec+MatchTiming.AutoDurationSec) * time.Second
}

func GetDurationToShift1Start() time.Duration {
	return time.Duration(
		MatchTiming.WarmupDurationSec+MatchTiming.AutoDurationSec+MatchTiming.TransitionShiftDurationSec,
	) * time.Second
}

// There are 1-4 shifts
func GetDurationToShiftEnd(shift int) time.Duration {
	return time.Duration(
		MatchTiming.WarmupDurationSec+MatchTiming.AutoDurationSec+MatchTiming.TransitionShiftDurationSec+(MatchTiming.AllianceShiftDurationSec * shift),
	) * time.Second
}

// func GetDurationToTeleopStart() time.Duration {
// 	return time.Duration(
// 		MatchTiming.WarmupDurationSec+MatchTiming.AutoDurationSec+MatchTiming.PauseDurationSec,
// 	) * time.Second
// }

func GetDurationToTeleopEnd() time.Duration {
	return time.Duration(
		MatchTiming.WarmupDurationSec+MatchTiming.AutoDurationSec+MatchTiming.TransitionShiftDurationSec+
			MatchTiming.AllianceShiftDurationSec*4 + MatchTiming.EndGameDurationSec,
	) * time.Second
}
