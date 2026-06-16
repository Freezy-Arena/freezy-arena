// Copyright 2026 Team 254. All Rights Reserved.

package remote

import (
	"encoding/json"
	"github.com/Team254/cheesy-arena/model"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPrimaryClientGetMatch(t *testing.T) {
	match := model.Match{Id: 1, Type: model.Practice, ShortName: "P1", Red1: 101, Blue1: 201}
	result := model.BuildTestMatchResult(match.Id, 1)
	response := MatchWithResult{
		Match: match,
		Result: &MatchResultWithSummary{
			MatchResult: *result,
			RedSummary:  result.RedScoreSummary(),
			BlueSummary: result.BlueScoreSummary(),
		},
	}

	server := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "GET", r.Method)
				assert.Equal(t, "/api/remote/matches/1", r.URL.Path)
				assert.Nil(t, json.NewEncoder(w).Encode(response))
			},
		),
	)
	defer server.Close()

	client := NewPrimaryClient(server.URL)
	matchWithResult, err := client.GetMatch(1)
	assert.Nil(t, err)
	assert.Equal(t, match.Id, matchWithResult.Id)
	assert.Equal(t, match.ShortName, matchWithResult.ShortName)
	if assert.NotNil(t, matchWithResult.Result) {
		assert.Equal(t, result.MatchId, matchWithResult.Result.MatchId)
		assert.NotNil(t, matchWithResult.Result.RedSummary)
		assert.NotNil(t, matchWithResult.Result.BlueSummary)
	}
}

func TestPrimaryClientPostMatchResult(t *testing.T) {
	match := model.Match{Id: 2, Type: model.Practice, ShortName: "P2", Red1: 102, Blue1: 202}
	result := model.BuildTestMatchResult(match.Id, 1)
	response := MatchWithResult{
		Match: match,
		Result: &MatchResultWithSummary{
			MatchResult: *result,
			RedSummary:  result.RedScoreSummary(),
			BlueSummary: result.BlueScoreSummary(),
		},
	}

	server := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/api/remote/matches/2/result", r.URL.Path)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				var postedResult model.MatchResult
				assert.Nil(t, json.NewDecoder(r.Body).Decode(&postedResult))
				assert.Equal(t, result.MatchId, postedResult.MatchId)
				assert.Equal(t, result.PlayNumber, postedResult.PlayNumber)
				assert.Nil(t, json.NewEncoder(w).Encode(response))
			},
		),
	)
	defer server.Close()

	client := NewPrimaryClient(server.URL + "/")
	matchWithResult, err := client.PostMatchResult(match.Id, result)
	assert.Nil(t, err)
	assert.Equal(t, match.Id, matchWithResult.Id)
	if assert.NotNil(t, matchWithResult.Result) {
		assert.Equal(t, result.MatchId, matchWithResult.Result.MatchId)
	}
}

func TestPrimaryClientReturnsServerError(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "no such match", http.StatusNotFound)
			},
		),
	)
	defer server.Close()

	client := NewPrimaryClient(server.URL)
	_, err := client.GetMatch(999)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "primary returned 404")
	assert.Contains(t, err.Error(), "no such match")
}
