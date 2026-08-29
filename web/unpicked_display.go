// Copyright 2026 Team 254. All Rights Reserved.

// Web handlers for unpicked teams display.

package web

import (
	"github.com/Team254/cheesy-arena/websocket"
	"net/http"
)

// Renders the unpicked teams display to be chroma keyed over the video feed.
func (web *Web) unpickedDisplayHandler(w http.ResponseWriter, r *http.Request) {
	if !web.enforceDisplayConfiguration(
		w,
		r,
		map[string]string{
			"inverted": "false",
		},
	) {
		return
	}

	template, err := web.parseFiles("templates/unpicked_display.html")
	if err != nil {
		handleWebErr(w, err)
		return
	}

	err = template.ExecuteTemplate(w, "unpicked_display.html", web.arena.EventSettings)
	if err != nil {
		handleWebErr(w, err)
		return
	}
}

// The websocket endpoint for the unpicked teams display client to receive status updates.
func (web *Web) unpickedDisplayWebsocketHandler(w http.ResponseWriter, r *http.Request) {
	display, err := web.registerDisplay(r)
	if err != nil {
		handleWebErr(w, err)
		return
	}
	defer web.arena.MarkDisplayDisconnected(display.DisplayConfiguration.Id)

	ws, err := websocket.NewWebsocket(w, r)
	if err != nil {
		handleWebErr(w, err)
		return
	}
	defer closeWebsocket(ws)

	// Subscribe the websocket to the notifiers whose messages will be passed on to the client.
	ws.HandleNotifiers(
		display.Notifier,
		web.arena.AllianceSelectionNotifier,
		web.arena.AudienceDisplayModeNotifier,
		web.arena.ReloadDisplaysNotifier,
	)
}
