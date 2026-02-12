// Copyright 2019 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Game-specific audience sound timings.

package game

type MatchSound struct {
	Name          string
	FileExtension string
	MatchTimeSec  float64
}

// List of sounds and how many seconds into the match they are played. A negative time indicates that the sound can only
// be triggered explicitly.
var MatchSounds []*MatchSound

func UpdateMatchSounds() {
	MatchSounds = []*MatchSound{
		{
			"start",
			"wav",
			0,
		},
		{
			"end",
			"wav",
			float64(MatchTiming.AutoDurationSec),
		},
		{
			"resume",
			"wav",
			float64(MatchTiming.AutoDurationSec + MatchTiming.PauseDurationSec),
		},
		{
			"alliance_shift",
			"wav",
			float64(MatchTiming.AutoDurationSec + MatchTiming.PauseDurationSec + MatchTiming.TransitionShiftDurationSec),
		},
		{
			"alliance_shift",
			"wav",
			float64(MatchTiming.AutoDurationSec + MatchTiming.PauseDurationSec + MatchTiming.TransitionShiftDurationSec + MatchTiming.AllianceShiftDurationSec),
		},
		{
			"alliance_shift",
			"wav",
			float64(MatchTiming.AutoDurationSec + MatchTiming.PauseDurationSec + MatchTiming.TransitionShiftDurationSec + (2*MatchTiming.AllianceShiftDurationSec)),
		},
		{
			"alliance_shift",
			"wav",
			float64(MatchTiming.AutoDurationSec + MatchTiming.PauseDurationSec + MatchTiming.TransitionShiftDurationSec + (3*MatchTiming.AllianceShiftDurationSec)),
		},
		{
			"steam_whistle",
			"wav",
			float64(
				MatchTiming.AutoDurationSec + MatchTiming.PauseDurationSec + MatchTiming.TransitionShiftDurationSec + MatchTiming.AllianceShiftDurationSec * 4,
			),
		},
		{
			"end",
			"wav",
			float64(MatchTiming.AutoDurationSec + MatchTiming.PauseDurationSec + MatchTiming.TransitionShiftDurationSec + MatchTiming.AllianceShiftDurationSec*4 + MatchTiming.EndGameDurationSec),
		},
		{
			"abort",
			"wav",
			-1,
		},
		{
			"match_result",
			"wav",
			-1,
		},
		{
			"pick_clock",
			"wav",
			-1,
		},
		{
			"pick_clock_expired",
			"wav",
			-1,
		},
	}
}
