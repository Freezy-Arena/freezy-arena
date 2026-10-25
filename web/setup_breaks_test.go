// Copyright 2024 Team 254. All Rights Reserved.
// Author: Justin Fischer <justin@justfishin99.xyz>

package web

import (
	"fmt"
	"github.com/Team254/cheesy-arena/model"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/url"
	"testing"
	"time"
)

func TestSetupBreaks(t *testing.T) {
	web := setupTestWeb(t)
	break1 := &model.ScheduledBreak{MatchType: model.Playoff, TypeOrderBefore: 4, Time: time.Unix(500, 0).UTC(), DurationSec: 900, Description: "Field Break 1"}
	break2 := &model.ScheduledBreak{MatchType: model.Playoff, TypeOrderBefore: 4, Time: time.Unix(500, 0).UTC(), DurationSec: 900, Description: "Field Break 2"}
	assert.NoError(t, web.arena.Database.CreateScheduledBreak(break1))
	assert.NoError(t, web.arena.Database.CreateScheduledBreak(break2))

	recorder := web.getHttpResponse("/setup/breaks")
	assert.Equal(t, 200, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Field Break 1")
	assert.Contains(t, recorder.Body.String(), "Field Break 2")
	assert.Contains(t, recorder.Body.String(), "durationMinutes")
	assert.Contains(t, recorder.Body.String(), "startTime")

	// Preserve support for the original description-only form submission.
	recorder = web.postHttpResponse("/setup/breaks", fmt.Sprintf("id=%d&description=Award+Break+3", break2.Id))
	assert.Equal(t, 303, recorder.Code)
	recorder = web.getHttpResponse("/setup/breaks")
	assert.Equal(t, 200, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Field Break 1")
	assert.NotContains(t, recorder.Body.String(), "Field Break 2")
	assert.Contains(t, recorder.Body.String(), "Award Break 3")
}

func TestSetupBreaksReschedulesFollowingEvents(t *testing.T) {
	web := setupTestWeb(t)
	start := time.Date(2026, 10, 25, 10, 0, 0, 0, time.Local)
	match1 := &model.Match{Type: model.Playoff, TypeOrder: 1, LongName: "Match 1", Time: start}
	match2 := &model.Match{Type: model.Playoff, TypeOrder: 2, LongName: "Match 2", Time: start.Add(10 * time.Minute)}
	match3 := &model.Match{Type: model.Playoff, TypeOrder: 3, LongName: "Match 3", Time: start.Add(30 * time.Minute)}
	match4 := &model.Match{Type: model.Playoff, TypeOrder: 4, LongName: "Match 4", Time: start.Add(45 * time.Minute)}
	for _, match := range []*model.Match{match1, match2, match3, match4} {
		assert.NoError(t, web.arena.Database.CreateMatch(match))
	}
	break1 := &model.ScheduledBreak{MatchType: model.Playoff, TypeOrderBefore: 3, Time: start.Add(20 * time.Minute), DurationSec: 600, Description: "Field Break"}
	break2 := &model.ScheduledBreak{MatchType: model.Playoff, TypeOrderBefore: 4, Time: start.Add(40 * time.Minute), DurationSec: 300, Description: "Awards"}
	assert.NoError(t, web.arena.Database.CreateScheduledBreak(break1))
	assert.NoError(t, web.arena.Database.CreateScheduledBreak(break2))

	values := url.Values{
		"id":              {fmt.Sprint(break1.Id)},
		"description":     {"Updated field break"},
		"startTime":       {start.Add(25 * time.Minute).Format(scheduledBreakTimeFormat)},
		"durationMinutes": {"15"},
	}
	recorder := web.postHttpResponse("/setup/breaks", values.Encode())
	assert.Equal(t, http.StatusSeeOther, recorder.Code)

	gotBreak, err := web.arena.Database.GetScheduledBreakById(break1.Id)
	assert.NoError(t, err)
	assert.Equal(t, start.Add(25 * time.Minute), gotBreak.Time)
	assert.Equal(t, 900, gotBreak.DurationSec)
	assert.Equal(t, "Updated field break", gotBreak.Description)

	gotNextBreak, err := web.arena.Database.GetScheduledBreakById(break2.Id)
	assert.NoError(t, err)
	assert.Equal(t, start.Add(50 * time.Minute), gotNextBreak.Time)

	gotMatch2, err := web.arena.Database.GetMatchById(match2.Id)
	assert.NoError(t, err)
	assert.Equal(t, match2.Time, gotMatch2.Time)
	gotMatch3, err := web.arena.Database.GetMatchById(match3.Id)
	assert.NoError(t, err)
	assert.Equal(t, start.Add(40 * time.Minute), gotMatch3.Time)
	gotMatch4, err := web.arena.Database.GetMatchById(match4.Id)
	assert.NoError(t, err)
	assert.Equal(t, start.Add(55 * time.Minute), gotMatch4.Time)
}

func TestSetupBreaksRejectsUnsafeTimingChanges(t *testing.T) {
	web := setupTestWeb(t)
	start := time.Date(2026, 10, 25, 10, 0, 0, 0, time.Local)
	previous := &model.Match{Type: model.Playoff, TypeOrder: 1, LongName: "Previous Match", Time: start}
	next := &model.Match{Type: model.Playoff, TypeOrder: 2, LongName: "Next Match", Time: start.Add(20 * time.Minute)}
	assert.NoError(t, web.arena.Database.CreateMatch(previous))
	assert.NoError(t, web.arena.Database.CreateMatch(next))
	pause := &model.ScheduledBreak{MatchType: model.Playoff, TypeOrderBefore: 2, Time: start.Add(10 * time.Minute), DurationSec: 600, Description: "Field Break"}
	assert.NoError(t, web.arena.Database.CreateScheduledBreak(pause))

	for _, tc := range []struct {
		name       string
		start      string
		duration   string
		wantError  string
	}{
		{"invalid start", "not-a-time", "10", "Invalid break start time"},
		{"zero duration", start.Add(10 * time.Minute).Format(scheduledBreakTimeFormat), "0", "Break duration must"},
		{"bad duration", start.Add(10 * time.Minute).Format(scheduledBreakTimeFormat), "abc", "Break duration must"},
		{"too long", start.Add(10 * time.Minute).Format(scheduledBreakTimeFormat), "1441", "Break duration must"},
		{"before previous match", start.Format(scheduledBreakTimeFormat), "10", "must start after"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			post := url.Values{"id": {fmt.Sprint(pause.Id)}, "description": {"Field Break"}, "startTime": {tc.start}, "durationMinutes": {tc.duration}}
			recorder := web.postHttpResponse("/setup/breaks", post.Encode())
			assert.Equal(t, http.StatusBadRequest, recorder.Code)
			assert.Contains(t, recorder.Body.String(), tc.wantError)
			got, err := web.arena.Database.GetScheduledBreakById(pause.Id)
			assert.NoError(t, err)
			assert.Equal(t, pause.Time, got.Time)
			assert.Equal(t, pause.DurationSec, got.DurationSec)
		})
	}

	next.StartedAt = start.Add(21 * time.Minute)
	assert.NoError(t, web.arena.Database.UpdateMatch(next))
	post := url.Values{
		"id": {fmt.Sprint(pause.Id)}, "description": {"Field Break"},
		"startTime": {start.Add(15 * time.Minute).Format(scheduledBreakTimeFormat)}, "durationMinutes": {"12"},
	}
	recorder := web.postHttpResponse("/setup/breaks", post.Encode())
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "following playoff match has started")
}

func TestSetupBreaksRejectsOtherBreakTypes(t *testing.T) {
	web := setupTestWeb(t)
	pause := &model.ScheduledBreak{MatchType: model.Qualification, TypeOrderBefore: 4, Time: time.Now(), DurationSec: 600, Description: "Lunch"}
	assert.NoError(t, web.arena.Database.CreateScheduledBreak(pause))
	recorder := web.postHttpResponse("/setup/breaks", fmt.Sprintf("id=%d&description=Changed", pause.Id))
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	got, err := web.arena.Database.GetScheduledBreakById(pause.Id)
	assert.NoError(t, err)
	assert.Equal(t, "Lunch", got.Description)
}
