package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Team254/cheesy-arena/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (web *Web) kioskGetHttpResponse(path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://kiosk.test:8081"+path, nil)
	web.newKioskHandler().ServeHTTP(rec, req)
	return rec
}

func (web *Web) kioskPostHttpResponse(path, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "http://kiosk.test:8081"+path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	web.newKioskHandler().ServeHTTP(rec, req)
	return rec
}

func TestKioskOnlyRouterCannotAccessManagement(t *testing.T) {
	web := setupTestWeb(t)
	for _, path := range []string{
		"/login", "/match_play", "/setup/settings", "/panel/freezy/add_practice_match",
		"/panel/freezy/filler/revision", "/api/arena/websocket", "/api/freezy/startMatch",
		"/reports/csv/wpa_keys", "/static/js/cheesy-common.js", "/static/css/display_mode_controls.css",
		"/filler/kiosk/other",
	} {
		t.Run(path, func(t *testing.T) {
			resp := web.kioskGetHttpResponse(path)
			assert.Equal(t, http.StatusNotFound, resp.Code, path)
		})
	}
	for _, path := range []string{"/login", "/setup/settings", "/panel/freezy/add_practice_match", "/api/freezy/startMatch"} {
		resp := web.kioskPostHttpResponse(path, "test=value")
		assert.Equal(t, http.StatusNotFound, resp.Code, path)
	}
	assert.Equal(t, http.StatusSeeOther, web.kioskGetHttpResponse("/").Code)
	assert.Equal(t, http.StatusOK, web.kioskGetHttpResponse("/filler/kiosk").Code)
	// The management port has NO kiosk route: no accidental fallback.
	assert.Equal(t, http.StatusNotFound, web.getHttpResponse("/filler/kiosk").Code)
	assert.Equal(t, http.StatusNotFound, web.postHttpResponse("/filler/kiosk", "teamId=4728").Code)
}

func TestKioskSecureResponseAndAssets(t *testing.T) {
	web := setupTestWeb(t)
	resp := web.kioskGetHttpResponse("/filler/kiosk")
	require.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "Join Filler Queue")
	assert.NotContains(t, resp.Body.String(), "/match_play")
	assert.NotContains(t, resp.Body.String(), "/login")
	assert.NotContains(t, resp.Body.String(), "session_token")
	assert.Equal(t, "DENY", resp.Header().Get("X-Frame-Options"))
	assert.Equal(t, "nosniff", resp.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "no-store", resp.Header().Get("Cache-Control"))
	assert.Contains(t, resp.Header().Get("Content-Security-Policy"), "form-action 'self'")
	assert.Contains(t, resp.Header().Get("Content-Security-Policy"), "default-src 'none'")
	for _, asset := range []string{"/static/css/lib/bootstrap.min.css", "/static/css/cheesy-arena.css"} {
		assert.Equal(t, http.StatusOK, web.kioskGetHttpResponse(asset).Code)
	}
}

func TestKioskRejectsCrossOriginSubmissionAndEscapesInput(t *testing.T) {
	web := setupTestWeb(t)
	require.NoError(t, web.arena.Database.CreateTeam(&model.Team{Id: 4728, Nickname: "<script>alert('x')</script>"}))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "http://kiosk.test:8081/filler/kiosk", strings.NewReader("teamId=4728&station=ANY"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "https://malicious.example")
	web.newKioskHandler().ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
	entries, err := web.arena.Database.GetAllFillerQueueEntries()
	require.NoError(t, err)
	assert.Empty(t, entries)

	page := web.kioskGetHttpResponse("/filler/kiosk?message=%3Cscript%3Ealert(1)%3C%2Fscript%3E")
	assert.Equal(t, http.StatusOK, page.Code)
	assert.NotContains(t, page.Body.String(), "<script>")
	assert.Contains(t, page.Body.String(), "&lt;script&gt;")

	assert.Equal(t, http.StatusSeeOther, web.kioskPostHttpResponse("/filler/kiosk", "teamId=4728&station=ANY").Code)
}

func TestAdminPageLinksToDedicatedKioskPort(t *testing.T) {
	web := setupTestWeb(t)
	resp := web.getHttpResponse("/panel/freezy/add_practice_match")
	require.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), ":8081/filler/kiosk")
	web.KioskPort = 0
	resp = web.getHttpResponse("/panel/freezy/add_practice_match")
	assert.Contains(t, resp.Body.String(), "kiosk listener is disabled")
}
