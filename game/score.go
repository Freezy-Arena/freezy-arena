// Copyright 2023 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Model representing the instantaneous score of a match.

package game

type Score struct {
	RobotsBypassed  [3]bool
	LeaveStatuses   [3]bool
	Fuel            int
	EndgameStatuses [3]EndgameStatus
	Fouls           []Foul
	PlayoffDq       bool
}

// Game-specific settings that can be changed via the settings.
var BargeBonusPointThreshold = 16
var IncludeAlgaeInBargeBonus = false

// Represents the state of a robot at the end of the match.
type EndgameStatus int

const (
	EndgameNone EndgameStatus = iota
	EndgameParked
	EndgameShallowCage
	EndgameDeepCage
)

// Summarize calculates and returns the summary fields used for ranking and display.
func (score *Score) Summarize(opponentScore *Score) *ScoreSummary {
	summary := new(ScoreSummary)

	// Leave the score at zero if the alliance was disqualified.
	if score.PlayoffDq {
		return summary
	}

	// Calculate autonomous period points.
	for _, status := range score.LeaveStatuses {
		if status {
			summary.LeavePoints += 3
		}
	}
	summary.AutoPoints = summary.LeavePoints

	// Calculate fuel points (1 points each).
	summary.FuelCount = score.Fuel
	summary.FuelPoints = 1 * score.Fuel

	// Calculate endgame points.
	for _, status := range score.EndgameStatuses {
		switch status {
		case EndgameParked:
			summary.BargePoints += 2
		case EndgameShallowCage:
			summary.BargePoints += 6
		case EndgameDeepCage:
			summary.BargePoints += 12
		default:
		}
	}

	summary.MatchPoints = summary.LeavePoints + summary.FuelPoints + summary.BargePoints

	// Calculate penalty points.
	for _, foul := range opponentScore.Fouls {
		summary.FoulPoints += foul.PointValue()
		// Store the number of major fouls since it is used to break ties in playoffs.
		if foul.IsMajor {
			summary.NumOpponentMajorFouls++
		}

		rule := foul.Rule()
		if rule != nil {
			// Check for the opponent fouls that automatically trigger a ranking point.
			if rule.IsRankingPoint {
				switch rule.RuleNumber {
				case "G418":
					summary.BargeBonusRankingPoint = true
				case "G428":
					summary.BargeBonusRankingPoint = true
				}
			}
		}
	}

	summary.Score = summary.MatchPoints + summary.FoulPoints

	// Calculate bonus ranking points.
	// Autonomous bonus ranking point.
	allRobotsLeft := true
	for i, left := range score.LeaveStatuses {
		if !left && !score.RobotsBypassed[i] {
			allRobotsLeft = false
			break
		}
	}
	if allRobotsLeft {
		summary.AutoBonusRankingPoint = true
	}

	// Barge bonus ranking point.
	bargePointsForBonus := summary.BargePoints
	if IncludeAlgaeInBargeBonus {
		bargePointsForBonus += summary.FuelPoints
	}
	if bargePointsForBonus >= BargeBonusPointThreshold {
		summary.BargeBonusRankingPoint = true
	}

	// Check for G206 violation.
	for _, foul := range score.Fouls {
		if foul.Rule() != nil && foul.Rule().RuleNumber == "G206" {
			summary.BargeBonusRankingPoint = false
			break
		}
	}

	// Add up the bonus ranking points.
	if summary.AutoBonusRankingPoint {
		summary.BonusRankingPoints++
	}
	if summary.BargeBonusRankingPoint {
		summary.BonusRankingPoints++
	}

	return summary
}

// Equals returns true if and only if all fields of the two scores are equal.
func (score *Score) Equals(other *Score) bool {
	if score.RobotsBypassed != other.RobotsBypassed ||
		score.LeaveStatuses != other.LeaveStatuses ||
		score.Fuel != other.Fuel ||
		score.EndgameStatuses != other.EndgameStatuses ||
		score.PlayoffDq != other.PlayoffDq ||
		len(score.Fouls) != len(other.Fouls) {
		return false
	}

	for i, foul := range score.Fouls {
		if foul != other.Fouls[i] {
			return false
		}
	}

	return true
}
