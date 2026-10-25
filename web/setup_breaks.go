// Copyright 2026 Freezy Arena contributors.
// Author: Justin Fischer <justin@justfishin99.xyz>
//
// Configure the playoff breaks that occur after selected matches.

package web

import (
	"encoding/json"
	"fmt"
	"github.com/Team254/cheesy-arena/field"
	"github.com/Team254/cheesy-arena/model"
	"github.com/Team254/cheesy-arena/playoff"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

type breakEditorRow struct {
	AfterMatchOrder int
	DurationMinutes int
	Description     string
}

func (web *Web) breaksGetHandler(w http.ResponseWriter, r *http.Request) {
	if !web.userIsAdmin(w, r) {
		return
	}
	web.renderBreaks(w, "", http.StatusOK)
}

// Breaks can be configured before or after playoff matches are generated.
// Before creation, the bracket's built-in breaks are shown as defaults.
func (web *Web) renderBreaks(w http.ResponseWriter, errorMessage string, statusCode int) {
	matches, err := web.arena.Database.GetMatchesByType(model.Playoff, true)
	if err != nil {
		handleWebErr(w, err)
		return
	}
	rows := make([]breakEditorRow, 0)
	if len(matches) == 0 {
		for _, item := range web.arena.PlayoffTournament.ConfiguredBreaks() {
			rows = append(rows, breakEditorRow{
				AfterMatchOrder: item.AfterMatchOrder,
				DurationMinutes: item.DurationSec / 60,
				Description: item.Description,
			})
		}
	} else {
		slotsByBefore := make(map[int]playoff.BreakSlot)
		for _, slot := range web.arena.PlayoffTournament.BreakSlots() {
			slotsByBefore[slot.BeforeMatchOrder] = slot
		}
		breaks, err := web.arena.Database.GetScheduledBreaksByMatchType(model.Playoff)
		if err != nil {
			handleWebErr(w, err)
			return
		}
		for _, item := range breaks {
			if slot, ok := slotsByBefore[item.TypeOrderBefore]; ok {
				rows = append(rows, breakEditorRow{
					AfterMatchOrder: slot.AfterMatchOrder,
					DurationMinutes: item.DurationSec / 60,
					Description: item.Description,
				})
			}
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].AfterMatchOrder < rows[j].AfterMatchOrder })

	tmpl, err := web.parseFiles("templates/setup_breaks.html", "templates/base.html")
	if err != nil {
		handleWebErr(w, err)
		return
	}
	data := struct {
		*model.EventSettings
		ScheduledBreaks []breakEditorRow
		MatchSlots      []playoff.BreakSlot
		MatchesCreated  bool
		ErrorMessage    string
	}{
		web.arena.EventSettings,
		rows,
		web.arena.PlayoffTournament.BreakSlots(),
		len(matches) > 0,
		errorMessage,
	}
	w.WriteHeader(statusCode)
	if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
		handleWebErr(w, err)
	}
}

func (web *Web) breaksPostHandler(w http.ResponseWriter, r *http.Request) {
	if !web.userIsAdmin(w, r) {
		return
	}
	reject := func(message string) {
		web.renderBreaks(w, message, http.StatusBadRequest)
	}
	if err := r.ParseForm(); err != nil {
		reject("Invalid break schedule form.")
		return
	}

	// Validate against a fresh copy of the configured bracket. It retains the
	// hard-coded defaults, even if the in-memory tournament is customized.
	defaultTournament, err := playoff.NewPlayoffTournament(
		web.arena.EventSettings.PlayoffType, web.arena.EventSettings.NumPlayoffAlliances,
	)
	if err != nil {
		handleWebErr(w, err)
		return
	}

	var desired []model.PlayoffBreakSetting
	resetDefaults := r.PostFormValue("action") == "defaults"
	if resetDefaults {
		desired = defaultTournament.ConfiguredBreaks()
	} else {
		afterValues := r.PostForm["afterMatchOrder"]
		durationValues := r.PostForm["durationMinutes"]
		descriptionValues := r.PostForm["description"]
		if len(afterValues) != len(durationValues) || len(afterValues) != len(descriptionValues) ||
			len(afterValues) > len(defaultTournament.BreakSlots()) {
			reject("Incomplete or invalid break schedule.")
			return
		}
		desired = make([]model.PlayoffBreakSetting, 0, len(afterValues))
		for i := range afterValues {
			after, afterErr := strconv.Atoi(afterValues[i])
			minutes, durationErr := strconv.Atoi(durationValues[i])
			if afterErr != nil || durationErr != nil || minutes < 1 || minutes > 1440 {
				reject("Choose a valid match and a break duration between 1 and 1440 minutes.")
				return
			}
			desired = append(desired, model.PlayoffBreakSetting{
				AfterMatchOrder: after,
				DurationSec: minutes * 60,
				Description: strings.TrimSpace(descriptionValues[i]),
			})
		}
	}
	if err := defaultTournament.ConfigureBreaks(desired); err != nil {
		reject(err.Error())
		return
	}
	sort.Slice(desired, func(i, j int) bool { return desired[i].AfterMatchOrder < desired[j].AfterMatchOrder })

	matches, err := web.arena.Database.GetMatchesByType(model.Playoff, true)
	if err != nil {
		handleWebErr(w, err)
		return
	}
	if len(matches) > 0 {
		if err := web.applyPlayoffBreakSchedule(matches, desired); err != nil {
			reject(err.Error())
			return
		}
	}

	// Store explicit [] for a user-selected empty schedule; an empty string
	// means follow the original hard-coded defaults.
	newJSON := ""
	if !resetDefaults {
		bytes, err := json.Marshal(desired)
		if err != nil {
			handleWebErr(w, err)
			return
		}
		newJSON = string(bytes)
	}
	previousJSON := web.arena.EventSettings.PlayoffBreaksJSON
	web.arena.EventSettings.PlayoffBreaksJSON = newJSON
	if err := web.arena.Database.UpdateEventSettings(web.arena.EventSettings); err != nil {
		web.arena.EventSettings.PlayoffBreaksJSON = previousJSON
		handleWebErr(w, err)
		return
	}
	if err := web.arena.PlayoffTournament.ConfigureBreaks(desired); err != nil {
		handleWebErr(w, err)
		return
	}
	http.Redirect(w, r, "/setup/breaks", http.StatusSeeOther)
}

// Update match times using the difference between the old and new breaks at
// each boundary. This preserves custom match-to-match spacing outside breaks.
func (web *Web) applyPlayoffBreakSchedule(
	matches []model.Match, desired []model.PlayoffBreakSetting,
) error {
	existing, err := web.arena.Database.GetScheduledBreaksByMatchType(model.Playoff)
	if err != nil {
		return err
	}
	beforeByAfter := make(map[int]int)
	afterByBefore := make(map[int]int)
	for _, slot := range web.arena.PlayoffTournament.BreakSlots() {
		beforeByAfter[slot.AfterMatchOrder] = slot.BeforeMatchOrder
		afterByBefore[slot.BeforeMatchOrder] = slot.AfterMatchOrder
	}
	oldByBefore := make(map[int]model.ScheduledBreak)
	for _, item := range existing {
		oldByBefore[item.TypeOrderBefore] = item
	}
	newByBefore := make(map[int]model.PlayoffBreakSetting)
	for _, item := range desired {
		newByBefore[beforeByAfter[item.AfterMatchOrder]] = item
	}

	lastStartedOrder := 0
	for _, match := range matches {
		if !match.StartedAt.IsZero() || match.IsComplete() {
			if match.TypeOrder > lastStartedOrder {
				lastStartedOrder = match.TypeOrder
			}
		}
	}
	for before, old := range oldByBefore {
		newItem, ok := newByBefore[before]
		if !ok || old.DurationSec != newItem.DurationSec || old.Description != newItem.Description {
			if afterByBefore[before] <= lastStartedOrder {
				return fmt.Errorf("cannot change a break after a match that has already started")
			}
		}
	}
	for before, item := range newByBefore {
		old, ok := oldByBefore[before]
		if !ok || old.DurationSec != item.DurationSec || old.Description != item.Description {
			if afterByBefore[before] <= lastStartedOrder {
				return fmt.Errorf("cannot add or change a break after a match that has already started")
			}
		}
	}

	if web.arena.CurrentMatch != nil && web.arena.CurrentMatch.Type == model.Playoff &&
		(web.arena.MatchState == field.TimeoutActive || web.arena.MatchState == field.PostTimeout) {
		before := web.arena.CurrentMatch.TypeOrder
		old, oldPresent := oldByBefore[before]
		newItem, newPresent := newByBefore[before]
		if oldPresent != newPresent ||
			(oldPresent && (old.DurationSec != newItem.DurationSec || old.Description != newItem.Description)) {
			return fmt.Errorf("cannot edit a break whose field countdown is active")
		}
	}

	// Every downstream match moves by the cumulative difference in break
	// durations at all earlier positions; existing match intervals remain intact.
	shift := time.Duration(0)
	newMatchTimes := make(map[int]time.Time, len(matches))
	for _, match := range matches {
		newDuration := 0
		if item, ok := newByBefore[match.TypeOrder]; ok {
			newDuration = item.DurationSec
		}
		oldDuration := 0
		if item, ok := oldByBefore[match.TypeOrder]; ok {
			oldDuration = item.DurationSec
		}
		shift += time.Duration(newDuration-oldDuration) * time.Second
		newTime := match.Time.Add(shift)
		if !newTime.Equal(match.Time) && (!match.StartedAt.IsZero() || match.IsComplete()) {
			return fmt.Errorf("cannot reschedule a match that has already started or finished")
		}
		newMatchTimes[match.TypeOrder] = newTime
	}

	// All validation is complete. Apply the new schedule, then recreate the
	// scheduled break rows that the match progression mechanism reads.
	for i := range matches {
		match := &matches[i]
		newTime := newMatchTimes[match.TypeOrder]
		if !match.Time.Equal(newTime) {
			match.Time = newTime
			if err := web.arena.Database.UpdateMatch(match); err != nil {
				return err
			}
			if web.arena.CurrentMatch != nil && web.arena.CurrentMatch.Id == match.Id {
				web.arena.CurrentMatch.Time = match.Time
			}
		}
	}
	if err := web.arena.Database.DeleteScheduledBreaksByMatchType(model.Playoff); err != nil {
		return err
	}
	for _, item := range desired {
		before := beforeByAfter[item.AfterMatchOrder]
		pause := model.ScheduledBreak{
			MatchType: model.Playoff, TypeOrderBefore: before,
			Time: newMatchTimes[before].Add(-time.Duration(item.DurationSec) * time.Second),
			DurationSec: item.DurationSec, Description: item.Description,
		}
		if err := web.arena.Database.CreateScheduledBreak(&pause); err != nil {
			return err
		}
	}
	web.arena.MatchListNotifier.Notify()
	web.arena.MatchLoadNotifier.Notify()
	web.arena.EventStatusNotifier.Notify()
	return nil
}
