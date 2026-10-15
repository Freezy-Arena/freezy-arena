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
	Id                               int `db:"id"`
	Name                             string
	LogoSuffix                       string
	PlayoffType                      PlayoffType
	NumPlayoffAlliances              int
	SelectionRound2Order             string
	SelectionRound3Order             string
	SelectionShowUnpickedTeams       bool
	TbaDownloadEnabled               bool
	TbaPublishingEnabled             bool
	TbaEventCode                     string
	TbaSecretId                      string
	TbaSecret                        string
	AutoAudienceDisplayEnabled       bool
	AutoAudienceScoreSec            int
	AutoAudiencePlayoffDelaySec     int
	AutoAudienceBracketSec          int
	AutoAudienceLastMatchSec        int
	AudienceRankingsFields           string
	NexusEnabled                     bool
	NexusAutoQueueEnabled            bool
	NexusAutoQueueKey                string
	NetworkSecurityEnabled           bool
	ApAddress                        string
	ApPassword                       string
	ApChannel                        int
	SwitchAddress                    string
	SwitchPassword                   string
	SCCManagementEnabled             bool
	RedSCCAddress                    string
	BlueSCCAddress                   string
	SCCUsername                      string
	SCCPassword                      string
	SCCUpCommands                    string
	SCCDownCommands                  string
	PlcAddress                       string
	LedControllerAddress             string
	AlternateIOEnabled               bool
	ScoreTableEstopAddress           string
	RedAllianceStationEstopAddress   string
	BlueAllianceStationEstopAddress  string
	LedUniverseMode                  string
	AdminPassword                    string
	TeamSignRed1Id                   int
	TeamSignRed2Id                   int
	TeamSignRed3Id                   int
	TeamSignRedTimerId               int
	TeamSignBlue1Id                  int
	TeamSignBlue2Id                  int
	TeamSignBlue3Id                  int
	TeamSignBlueTimerId              int
	UseLiteUdpPort                   bool
	BlackmagicAddresses              string
	CompanionAddress                 string
	CompanionPort                    int
	CompanionMatchPreviewPage        int
	CompanionMatchPreviewRow         int
	CompanionMatchPreviewColumn      int
	CompanionSetAudiencePage         int
	CompanionSetAudienceRow          int
	CompanionSetAudienceColumn       int
	CompanionMatchStartPage          int
	CompanionMatchStartRow           int
	CompanionMatchStartColumn        int
	CompanionTeleopStartPage         int
	CompanionTeleopStartRow          int
	CompanionTeleopStartColumn       int
	CompanionEndgameStartPage        int
	CompanionEndgameStartRow         int
	CompanionEndgameStartColumn      int
	CompanionMatchEndPage            int
	CompanionMatchEndRow             int
	CompanionMatchEndColumn          int
	CompanionPostResultPage          int
	CompanionPostResultRow           int
	CompanionPostResultColumn        int
	CompanionAllianceSelectionPage   int
	CompanionAllianceSelectionRow    int
	CompanionAllianceSelectionColumn int
	CompanionMatchAbortPage          int
	CompanionMatchAbortRow           int
	CompanionMatchAbortColumn        int
	AutoDurationSec                  int
	PauseDurationSec                 int
	TransitionShiftDurationSec       int
	ShiftDurationSec                 int
	EndgameDurationSec               int
	EnergizedBonusThreshold          int
	SuperchargedBonusThreshold       int
	TraversalBonusThreshold          int
	FlashDSEnabled                   bool
}

func (database *Database) GetEventSettings() (*EventSettings, error) {
	allEventSettings, err := database.eventSettingsTable.getAll()
	if err != nil {
		return nil, err
	}
	if len(allEventSettings) == 1 {
		eventSettings := allEventSettings[0]
		return &eventSettings, nil
	}

	// Database record doesn't exist yet; create it now.
	eventSettings := EventSettings{
		Name:                       "Untitled Event",
		LogoSuffix:                 "",
		PlayoffType:                DoubleEliminationPlayoff,
		NumPlayoffAlliances:        8,
		SelectionRound2Order:       "L",
		AutoAudienceScoreSec:       20,
		AutoAudiencePlayoffDelaySec: 10,
		AutoAudienceBracketSec:     20,
		AutoAudienceLastMatchSec:   40,
		SelectionRound3Order:       "",
		SelectionShowUnpickedTeams: true,
		TbaDownloadEnabled:         true,
		ApChannel:                  36,
		AlternateIOEnabled:         false,
		SCCUpCommands:              strings.Join(sccDefaultUpCommands, "\n"),
		SCCDownCommands:            strings.Join(sccDefaultDownCommands, "\n"),
		LedUniverseMode:            "single",
		CompanionAddress:           "",
		AutoDurationSec:            game.MatchTiming.AutoDurationSec,
		PauseDurationSec:           game.MatchTiming.PauseDurationSec,
		TransitionShiftDurationSec: game.MatchTiming.TransitionShiftDurationSec,
		ShiftDurationSec:           game.MatchTiming.ShiftDurationSec,
		EndgameDurationSec:         game.MatchTiming.EndgameDurationSec,
		EnergizedBonusThreshold:    game.EnergizedBonusThreshold,
		SuperchargedBonusThreshold: game.SuperchargedBonusThreshold,
		TraversalBonusThreshold:    game.TraversalBonusThreshold,
	}

	if err := database.eventSettingsTable.create(&eventSettings); err != nil {
		return nil, err
	}
	return &eventSettings, nil
}

func (database *Database) UpdateEventSettings(eventSettings *EventSettings) error {
	return database.eventSettingsTable.update(eventSettings)
}

// RankingsDisplayFields is the event-wide selection for both audience rankings modes.
// An empty string indicates an event created before this feature (RP by default).
// A dash represents an explicit choice to show only rank and team information.
func (settings *EventSettings) RankingsDisplayFields() []string {
	if settings.AudienceRankingsFields == "" {
		return []string{"rp"}
	}
	if settings.AudienceRankingsFields == "-" {
		return []string{}
	}
	return strings.Split(settings.AudienceRankingsFields, ",")
}

// AutomationDurationSec keeps pre-existing event databases compatible: prior
// records do not have the new timing fields, so their zero values use the
// historical timeouts instead of advancing instantly.
func automationDurationSec(configured, fallback int) int {
	if configured <= 0 {
		return fallback
	}
	return configured
}

func (settings *EventSettings) AudienceScoreSeconds() int {
	return automationDurationSec(settings.AutoAudienceScoreSec, 20)
}
func (settings *EventSettings) AudiencePlayoffDelaySeconds() int {
	return automationDurationSec(settings.AutoAudiencePlayoffDelaySec, 10)
}
func (settings *EventSettings) AudienceBracketSeconds() int {
	return automationDurationSec(settings.AutoAudienceBracketSec, 20)
}
func (settings *EventSettings) AudienceLastMatchSeconds() int {
	return automationDurationSec(settings.AutoAudienceLastMatchSec, 40)
}

func (settings *EventSettings) HasAudienceRankingField(field string) bool {
	for _, selected := range settings.RankingsDisplayFields() {
		if selected == field {
			return true
		}
	}
	return false
}
