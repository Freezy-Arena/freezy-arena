// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)

package web

import (
	"bytes"
	"encoding/json"
	"github.com/Team254/cheesy-arena/game"
	"github.com/Team254/cheesy-arena/model"
	"github.com/Team254/cheesy-arena/tournament"
	"github.com/Team254/cheesy-arena/websocket"
	gorillawebsocket "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
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

func TestRemoteMatchApi(t *testing.T) {
	web := setupTestWeb(t)

	match := model.Match{
		Type:      model.Practice,
		ShortName: "P1",
		Time:      time.Unix(0, 0),
		Red1:      1,
		Red2:      2,
		Red3:      3,
		Blue1:     4,
		Blue2:     5,
		Blue3:     6,
	}
	assert.Nil(t, web.arena.Database.CreateMatch(&match))

	recorder := web.getHttpResponse("/api/remote/matches/1")
	assert.Equal(t, 200, recorder.Code)
	assert.Equal(t, "application/json", recorder.Header()["Content-Type"][0])
	var matchData MatchWithResult
	err := json.Unmarshal([]byte(recorder.Body.String()), &matchData)
	assert.Nil(t, err)
	assert.Equal(t, match.Id, matchData.Match.Id)
	assert.Equal(t, match.ShortName, matchData.Match.ShortName)
	assert.Nil(t, matchData.Result)
}

func TestRemoteMatchResultPostApi(t *testing.T) {
	web := setupTestWeb(t)

	match := model.Match{
		Type:      model.Practice,
		ShortName: "P1",
		Time:      time.Unix(0, 0),
		Red1:      1,
		Red2:      2,
		Red3:      3,
		Blue1:     4,
		Blue2:     5,
		Blue3:     6,
	}
	assert.Nil(t, web.arena.Database.CreateMatch(&match))
	matchResult := model.BuildTestMatchResult(match.Id, 0)
	matchResult.MatchType = match.Type
	requestBody, err := json.Marshal(matchResult)
	assert.Nil(t, err)

	recorder := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/remote/matches/1/result", bytes.NewReader(requestBody))
	web.newHandler().ServeHTTP(recorder, req)
	assert.Equal(t, 200, recorder.Code)
	assert.Equal(t, "application/json", recorder.Header()["Content-Type"][0])

	var matchData MatchWithResult
	err = json.Unmarshal([]byte(recorder.Body.String()), &matchData)
	assert.Nil(t, err)
	if assert.NotNil(t, matchData.Result) {
		assert.Equal(t, match.Id, matchData.Result.MatchId)
		assert.Equal(t, 1, matchData.Result.PlayNumber)
		assert.NotNil(t, matchData.Result.RedSummary)
		assert.NotNil(t, matchData.Result.BlueSummary)
	}

	savedResult, err := web.arena.Database.GetMatchResultForMatch(match.Id)
	assert.Nil(t, err)
	if assert.NotNil(t, savedResult) {
		assert.Equal(t, 1, savedResult.PlayNumber)
		assert.Equal(t, match.Type, savedResult.MatchType)
	}
	savedMatch, err := web.arena.Database.GetMatchById(match.Id)
	assert.Nil(t, err)
	assert.True(t, savedMatch.IsComplete())
}

func TestRemotePrimaryLoadMatchPostApiRequiresSecondaryMode(t *testing.T) {
	web := setupTestWeb(t)

	recorder := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/remote/primary/matches/1/load", nil)
	web.newHandler().ServeHTTP(recorder, req)

	assert.Equal(t, 400, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Secondary arena mode is not enabled.")
}

func TestRemotePrimaryLoadMatchPostApi(t *testing.T) {
	primaryMatch := model.Match{
		Id:        47,
		Type:      model.Qualification,
		ShortName: "Q47",
		LongName:  "Qualification 47",
		Red1:      101,
		Red2:      102,
		Red3:      103,
		Blue1:     201,
		Blue2:     202,
		Blue3:     203,
	}
	primaryServer := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "GET", r.Method)
				assert.Equal(t, "/api/remote/matches/47", r.URL.Path)
				assert.Nil(t, json.NewEncoder(w).Encode(MatchWithResult{Match: primaryMatch}))
			},
		),
	)
	defer primaryServer.Close()

	web := setupTestWeb(t)
	web.arena.EventSettings.SecondaryArenaEnabled = true
	web.arena.EventSettings.PrimaryArenaUrl = primaryServer.URL

	recorder := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/remote/primary/matches/47/load", nil)
	web.newHandler().ServeHTTP(recorder, req)

	assert.Equal(t, 200, recorder.Code)
	assert.Equal(t, "application/json", recorder.Header()["Content-Type"][0])
	assert.Equal(t, primaryMatch.Id, web.arena.CurrentMatch.Id)
	assert.Equal(t, model.Test, web.arena.CurrentMatch.Type)
	assert.Equal(t, primaryMatch.ShortName, web.arena.CurrentMatch.ShortName)
	assert.Equal(t, primaryMatch.Red1, web.arena.AllianceStations["R1"].Team.Id)
	assert.Equal(t, primaryMatch.Blue3, web.arena.AllianceStations["B3"].Team.Id)
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
			"DsConn":       nil,
			"Ethernet":     false,
			"AStop":        false,
			"EStop":        false,
			"Bypass":       false,
			"Team":         nil,
			"GameData":     "",
			"TeamMatchLog": interface{}(nil),
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
			"DsConn":       nil,
			"Ethernet":     false,
			"AStop":        false,
			"EStop":        false,
			"Bypass":       false,
			"Team":         nil,
			"GameData":     "",
			"TeamMatchLog": interface{}(nil),
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
			"DsConn":       nil,
			"Ethernet":     false,
			"AStop":        false,
			"EStop":        false,
			"Bypass":       false,
			"Team":         nil,
			"GameData":     "",
			"TeamMatchLog": interface{}(nil),
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
			"DsConn":       nil,
			"Ethernet":     false,
			"AStop":        false,
			"EStop":        false,
			"Bypass":       false,
			"Team":         nil,
			"GameData":     "",
			"TeamMatchLog": interface{}(nil),
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
			"DsConn":       nil,
			"Ethernet":     false,
			"AStop":        false,
			"EStop":        false,
			"Bypass":       false,
			"Team":         nil,
			"GameData":     "",
			"TeamMatchLog": interface{}(nil),
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
			"DsConn":       nil,
			"Ethernet":     false,
			"AStop":        false,
			"EStop":        false,
			"Bypass":       false,
			"Team":         nil,
			"GameData":     "",
			"TeamMatchLog": interface{}(nil),
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
