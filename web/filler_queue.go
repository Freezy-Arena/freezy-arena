package web

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"html"
	htmltemplate "html/template"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Team254/cheesy-arena/game"
	"github.com/Team254/cheesy-arena/model"
)

type fillerQueuePageData struct {
	EventSettings    *model.EventSettings
	Matches          []model.Match
	Teams            []model.Team
	Entries          []model.FillerQueueEntry
	RecentAssignments []model.FillerQueueEntry
	MatchNames       map[int]string
	Revision         string
	Error            string
	Message          string
	KioskURL         string
}

// The revision covers both match lineups and queue entries so the operator page
// also refreshes when a kiosk signup was assigned instantly.
func fillerQueueRevision(matches []model.Match, entries []model.FillerQueueEntry) string {
	content, _ := json.Marshal(struct {
		Matches []model.Match
		Entries []model.FillerQueueEntry
	}{matches, entries})
	return fmt.Sprintf("%x", sha256.Sum256(content))
}

func (web *Web) fillerQueueRevisionGetHandler(w http.ResponseWriter, r *http.Request) {
	if !web.userIsAdmin(w, r) {
		return
	}
	matches, err := web.arena.Database.GetMatchesByType(model.Practice, true)
	if err != nil {
		handleWebErr(w, err)
		return
	}
	entries, err := web.arena.Database.GetAllFillerQueueEntries()
	if err != nil {
		handleWebErr(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{
		"revision": fillerQueueRevision(matches, entries),
	}); err != nil {
		handleWebErr(w, err)
	}
}

// The operator's scheduling and queue-management page remains admin-only.
func (web *Web) addPracticeMatchGetHandler(w http.ResponseWriter, r *http.Request) {
	if !web.userIsAdmin(w, r) { return }
	web.renderFillerQueue(w, r, false)
}

func (web *Web) renderFillerQueue(w http.ResponseWriter, r *http.Request, kiosk bool) {
	// Parse the standalone kiosk through html/template, not the management
	// text/template helper: query strings and team nicknames must be HTML-escaped.
	var (
		render func(http.ResponseWriter, fillerQueuePageData) error
		err error
	)
	if kiosk {
		tmpl, parseErr := htmltemplate.New("").Funcs(htmltemplate.FuncMap(web.templateHelpers)).
			ParseFiles(filepath.Join(model.BaseDir, "templates/filler_kiosk.html"))
		err = parseErr
		if err == nil {
			render = func(w http.ResponseWriter, data fillerQueuePageData) error {
				return tmpl.ExecuteTemplate(w, "kiosk", data)
			}
		}
	} else {
		tmpl, parseErr := web.parseFiles("templates/filler_queue.html", "templates/base.html")
		err = parseErr
		if err == nil {
			render = func(w http.ResponseWriter, data fillerQueuePageData) error {
				return tmpl.ExecuteTemplate(w, "base", data)
			}
		}
	}
	if err != nil { handleWebErr(w, err); return }
	matches, err := web.arena.Database.GetMatchesByType(model.Practice, true)
	if err != nil { handleWebErr(w, err); return }
	teams, err := web.arena.Database.GetAllTeams()
	if err != nil { handleWebErr(w, err); return }
	entries, err := web.arena.Database.GetAllFillerQueueEntries()
	if err != nil { handleWebErr(w, err); return }
	names := make(map[int]string, len(matches))
	for _, match := range matches {
		names[match.Id] = match.ShortName
	}
	recent := make([]model.FillerQueueEntry, 0, 15)
	for i := len(entries) - 1; i >= 0 && len(recent) < 15; i-- {
		if entries[i].Status == model.FillerQueueAssigned {
			recent = append(recent, entries[i])
		}
	}
	data := fillerQueuePageData{
		EventSettings: web.arena.EventSettings,
		Matches: matches,
		Teams: teams,
		Entries: entries,
		RecentAssignments: recent,
		MatchNames: names,
		Revision: fillerQueueRevision(matches, entries),
		Error: r.URL.Query().Get("error"),
		Message: r.URL.Query().Get("message"),
		KioskURL: html.EscapeString(web.kioskURL(r)),
	}
	w.Header().Set("Cache-Control", "no-store")
	if err := render(w, data); err != nil { handleWebErr(w, err) }
}

func parsePracticeLineup(r *http.Request, allowIncomplete bool) ([6]int, error) {
	var lineup [6]int
	seen := make(map[int]bool)
	for i, name := range []string{"red1", "red2", "red3", "blue1", "blue2", "blue3"} {
		raw := strings.TrimSpace(r.FormValue(name))
		if raw == "" { continue }
		team, err := strconv.Atoi(raw)
		if err != nil || team < 0 { return lineup, fmt.Errorf("%s must be a positive team number", name) }
		if team != 0 {
			if seen[team] { return lineup, fmt.Errorf("team %d cannot occupy multiple stations", team) }
			seen[team] = true
		}
		lineup[i] = team
	}
	if !allowIncomplete && (lineup[0] == 0 && lineup[1] == 0 && lineup[2] == 0) { return lineup, fmt.Errorf("assign at least one red team") }
	if !allowIncomplete && (lineup[3] == 0 && lineup[4] == 0 && lineup[5] == 0) { return lineup, fmt.Errorf("assign at least one blue team") }
	return lineup, nil
}

func (web *Web) validatePracticeLineup(lineup [6]int) error {
	for _, id := range lineup {
		if id == 0 { continue }
		team, err := web.arena.Database.GetTeamById(id)
		if err != nil { return err }
		if team == nil { return fmt.Errorf("team %d is not in the event team list", id) }
	}
	return nil
}

func (web *Web) addPracticeMatchPostHandler(w http.ResponseWriter, r *http.Request) {
	if !web.userIsAdmin(w, r) { return }
	lineup, err := parsePracticeLineup(r, false)
	if err == nil { err = web.validatePracticeLineup(lineup) }
	if err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
	matches, err := web.arena.Database.GetMatchesByType(model.Practice, true)
	if err != nil { handleWebErr(w, err); return }
	nextOrder := 1
	nextTime := time.Now()
	for _, existing := range matches {
		if existing.TypeOrder >= nextOrder { nextOrder = existing.TypeOrder + 1 }
		if !existing.Time.IsZero() && !existing.Time.Before(nextTime) { nextTime = existing.Time.Add(15*time.Minute) }
	}
	if dateTime := strings.TrimSpace(r.FormValue("matchTime")); dateTime != "" {
		nextTime, err = time.ParseInLocation("2006-01-02T15:04", dateTime, time.Local)
		if err != nil { http.Error(w, "invalid match date/time", http.StatusBadRequest); return }
	}
	match := &model.Match{
		Type: model.Practice, TypeOrder: nextOrder, Time: nextTime,
		Red1: lineup[0], Red2: lineup[1], Red3: lineup[2],
		Blue1: lineup[3], Blue2: lineup[4], Blue3: lineup[5],
		Status: game.MatchScheduled, LongName: fmt.Sprintf("Practice %d", nextOrder),
		ShortName: fmt.Sprintf("P%d", nextOrder),
		TbaMatchKey: model.TbaMatchKey{CompLevel: "p", MatchNumber: nextOrder},
	}
	if err := web.arena.Database.CreateMatch(match); err != nil { handleWebErr(w, err); return }
	web.arena.MatchListNotifier.Notify()
	http.Redirect(w, r, "/panel/freezy/add_practice_match", http.StatusSeeOther)
}

func (web *Web) editPracticeMatchHandler(w http.ResponseWriter, r *http.Request) {
	if !web.userIsAdmin(w, r) { return }
	id, err := strconv.Atoi(r.FormValue("matchId"))
	if err != nil { http.Error(w, "invalid match ID", 400); return }
	match, err := web.arena.Database.GetMatchById(id)
	if err != nil { handleWebErr(w, err); return }
	if match == nil || match.Type != model.Practice { http.Error(w, "match not found", 404); return }
	if (match.Status != game.MatchScheduled && !match.IsFillerQueueDraft()) || (web.arena.CurrentMatch != nil && web.arena.CurrentMatch.Id == id) {
		http.Error(w, "cannot edit a loaded or completed match", http.StatusConflict); return
	}
	lineup, err := parsePracticeLineup(r, match.IsFillerQueueGenerated())
	if err == nil { err = web.validatePracticeLineup(lineup) }
	if err != nil { http.Error(w, err.Error(), 400); return }
	match.Red1, match.Red2, match.Red3 = lineup[0], lineup[1], lineup[2]
	match.Blue1, match.Blue2, match.Blue3 = lineup[3], lineup[4], lineup[5]
	if match.IsFillerQueueGenerated() {
		if match.HasBothAlliances() {
			match.Status = game.MatchScheduled
		} else {
			match.Status = game.MatchHidden
		}
	}
	if dateTime := strings.TrimSpace(r.FormValue("matchTime")); dateTime != "" {
		match.Time, err = time.ParseInLocation("2006-01-02T15:04", dateTime, time.Local)
		if err != nil { http.Error(w, "invalid match date/time", 400); return }
	}
	if err := web.arena.Database.UpdatePracticeMatchIfUnchanged(match, r.FormValue("matchRevision")); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	web.arena.MatchListNotifier.Notify()
	http.Redirect(w, r, "/panel/freezy/add_practice_match", 303)
}

func (web *Web) fillerKioskGetHandler(w http.ResponseWriter, r *http.Request) {
	web.renderFillerQueue(w, r, true)
}

func (web *Web) fillerKioskPostHandler(w http.ResponseWriter, r *http.Request) {
	teamId, err := strconv.Atoi(r.FormValue("teamId"))
	if err != nil || teamId <= 0 { http.Error(w, "select a team", 400); return }
	matchId := 0
	if val := r.FormValue("matchId"); val != "" {
		matchId, err = strconv.Atoi(val)
		if err != nil || matchId < 0 { http.Error(w, "invalid match", 400); return }
	}
	station := r.FormValue("station")
	entry := &model.FillerQueueEntry{TeamId: teamId, MatchId: matchId, Station: station}
	loadedMatchId := 0
	if web.arena.CurrentMatch != nil { loadedMatchId = web.arena.CurrentMatch.Id }
	if err := web.arena.Database.CreateFillerQueueEntryWithExclusion(entry, loadedMatchId); err != nil {
		http.Error(w, err.Error(), 409); return
	}
	match, err := web.arena.Database.GetMatchById(entry.MatchId)
	if err != nil { handleWebErr(w, err); return }
	web.arena.MatchListNotifier.Notify()
	message := "Your request was received."
	if match != nil {
		message = fmt.Sprintf("Team %d assigned to %s in %s.", entry.TeamId, entry.AssignedStation, match.ShortName)
		if entry.CreatedNewMatch {
			message += " A new practice match was created automatically."
		}
		if match.IsFillerQueueDraft() {
			message += " The match is forming and will open when both alliances have teams."
		}
	}
	http.Redirect(w, r, "/filler/kiosk?message="+url.QueryEscape(message), 303)
}

func (web *Web) fillerAssignPostHandler(w http.ResponseWriter, r *http.Request) {
	if !web.userIsAdmin(w, r) { return }
	entryId, err := strconv.Atoi(r.FormValue("entryId"))
	if err != nil { http.Error(w, "invalid request", 400); return }
	matchId, err := strconv.Atoi(r.FormValue("matchId"))
	if err != nil { http.Error(w, "invalid match", 400); return }
	if web.arena.CurrentMatch != nil && web.arena.CurrentMatch.Id == matchId {
		http.Error(w, "cannot assign a loaded match", 409); return
	}
	if err := web.arena.Database.AssignFillerQueueEntry(entryId, matchId, r.FormValue("station")); err != nil {
		http.Error(w, err.Error(), 409); return
	}
	web.arena.MatchListNotifier.Notify()
	http.Redirect(w, r, "/panel/freezy/add_practice_match", 303)
}

func (web *Web) fillerCancelPostHandler(w http.ResponseWriter, r *http.Request) {
	if !web.userIsAdmin(w, r) { return }
	id, err := strconv.Atoi(r.FormValue("entryId"))
	if err != nil { http.Error(w, "invalid request", 400); return }
	if err := web.arena.Database.CancelFillerQueueEntry(id); err != nil { http.Error(w, err.Error(), 409); return }
	http.Redirect(w, r, "/panel/freezy/add_practice_match", 303)
}
