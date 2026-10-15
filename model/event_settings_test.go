// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)

package model

import (
	"github.com/Team254/cheesy-arena/game"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEventSettingsReadWrite(t *testing.T) {
	db := setupTestDb(t)
	defer db.Close()

	eventSettings, err := db.GetEventSettings()
	assert.Nil(t, err)
	assert.Equal(
		t,
		EventSettings{
			Id:                         1,
			Name:                       "Untitled Event",
			LogoSuffix:                 "",
			PlayoffType:                DoubleEliminationPlayoff,
			NumPlayoffAlliances:        8,
			AutoAudienceScoreSec:       20,
			AutoAudiencePlayoffDelaySec: 10,
			AutoAudienceBracketSec:     20,
			AutoAudienceLastMatchSec:   40,
			SelectionRound2Order:       "L",
			SelectionRound3Order:       "",
			SelectionShowUnpickedTeams: true,
			TbaDownloadEnabled:         true,
			ApChannel:                  36,
			SCCUpCommands:              "configure terminal\ninterface range gigabitEthernet 1/2-4\nno shutdown\nexit\nexit\nexit",
			SCCDownCommands:            "configure terminal\ninterface range gigabitEthernet 1/2-4\nshutdown\nexit\nexit\nexit",
			LedControllerAddress:       "",
			LedUniverseMode:            "single",
			AutoDurationSec:            20,
			PauseDurationSec:           3,
			TransitionShiftDurationSec: 10,
			ShiftDurationSec:           25,
			EndgameDurationSec:         30,
			EnergizedBonusThreshold:    100,
			SuperchargedBonusThreshold: 360,
			TraversalBonusThreshold:    50,
			CompanionAddress:           "",
			CompanionPort:              0,
		},
		*eventSettings,
	)
	assert.Equal(t, 140, game.GetTeleopDurationSec())

	eventSettings.Name = "Chezy Champs"
	eventSettings.NumPlayoffAlliances = 6
	eventSettings.SelectionRound2Order = "F"
	eventSettings.SelectionRound3Order = "L"
	err = db.UpdateEventSettings(eventSettings)
	assert.Nil(t, err)
	eventSettings2, err := db.GetEventSettings()
	assert.Nil(t, err)
	assert.Equal(t, eventSettings, eventSettings2)
}

func TestAudienceRankingsDisplayFieldsBackwardCompatible(t *testing.T) {
	db := setupTestDb(t)
	defer db.Close()
	settings, err := db.GetEventSettings()
	assert.NoError(t, err)
	assert.Equal(t, []string{"rp"}, settings.RankingsDisplayFields())

	settings.AudienceRankingsFields = "nextMatch,record"
	assert.NoError(t, db.UpdateEventSettings(settings))
	stored, err := db.GetEventSettings()
	assert.NoError(t, err)
	assert.Equal(t, []string{"nextMatch", "record"}, stored.RankingsDisplayFields())

	stored.AudienceRankingsFields = "-"
	assert.NoError(t, db.UpdateEventSettings(stored))
	final, err := db.GetEventSettings()
	assert.NoError(t, err)
	assert.Equal(t, []string{}, final.RankingsDisplayFields())
}

func TestAudienceAutomationDefaultsForOldEvents(t *testing.T) {
	old := &EventSettings{} // Missing timing keys in a pre-upgrade Bolt record.
	assert.Equal(t, 20, old.AudienceScoreSeconds())
	assert.Equal(t, 10, old.AudiencePlayoffDelaySeconds())
	assert.Equal(t, 20, old.AudienceBracketSeconds())
	assert.Equal(t, 40, old.AudienceLastMatchSeconds())

	old.AutoAudienceScoreSec = 33
	old.AutoAudiencePlayoffDelaySec = 7
	old.AutoAudienceBracketSec = 15
	old.AutoAudienceLastMatchSec = 55
	assert.Equal(t, 33, old.AudienceScoreSeconds())
	assert.Equal(t, 7, old.AudiencePlayoffDelaySeconds())
	assert.Equal(t, 15, old.AudienceBracketSeconds())
	assert.Equal(t, 55, old.AudienceLastMatchSeconds())
}

func TestAudienceRankingsFieldVisibility(t *testing.T) {
	settings := &EventSettings{}
	assert.True(t, settings.HasAudienceRankingField("rp"))
	assert.False(t, settings.HasAudienceRankingField("nextMatch"))
	settings.AudienceRankingsFields = "nextMatch,record"
	assert.True(t, settings.HasAudienceRankingField("nextMatch"))
	assert.False(t, settings.HasAudienceRankingField("rp"))
	settings.AudienceRankingsFields = "-"
	assert.False(t, settings.HasAudienceRankingField("record"))
}
