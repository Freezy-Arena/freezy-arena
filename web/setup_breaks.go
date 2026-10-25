// Copyright 2024 Team 254. All Rights Reserved.
// Author: Justin Fischer <justin@justfishin99.xyz>
//
// Web routes for configuring playoff break descriptions, start times, and durations.

package web

import (
	"fmt"
	"github.com/Team254/cheesy-arena/field"
	"github.com/Team254/cheesy-arena/model"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const scheduledBreakTimeFormat = "2006-01-02T15:04"

type scheduledBreakEditorRow struct {
	model.ScheduledBreak
	StartTimeInput  string
	EndTimeDisplay  string
	DurationMinutes int
	NextMatchName   string
}

// Shows the breaks configuration page.
func (web *Web) breaksGetHandler(w http.ResponseWriter, r *http.Request) {
	if !web.userIsAdmin(w, r) {
		return
	}
	web.renderBreaks(w, "", http.StatusOK)
}

func (web *Web) renderBreaks(w http.ResponseWriter, errorMessage string, statusCode int) {
	breaks, err := web.arena.Database.GetScheduledBreaksByMatchType(model.Playoff)
	if err != nil {
		handleWebErr(w, err)
		return
	}
	matches, err := web.arena.Database.GetMatchesByType(model.Playoff, true)
	if err != nil {
		handleWebErr(w, err)
		return
	}
	matchNames := make(map[int]string, len(matches))
	for _, match := range matches {
		matchNames[match.TypeOrder] = match.LongName
	}
	rows := make([]scheduledBreakEditorRow, 0, len(breaks))
	for _, scheduledBreak := range breaks {
		nextMatchName := matchNames[scheduledBreak.TypeOrderBefore]
		if nextMatchName == "" {
			nextMatchName = fmt.Sprintf("Match %d", scheduledBreak.TypeOrderBefore)
		}
		rows = append(rows, scheduledBreakEditorRow{
			ScheduledBreak:  scheduledBreak,
			StartTimeInput:  scheduledBreak.Time.Local().Format(scheduledBreakTimeFormat),
			EndTimeDisplay:  scheduledBreak.Time.Add(time.Duration(scheduledBreak.DurationSec) * time.Second).Local().Format("Mon Jan 2, 3:04 PM"),
			DurationMinutes: scheduledBreak.DurationSec / 60,
			NextMatchName:   nextMatchName,
		})
	}

	tmpl, err := web.parseFiles("templates/setup_breaks.html", "templates/base.html")
	if err != nil {
		handleWebErr(w, err)
		return
	}
	data := struct {
		*model.EventSettings
		ScheduledBreaks []scheduledBreakEditorRow
		ErrorMessage    string
	}{web.arena.EventSettings, rows, errorMessage}
	w.WriteHeader(statusCode)
	if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
		handleWebErr(w, err)
	}
}

// Saves a break and moves later unplayed playoff matches and breaks whenever its
// scheduled end time changes. The on-field countdown is started by match flow,
// not by the configured wall-clock start time.
func (web *Web) breaksPostHandler(w http.ResponseWriter, r *http.Request) {
	if !web.userIsAdmin(w, r) {
		return
	}
	reject := func(message string) {
		web.renderBreaks(w, message, http.StatusBadRequest)
	}

	id, err := strconv.Atoi(r.PostFormValue("id"))
	if err != nil || id <= 0 {
		reject("Invalid break ID.")
		return
	}
	scheduledBreak, err := web.arena.Database.GetScheduledBreakById(id)
	if err != nil {
		handleWebErr(w, err)
		return
	}
	if scheduledBreak == nil || scheduledBreak.MatchType != model.Playoff {
		reject("The selected playoff break does not exist.")
		return
	}

	description := strings.TrimSpace(r.PostFormValue("description"))
	if description == "" {
		reject("A break description is required.")
		return
	}

	newStartTime := scheduledBreak.Time
	if _, provided := r.PostForm["startTime"]; provided {
		input := r.PostFormValue("startTime")
		newStartTime, err = time.ParseInLocation(scheduledBreakTimeFormat, input, time.Local)
		if err != nil {
			reject("Invalid break start time.")
			return
		}
		// Keep existing seconds if the browser supplied the unmodified minute.
		if input == scheduledBreak.Time.Local().Format(scheduledBreakTimeFormat) {
			newStartTime = scheduledBreak.Time
		}
	}

	newDurationSec := scheduledBreak.DurationSec
	if _, provided := r.PostForm["durationMinutes"]; provided {
		minutes, parseErr := strconv.Atoi(r.PostFormValue("durationMinutes"))
		if parseErr != nil || minutes < 1 || minutes > 1440 {
			reject("Break duration must be between 1 and 1440 whole minutes.")
			return
		}
		newDurationSec = minutes * 60
	}

	changedTiming := !newStartTime.Equal(scheduledBreak.Time) || newDurationSec != scheduledBreak.DurationSec
	endTimeDelta := newStartTime.Add(time.Duration(newDurationSec) * time.Second).Sub(
		scheduledBreak.Time.Add(time.Duration(scheduledBreak.DurationSec) * time.Second),
	)

	var matchesToUpdate []*model.Match
	var breaksToUpdate []*model.ScheduledBreak
	if changedTiming {
		if web.arena.CurrentMatch != nil &&
			web.arena.CurrentMatch.Type == model.Playoff &&
			web.arena.CurrentMatch.TypeOrder == scheduledBreak.TypeOrderBefore &&
			(web.arena.MatchState == field.TimeoutActive || web.arena.MatchState == field.PostTimeout) {
			reject("Cannot change a break while its countdown is active.")
			return
		}

		matches, getErr := web.arena.Database.GetMatchesByType(model.Playoff, true)
		if getErr != nil {
			handleWebErr(w, getErr)
			return
		}
		for i := range matches {
			match := &matches[i]
			if match.TypeOrder < scheduledBreak.TypeOrderBefore {
				if !newStartTime.After(match.Time) {
					reject(fmt.Sprintf("The break must start after %s is scheduled to begin.", match.LongName))
					return
				}
				continue
			}
			if !match.StartedAt.IsZero() || match.IsComplete() {
				reject("Cannot change a break once a following playoff match has started or finished.")
				return
			}
			if endTimeDelta != 0 {
				match.Time = match.Time.Add(endTimeDelta)
				matchesToUpdate = append(matchesToUpdate, match)
			}
		}

		breaks, getErr := web.arena.Database.GetScheduledBreaksByMatchType(model.Playoff)
		if getErr != nil {
			handleWebErr(w, getErr)
			return
		}
		for i := range breaks {
			other := &breaks[i]
			if other.Id == scheduledBreak.Id {
				continue
			}
			if other.TypeOrderBefore < scheduledBreak.TypeOrderBefore {
				otherEnd := other.Time.Add(time.Duration(other.DurationSec) * time.Second)
				if newStartTime.Before(otherEnd) {
					reject("The break cannot start before an earlier scheduled break ends.")
					return
				}
			} else if endTimeDelta != 0 {
				other.Time = other.Time.Add(endTimeDelta)
				breaksToUpdate = append(breaksToUpdate, other)
			}
		}
	}

	// Validate every downstream event before persisting any of the changes.
	for _, match := range matchesToUpdate {
		if err := web.arena.Database.UpdateMatch(match); err != nil {
			handleWebErr(w, err)
			return
		}
	}
	for _, other := range breaksToUpdate {
		if err := web.arena.Database.UpdateScheduledBreak(other); err != nil {
			handleWebErr(w, err)
			return
		}
	}
	scheduledBreak.Description = description
	scheduledBreak.Time = newStartTime
	scheduledBreak.DurationSec = newDurationSec
	if err := web.arena.Database.UpdateScheduledBreak(scheduledBreak); err != nil {
		handleWebErr(w, err)
		return
	}
	if changedTiming {
		for _, match := range matchesToUpdate {
			if web.arena.CurrentMatch != nil && web.arena.CurrentMatch.Id == match.Id {
				web.arena.CurrentMatch.Time = match.Time
			}
		}
		web.arena.MatchListNotifier.Notify()
		web.arena.MatchLoadNotifier.Notify()
		web.arena.EventStatusNotifier.Notify()
	}
	http.Redirect(w, r, "/setup/breaks", http.StatusSeeOther)
}
