// Copyright 2026 Freezy Arena contributors.
// Author: Justin Fischer <justin@justfishin99.xyz>
//
// A configurable pause after an actual playoff match. The selected match order
// is converted to a break before the next actual match at schedule generation.
package model

type PlayoffBreakSetting struct {
	AfterMatchOrder int
	DurationSec     int
	Description     string
}
