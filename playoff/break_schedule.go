// Copyright 2026 Freezy Arena contributors.
// Author: Justin Fischer <justin@justfishin99.xyz>
//
// Editable playoff break placement, using the actual previous and next match
// rather than assuming consecutive match order numbers.

package playoff

import (
	"fmt"
	"github.com/Team254/cheesy-arena/model"
	"sort"
	"strings"
)

// BreakSlot describes an available position between two playoff matches.
type BreakSlot struct {
	AfterMatchOrder  int
	AfterMatchName   string
	BeforeMatchOrder int
	BeforeMatchName  string
}

// BreakSlots returns all valid positions for a playoff break, including
// brackets that skip match numbers when fewer alliances are participating.
func (tournament *PlayoffTournament) BreakSlots() []BreakSlot {
	slots := make([]BreakSlot, 0, len(tournament.matchSpecs))
	for i := 1; i < len(tournament.matchSpecs); i++ {
		previous := tournament.matchSpecs[i-1]
		next := tournament.matchSpecs[i]
		slots = append(slots, BreakSlot{
			AfterMatchOrder: previous.order,
			AfterMatchName: previous.longName,
			BeforeMatchOrder: next.order,
			BeforeMatchName: next.longName,
		})
	}
	return slots
}

// ConfiguredBreaks converts the tournament's break specifications to editable
// "after match" settings. A new tournament starts with hard-coded defaults.
func (tournament *PlayoffTournament) ConfiguredBreaks() []model.PlayoffBreakSetting {
	afterByBefore := make(map[int]int)
	for _, slot := range tournament.BreakSlots() {
		afterByBefore[slot.BeforeMatchOrder] = slot.AfterMatchOrder
	}
	settings := make([]model.PlayoffBreakSetting, 0, len(tournament.breakSpecs))
	for _, spec := range tournament.breakSpecs {
		if after, ok := afterByBefore[spec.orderBefore]; ok {
			settings = append(settings, model.PlayoffBreakSetting{
				AfterMatchOrder: after, DurationSec: spec.durationSec, Description: spec.description,
			})
		}
	}
	return settings
}

// ConfigureBreaks replaces hard-coded playoff break rules with the chosen
// positions, descriptions, and durations. Invalid configurations are rejected
// before modifying the in-memory tournament.
func (tournament *PlayoffTournament) ConfigureBreaks(settings []model.PlayoffBreakSetting) error {
	orderBeforeByAfter := make(map[int]int)
	for _, slot := range tournament.BreakSlots() {
		orderBeforeByAfter[slot.AfterMatchOrder] = slot.BeforeMatchOrder
	}

	specs := make([]breakSpec, 0, len(settings))
	seen := make(map[int]bool)
	for _, setting := range settings {
		before, ok := orderBeforeByAfter[setting.AfterMatchOrder]
		if !ok {
			return fmt.Errorf("match %d is not a valid break position", setting.AfterMatchOrder)
		}
		if seen[setting.AfterMatchOrder] {
			return fmt.Errorf("more than one break is configured after match %d", setting.AfterMatchOrder)
		}
		if setting.DurationSec < 60 || setting.DurationSec > 86400 || setting.DurationSec%60 != 0 {
			return fmt.Errorf("break duration must be between 1 and 1440 whole minutes")
		}
		description := strings.TrimSpace(setting.Description)
		if len(description) == 0 || len(description) > 128 {
			return fmt.Errorf("break name must be between 1 and 128 characters")
		}
		specs = append(specs, breakSpec{
			orderBefore: before, durationSec: setting.DurationSec, description: description,
		})
		seen[setting.AfterMatchOrder] = true
	}
	sort.Slice(specs, func(i, j int) bool { return specs[i].orderBefore < specs[j].orderBefore })
	tournament.breakSpecs = specs
	return nil
}
