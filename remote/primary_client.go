// Copyright 2026 Team 254. All Rights Reserved.
//
// Client for secondary field instances to exchange match data with a primary Freezy Arena instance.

package remote

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/Team254/cheesy-arena/game"
	"github.com/Team254/cheesy-arena/model"
	"io"
	"net/http"
	"strings"
	"time"
)

const primaryClientTimeout = 5 * time.Second

type MatchResultWithSummary struct {
	model.MatchResult
	RedSummary  *game.ScoreSummary
	BlueSummary *game.ScoreSummary
}

type MatchWithResult struct {
	model.Match
	Result *MatchResultWithSummary
}

type PrimaryClient struct {
	BaseUrl    string
	httpClient *http.Client
}

func NewPrimaryClient(baseUrl string) *PrimaryClient {
	return &PrimaryClient{
		BaseUrl: strings.TrimRight(baseUrl, "/"),
		httpClient: &http.Client{
			Timeout: primaryClientTimeout,
		},
	}
}

func (client *PrimaryClient) GetMatch(matchId int) (*MatchWithResult, error) {
	var matchWithResult MatchWithResult
	err := client.doJsonRequest("GET", fmt.Sprintf("/api/remote/matches/%d", matchId), nil, &matchWithResult)
	return &matchWithResult, err
}

func (client *PrimaryClient) PostMatchResult(matchId int, matchResult *model.MatchResult) (*MatchWithResult, error) {
	var matchWithResult MatchWithResult
	err := client.doJsonRequest(
		"POST",
		fmt.Sprintf("/api/remote/matches/%d/result", matchId),
		matchResult,
		&matchWithResult,
	)
	return &matchWithResult, err
}

func (client *PrimaryClient) doJsonRequest(method string, path string, requestBody any, responseBody any) error {
	var body io.Reader
	if requestBody != nil {
		jsonBody, err := json.Marshal(requestBody)
		if err != nil {
			return err
		}
		body = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, client.BaseUrl+path, body)
	if err != nil {
		return err
	}
	if requestBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("primary returned %d: %s", resp.StatusCode, string(respBytes))
	}

	return json.Unmarshal(respBytes, responseBody)
}
