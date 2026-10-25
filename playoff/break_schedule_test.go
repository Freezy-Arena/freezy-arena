// Copyright 2026 Freezy Arena contributors.
// Author: Justin Fischer <justin@justfishin99.xyz>

package playoff

import (
	"github.com/Team254/cheesy-arena/model"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBreakScheduleDefaultsMatchHardCodedPlayoffs(t *testing.T) {
	for _, tc := range []struct {
		name         string
		playoffType  model.PlayoffType
		numAlliances int
		afterOrders  []int
	}{
		{"8 alliance double elimination", model.DoubleEliminationPlayoff, 8, []int{8, 10, 12, 13, 14, 15}},
		{"4 alliance double elimination", model.DoubleEliminationPlayoff, 4, []int{2, 4, 5, 6, 7}},
		{"3 alliance single elimination", model.SingleEliminationPlayoff, 3, []int{42, 43, 44}},
		{"2 alliance single elimination", model.SingleEliminationPlayoff, 2, []int{43, 44}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bracket, err := NewPlayoffTournament(tc.playoffType, tc.numAlliances)
			assert.NoError(t, err)
			got := bracket.ConfiguredBreaks()
			after := make([]int, len(got))
			for i, pause := range got {
				after[i] = pause.AfterMatchOrder
			}
			assert.Equal(t, tc.afterOrders, after)
		})
	}
}

func TestBreakScheduleConfiguration(t *testing.T) {
	bracket, err := NewPlayoffTournament(model.SingleEliminationPlayoff, 3)
	assert.NoError(t, err)
	custom := []model.PlayoffBreakSetting{
		{AfterMatchOrder: 38, DurationSec: 600, Description: "Inspection"},
		{AfterMatchOrder: 43, DurationSec: 1200, Description: "Awards"},
	}
	assert.NoError(t, bracket.ConfigureBreaks(custom))
	assert.Equal(t, custom, bracket.ConfiguredBreaks())
	assert.Equal(t, 40, bracket.breakSpecs[0].orderBefore) // The bracket skips order 39.
	assert.Equal(t, 44, bracket.breakSpecs[1].orderBefore)

	assert.Error(t, bracket.ConfigureBreaks([]model.PlayoffBreakSetting{
		{AfterMatchOrder: 43, DurationSec: 600, Description: "One"},
		{AfterMatchOrder: 43, DurationSec: 600, Description: "Two"},
	}))
	assert.Equal(t, custom, bracket.ConfiguredBreaks()) // Invalid edits do not partially apply.
	assert.Error(t, bracket.ConfigureBreaks([]model.PlayoffBreakSetting{
		{AfterMatchOrder: 48, DurationSec: 600, Description: "After last match"},
	}))
	assert.Error(t, bracket.ConfigureBreaks([]model.PlayoffBreakSetting{
		{AfterMatchOrder: 43, DurationSec: 0, Description: "Too short"},
	}))
	assert.NoError(t, bracket.ConfigureBreaks([]model.PlayoffBreakSetting{}))
	assert.Empty(t, bracket.ConfiguredBreaks())
}
