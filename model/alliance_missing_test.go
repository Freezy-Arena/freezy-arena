// Copyright 2026 Freezy Arena contributors.
// Author: Justin Fischer <justin@justfishin99.xyz>

package model

import (
 "github.com/stretchr/testify/assert"
 "testing"
)

func TestUpdateAllianceFromMatchUnassigned(t *testing.T) {
 db := setupTestDb(t)
 defer db.Close()

 // An unresolved playoff bracket slot has alliance ID zero.
 // A scoring commit must not panic when its alliance has not been assigned.
 assert.NoError(t, db.UpdateAllianceFromMatch(0, [3]int{}))
 assert.NoError(t, db.UpdateAllianceFromMatch(0, [3]int{101, 102, 103}))

 // Corrupt / deleted nonzero alliance references must produce a useful
 // error instead of dereferencing a nil alliance pointer.
 assert.ErrorContains(t, db.UpdateAllianceFromMatch(77, [3]int{101, 102, 103}), "alliance 77")

 alliance := &Alliance{Id: 2, TeamIds: []int{101, 102, 103}, Lineup: [3]int{101, 102, 103}}
 assert.NoError(t, db.CreateAlliance(alliance))
 assert.NoError(t, db.UpdateAllianceFromMatch(2, [3]int{101, 102, 0}))
 actual, err := db.GetAllianceById(2)
 assert.NoError(t, err)
 if assert.NotNil(t, actual) {
  assert.Equal(t, [3]int{101, 102, 0}, actual.Lineup)
  assert.Equal(t, []int{101, 102, 103}, actual.TeamIds)
 }
}
