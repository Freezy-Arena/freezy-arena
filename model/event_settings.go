// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Model and datastore read/write methods for event-level configuration.

package model

import (
	"strings"

	"github.com/Team254/cheesy-arena/game"
)

type PlayoffType int

const (
	DoubleEliminationPlayoff PlayoffType = iota
	SingleEliminationPlayoff
)

// Configured here to avoid circular import dependencies.
var (
	sccDefaultUpCommands = []string{
		"configure terminal",
		"interface range gigabitEthernet 1/2-4",
		"no shutdown",
		"exit",
		"exit",
		"exit",
	}
	sccDefaultDownCommands = []string{
		"configure terminal",
		"interface range gigabitEthernet 1/2-4",
		"shutdown",
		"exit",
		"exit",
		"exit",
	}
)

type EventSettings struct {
	Id                              int `db:"id"`
	Name                            string
	LogoSuffix                      string
	PlayoffType                     PlayoffType
	NumPlayoffAlliances             int
	SelectionRound2Order            string
	SelectionRound3Order            string
	SelectionShowUnpickedTeams      bool
	TbaDownloadEnabled              bool
	TbaPublishingEnabled            bool
	TbaEventCode                    string
	TbaSecretId                     string
	TbaSecret                       string
	NexusEnabled                    bool
	NetworkSecurityEnabled          bool
	ApAddress                       string
	ApPassword                      string
	ApChannel                       int
	SwitchAddress                   string
	SwitchPassword                  string
	SCCManagementEnabled            bool
	RedSCCAddress                   string
	BlueSCCAddress                  string
	SCCUsername                     string
	SCCPassword                     string
	SCCUpCommands                   string
	SCCDownCommands                 string
	PlcAddress                      string
	AlternateIOEnabled              bool
	ApiMonitorEnabled              bool
	ScoreTableEstopAddress          string
	RedAllianceStationEstopAddress  string
	BlueAllianceStationEstopAddress string
	RedHubAddress                   string
	BlueHubAddress                  string
	AdminPassword                   string
	TeamSignRed1Id                  int
	TeamSignRed2Id                  int
	TeamSignRed3Id                  int
	TeamSignRedTimerId              int
	TeamSignBlue1Id                 int
	TeamSignBlue2Id                 int
	TeamSignBlue3Id                 int
	TeamSignBlueTimerId             int
	UseLiteUdpPort                  bool
	BlackmagicAddresses             string
	WarmupDurationSec               int
	AutoDurationSec                 int
	PauseDurationSec                int
	TransitionShiftDurationSec      int
	AllianceShiftDurationSec        int
	EndGameDurationSec              int
	FirstShiftAlliance       string
	BargeBonusPointThreshold int
	FlashDSEnabled                  bool
	IncludeAlgaeInBargeBonus        bool
}

func (database *Database) GetEventSettings() (*EventSettings, error) {
	allEventSettings, err := database.eventSettingsTable.getAll()
	if err != nil {
		return nil, err
	}
	if len(allEventSettings) == 1 {
		return &allEventSettings[0], nil
	}

	// Database record doesn't exist yet; create it now.
	eventSettings := EventSettings{
		Name:                        "Untitled Event",
		LogoSuffix:                  "",
		PlayoffType:                 DoubleEliminationPlayoff,
		NumPlayoffAlliances:         8,
		SelectionRound2Order:        "L",
		SelectionRound3Order:        "",
		SelectionShowUnpickedTeams:  true,
		TbaDownloadEnabled:          true,
		ApChannel:                   36,
		AlternateIOEnabled:          false,
		ApiMonitorEnabled:          false,
		SCCUpCommands:               strings.Join(sccDefaultUpCommands, "\n"),
		SCCDownCommands:             strings.Join(sccDefaultDownCommands, "\n"),
		WarmupDurationSec:           game.MatchTiming.WarmupDurationSec,
		AutoDurationSec:             game.MatchTiming.AutoDurationSec,
		PauseDurationSec:            game.MatchTiming.PauseDurationSec,
		TransitionShiftDurationSec:  game.MatchTiming.TransitionShiftDurationSec,
		AllianceShiftDurationSec:    game.MatchTiming.AllianceShiftDurationSec,
		EndGameDurationSec:          game.MatchTiming.EndGameDurationSec,
		FirstShiftAlliance:       "blue",
		BargeBonusPointThreshold: game.BargeBonusPointThreshold,
		IncludeAlgaeInBargeBonus:    game.IncludeAlgaeInBargeBonus,
	}

	if err := database.eventSettingsTable.create(&eventSettings); err != nil {
		return nil, err
	}
	return &eventSettings, nil
}

func (database *Database) UpdateEventSettings(eventSettings *EventSettings) error {
	return database.eventSettingsTable.update(eventSettings)
}
