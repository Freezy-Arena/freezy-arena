// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)

package web

import (
	"encoding/json"
	"github.com/Team254/cheesy-arena/game"
	"github.com/Team254/cheesy-arena/model"
	"github.com/Team254/cheesy-arena/tournament"
	"github.com/Team254/cheesy-arena/websocket"
	gorillawebsocket "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestMatchesApi(t *testing.T) {
	web := setupTestWeb(t)

	match1 := model.Match{
		Type:             model.Qualification,
		ShortName:        "Q1",
		Time:             time.Unix(0, 0),
		Red1:             1,
		Red2:             2,
		Red3:             3,
		Blue1:            4,
		Blue2:            5,
		Blue3:            6,
		Blue1IsSurrogate: true,
		Blue2IsSurrogate: true,
		Blue3IsSurrogate: true,
	}
	match2 := model.Match{
		Type:            model.Qualification,
		ShortName:       "Q2",
		Time:            time.Unix(600, 0),
		Red1:            7,
		Red2:            8,
		Red3:            9,
		Blue1:           10,
		Blue2:           11,
		Blue3:           12,
		Red1IsSurrogate: true,
		Red2IsSurrogate: true,
		Red3IsSurrogate: true,
	}
	match3 := model.Match{
		Type:      model.Practice,
		ShortName: "P1",
		Time:      time.Now(),
		Red1:      6,
		Red2:      5,
		Red3:      4,
		Blue1:     3,
		Blue2:     2,
		Blue3:     1,
	}
	web.arena.Database.CreateMatch(&match1)
	web.arena.Database.CreateMatch(&match2)
	web.arena.Database.CreateMatch(&match3)
	matchResult1 := model.BuildTestMatchResult(match1.Id, 1)
	web.arena.Database.CreateMatchResult(matchResult1)

	recorder := web.getHttpResponse("/api/matches/qualification")
	assert.Equal(t, 200, recorder.Code)
	assert.Equal(t, "application/json", recorder.Header()["Content-Type"][0])
	var matchesData []MatchWithResult
	err := json.Unmarshal([]byte(recorder.Body.String()), &matchesData)
	assert.Nil(t, err)
	if assert.Equal(t, 2, len(matchesData)) {
		assert.Equal(t, match1.Id, matchesData[0].Match.Id)
		assert.Equal(t, *matchResult1, matchesData[0].Result.MatchResult)
		assert.Equal(t, match2.Id, matchesData[1].Match.Id)
		assert.Nil(t, matchesData[1].Result)
	}
}

func TestRankingsApi(t *testing.T) {
	web := setupTestWeb(t)

	// Test that empty rankings produces an empty array.
	recorder := web.getHttpResponse("/api/rankings")
	assert.Equal(t, 200, recorder.Code)
	assert.Equal(t, "application/json", recorder.Header()["Content-Type"][0])
	rankingsData := struct {
		Rankings           []RankingWithNickname
		TeamNicknames      map[string]string
		HighestPlayedMatch string
	}{}
	err := json.Unmarshal([]byte(recorder.Body.String()), &rankingsData)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(rankingsData.Rankings))
	assert.Equal(t, "", rankingsData.HighestPlayedMatch)

	ranking1 := RankingWithNickname{*game.TestRanking2(), "Simbots"}
	ranking2 := RankingWithNickname{*game.TestRanking1(), "ChezyPof"}
	web.arena.Database.CreateRanking(&ranking1.Ranking)
	web.arena.Database.CreateRanking(&ranking2.Ranking)
	web.arena.Database.CreateMatch(&model.Match{Type: model.Qualification, ShortName: "Q29", Status: game.RedWonMatch})
	web.arena.Database.CreateMatch(&model.Match{Type: model.Qualification, ShortName: "Q30"})
	web.arena.Database.CreateTeam(&model.Team{Id: 254, Nickname: "ChezyPof"})
	web.arena.Database.CreateTeam(&model.Team{Id: 1114, Nickname: "Simbots"})

	recorder = web.getHttpResponse("/api/rankings")
	assert.Equal(t, 200, recorder.Code)
	assert.Equal(t, "application/json", recorder.Header()["Content-Type"][0])
	err = json.Unmarshal([]byte(recorder.Body.String()), &rankingsData)
	assert.Nil(t, err)
	if assert.Equal(t, 2, len(rankingsData.Rankings)) {
		assert.Equal(t, ranking1, rankingsData.Rankings[1])
		assert.Equal(t, ranking2, rankingsData.Rankings[0])
	}
	assert.Equal(t, "Q29", rankingsData.HighestPlayedMatch)
}

// Audience rankings options are shared, persisted, validated, and include
// the next scheduled (not already completed) match for every ranked team.
func TestAudienceRankingsDisplayOptions(t *testing.T) {
	web := setupTestWeb(t)
	assert.Nil(t, web.arena.Database.CreateRanking(game.TestRanking1()))
	assert.Nil(t, web.arena.Database.CreateRanking(game.TestRanking2()))

	// Completed matches do not appear as upcoming matches.
	assert.Nil(t, web.arena.Database.CreateMatch(&model.Match{
		Type: model.Qualification, TypeOrder: 1, ShortName: "Q1",
		Red1: 254, Blue1: 1114, Status: game.RedWonMatch,
	}))
	assert.Nil(t, web.arena.Database.CreateMatch(&model.Match{
		Type: model.Qualification, TypeOrder: 2, ShortName: "Q2",
		Red1: 254, Red2: 0, Blue1: 0,
	}))
	assert.Nil(t, web.arena.Database.CreateMatch(&model.Match{
		Type: model.Qualification, TypeOrder: 3, ShortName: "Q3",
		Red1: 254, Blue1: 1114,
	}))
	assert.Nil(t, web.arena.Database.CreateMatch(&model.Match{
		Type: model.Playoff, TypeOrder: 1, ShortName: "SF1-1",
		Red1: 1114,
	}))

	decode := func() map[string]any {
		recorder := web.getHttpResponse("/api/rankings")
		assert.Equal(t, 200, recorder.Code)
		var payload map[string]any
		assert.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
		return payload
	}
	initial := decode()
	assert.Equal(t, []any{"rp"}, initial["DisplayFields"])
	next := initial["NextMatches"].(map[string]any)
	assert.Equal(t, "Q2", next["254"])
	assert.Equal(t, "Q3", next["1114"])

	recorder := web.postHttpResponse("/setup/settings",
		"activeSettingsTab=automation&audienceRankingsOptionsPresent=1&"+
			"audienceRankingsOption=rp&audienceRankingsOption=nextMatch&audienceRankingsOption=record"+
			"&autoDurationSec=20&pauseDurationSec=3&transitionShiftDurationSec=10&shiftDurationSec=25&endgameDurationSec=30&energizedBonusThreshold=100&superchargedBonusThreshold=360&traversalBonusThreshold=50")
	assert.Equal(t, 303, recorder.Code)
	assert.Equal(t, "rp,nextMatch,record", web.arena.EventSettings.AudienceRankingsFields)
	stored, err := web.arena.Database.GetEventSettings()
	assert.NoError(t, err)
	assert.Equal(t, "rp,nextMatch,record", stored.AudienceRankingsFields)
	assert.Equal(t, []any{"rp", "nextMatch", "record"}, decode()["DisplayFields"])

	for _, invalid := range []string{
		"audienceRankingsOption=unknown",
		"audienceRankingsOption=rp&audienceRankingsOption=rp",
		"audienceRankingsOption=rp&audienceRankingsOption=nextMatch&"+
			"audienceRankingsOption=record&audienceRankingsOption=played&audienceRankingsOption=autoFuel",
	} {
		bad := web.postHttpResponse(
			"/setup/settings", "activeSettingsTab=automation&audienceRankingsOptionsPresent=1&"+invalid,
		)
		assert.Equal(t, 200, bad.Code)
		assert.Contains(t, bad.Body.String(), "statistic")
		assert.Equal(t, "rp,nextMatch,record", web.arena.EventSettings.AudienceRankingsFields)
	}

	// No selection still shows the team and rank; a presence flag differentiates
	// zero checked boxes from an older POST that omitted the field.
	recorder = web.postHttpResponse("/setup/settings",
		"activeSettingsTab=automation&audienceRankingsOptionsPresent=1"+ "&autoDurationSec=20&pauseDurationSec=3&transitionShiftDurationSec=10&shiftDurationSec=25&endgameDurationSec=30&energizedBonusThreshold=100&superchargedBonusThreshold=360&traversalBonusThreshold=50")
	assert.Equal(t, 303, recorder.Code)
	assert.Equal(t, []any{}, decode()["DisplayFields"])
	stored, err = web.arena.Database.GetEventSettings()
	assert.NoError(t, err)
	assert.Equal(t, "-", stored.AudienceRankingsFields)
}

func TestSponsorSlidesApi(t *testing.T) {
	web := setupTestWeb(t)

	slide1 := model.SponsorSlide{0, "subtitle", "line1", "line2", "image", 2, 1}
	slide2 := model.SponsorSlide{0, "Chezy Sponsaur", "Teh", "Chezy Pofs", "ejface.jpg", 54, 2}
	assert.Nil(t, web.arena.Database.CreateSponsorSlide(&slide1))
	assert.Nil(t, web.arena.Database.CreateSponsorSlide(&slide2))

	recorder := web.getHttpResponse("/api/sponsor_slides")
	assert.Equal(t, 200, recorder.Code)
	assert.Equal(t, "application/json", recorder.Header()["Content-Type"][0])
	var sponsorSlides []model.SponsorSlide
	err := json.Unmarshal([]byte(recorder.Body.String()), &sponsorSlides)
	assert.Nil(t, err)
	if assert.Equal(t, 2, len(sponsorSlides)) {
		assert.Equal(t, slide1, sponsorSlides[0])
		assert.Equal(t, slide2, sponsorSlides[1])
	}
}

func TestAlliancesApi(t *testing.T) {
	web := setupTestWeb(t)

	model.BuildTestAlliances(web.arena.Database)

	recorder := web.getHttpResponse("/api/alliances")
	assert.Equal(t, 200, recorder.Code)
	assert.Equal(t, "application/json", recorder.Header()["Content-Type"][0])
	var alliances []model.Alliance
	err := json.Unmarshal([]byte(recorder.Body.String()), &alliances)
	assert.Nil(t, err)
	if assert.Equal(t, 2, len(alliances)) {
		if assert.Equal(t, 5, len(alliances[0].TeamIds)) {
			assert.Equal(t, 254, alliances[0].TeamIds[0])
			assert.Equal(t, 469, alliances[0].TeamIds[1])
			assert.Equal(t, 2848, alliances[0].TeamIds[2])
			assert.Equal(t, 74, alliances[0].TeamIds[3])
			assert.Equal(t, 3175, alliances[0].TeamIds[4])
		}
		if assert.Equal(t, 3, len(alliances[1].TeamIds)) {
			assert.Equal(t, 1718, alliances[1].TeamIds[0])
			assert.Equal(t, 2451, alliances[1].TeamIds[1])
			assert.Equal(t, 1619, alliances[1].TeamIds[2])
		}
	}
}

func TestArenaWebsocketApi(t *testing.T) {
	web := setupTestWeb(t)

	server, wsUrl := web.startTestServer()
	defer server.Close()
	conn, _, err := gorillawebsocket.DefaultDialer.Dial(wsUrl+"/api/arena/websocket", nil)
	assert.Nil(t, err)
	defer conn.Close()
	ws := websocket.NewTestWebsocket(conn)

	// Should get a few status updates right after connection.
	readWebsocketType(t, ws, "matchTiming")
	readWebsocketType(t, ws, "matchLoad")
	readWebsocketType(t, ws, "matchTime")
}

func TestBracketSvgApiDoubleElimination(t *testing.T) {
	web := setupTestWeb(t)
	web.arena.EventSettings.PlayoffType = model.DoubleEliminationPlayoff
	tournament.CreateTestAlliances(web.arena.Database, 8)
	web.arena.CreatePlayoffTournament()

	recorder := web.getHttpResponse("/api/bracket/svg")
	assert.Equal(t, 200, recorder.Code)
	assert.Equal(t, "image/svg+xml", recorder.Header()["Content-Type"][0])
	assert.Contains(t, recorder.Body.String(), "Best-of-3")
}

// Audience-only dark bracket styling must not alter standalone or printable SVG output.
func TestBracketSvgApiDarkTheme(t *testing.T) {
	web := setupTestWeb(t)
	web.arena.EventSettings.PlayoffType = model.DoubleEliminationPlayoff
	tournament.CreateTestAlliances(web.arena.Database, 8)
	web.arena.CreatePlayoffTournament()

	dark := web.getHttpResponse("/api/bracket/svg?theme=dark")
	assert.Equal(t, 200, dark.Code)
	assert.Contains(t, dark.Body.String(), "Audience-only charcoal presentation")
	assert.Contains(t, dark.Body.String(), "#background rect {fill:#212529;stroke:#6c757d;}")
	assert.Contains(t, dark.Body.String(), ".red {fill:#c9676e;}")
	assert.Contains(t, dark.Body.String(), ".blue {fill:#6d9dcb;}")

	light := web.getHttpResponse("/api/bracket/svg")
	assert.Equal(t, 200, light.Code)
	assert.NotContains(t, light.Body.String(), "Audience-only charcoal presentation")
}

func TestBracketSvgApiFourAllianceDoubleElimination(t *testing.T) {
	web := setupTestWeb(t)
	web.arena.EventSettings.PlayoffType = model.DoubleEliminationPlayoff
	web.arena.EventSettings.NumPlayoffAlliances = 4
	tournament.CreateTestAlliances(web.arena.Database, 4)
	web.arena.CreatePlayoffTournament()

	recorder := web.getHttpResponse("/api/bracket/svg")
	assert.Equal(t, 200, recorder.Code)
	assert.Equal(t, "image/svg+xml", recorder.Header()["Content-Type"][0])
	assert.Contains(t, recorder.Body.String(), "bracket_double4")
	assert.Contains(t, recorder.Body.String(), "match_M5")
	assert.Contains(t, recorder.Body.String(), "Finals")
}
func TestAllianceStatusApi_Empty(t *testing.T) {
    web := setupTestWeb(t)

	model.BuildTestAlliances(web.arena.Database)

    // No alliances or teams are created, so all fields should be default/empty.
    recorder := web.getHttpResponse("/api/freezy/allianceStatus")
    assert.Equal(t, 200, recorder.Code)
    assert.Equal(t, "application/json", recorder.Header()["Content-Type"][0])

    expected := map[string]interface{}{
        "B1": map[string]interface{}{
            "DsConn":   nil,
            "Ethernet": false,
            "AStop":    false,
            "EStop":    false,
            "Bypass":   false,
            "Team":     nil,
			"GameData": "",
			"TeamMatchLog": interface {}(nil),
            "WifiStatus": map[string]interface{}{
                "TeamId":            float64(0),
                "RadioLinked":       false,
				"ConnectionQuality": float64(0),
                "MBits":             float64(0),
                "RxRate":            float64(0),
                "TxRate":            float64(0),
                "SignalNoiseRatio":  float64(0),
            },
        },
        "B2": map[string]interface{}{
            "DsConn":   nil,
            "Ethernet": false,
            "AStop":    false,
            "EStop":    false,
            "Bypass":   false,
            "Team":     nil,
			"GameData": "",
			"TeamMatchLog": interface {}(nil),
            "WifiStatus": map[string]interface{}{
                "TeamId":            float64(0),
                "RadioLinked":       false,
				"ConnectionQuality": float64(0),
                "MBits":             float64(0),
                "RxRate":            float64(0),
                "TxRate":            float64(0),
                "SignalNoiseRatio":  float64(0),
            },
        },
        "B3": map[string]interface{}{
            "DsConn":   nil,
            "Ethernet": false,
            "AStop":    false,
            "EStop":    false,
            "Bypass":   false,
            "Team":     nil,
			"GameData": "",
			"TeamMatchLog": interface {}(nil),
            "WifiStatus": map[string]interface{}{
                "TeamId":            float64(0),
                "RadioLinked":       false,
				"ConnectionQuality": float64(0),
                "MBits":             float64(0),
                "RxRate":            float64(0),
                "TxRate":            float64(0),
                "SignalNoiseRatio":  float64(0),
            },
        },
        "R1": map[string]interface{}{
            "DsConn":   nil,
            "Ethernet": false,
            "AStop":    false,
            "EStop":    false,
            "Bypass":   false,
            "Team":     nil,
			"GameData": "",
			"TeamMatchLog": interface {}(nil),
            "WifiStatus": map[string]interface{}{
                "TeamId":            float64(0),
                "RadioLinked":       false,
				"ConnectionQuality": float64(0),
                "MBits":             float64(0),
                "RxRate":            float64(0),
                "TxRate":            float64(0),
                "SignalNoiseRatio":  float64(0),
            },
        },
        "R2": map[string]interface{}{
            "DsConn":   nil,
            "Ethernet": false,
            "AStop":    false,
            "EStop":    false,
            "Bypass":   false,
            "Team":     nil,
			"GameData": "",
			"TeamMatchLog": interface {}(nil),
            "WifiStatus": map[string]interface{}{
                "TeamId":            float64(0),
                "RadioLinked":       false,
				"ConnectionQuality": float64(0),
                "MBits":             float64(0),
                "RxRate":            float64(0),
                "TxRate":            float64(0),
                "SignalNoiseRatio":  float64(0),
            },
        },
        "R3": map[string]interface{}{
            "DsConn":   nil,
            "Ethernet": false,
            "AStop":    false,
            "EStop":    false,
            "Bypass":   false,
            "Team":     nil,
			"GameData": "",
			"TeamMatchLog": interface {}(nil),
            "WifiStatus": map[string]interface{}{
                "TeamId":            float64(0),
                "RadioLinked":       false,
				"ConnectionQuality": float64(0),
                "MBits":             float64(0),
                "RxRate":            float64(0),
                "TxRate":            float64(0),
                "SignalNoiseRatio":  float64(0),
            },
        },
    }

    var actual map[string]interface{}
    err := json.Unmarshal([]byte(recorder.Body.String()), &actual)
    assert.Nil(t, err)
    assert.Equal(t, expected, actual)
}
