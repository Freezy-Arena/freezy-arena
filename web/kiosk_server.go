package web

import (
	"log"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Team254/cheesy-arena/model"
)

// ServeKioskInterface deliberately uses an entirely separate HTTP router. The
// management router and its API, login, and websocket routes are NOT mounted
// on this port. Binding to a different port is not a firewall: see README.
func (web *Web) ServeKioskInterface(address string) {
	log.Printf("Serving kiosk-only HTTP requests on %s", address)
	if err := http.ListenAndServe(address, web.newKioskHandler()); err != nil {
		log.Printf("Kiosk HTTP server error: %v", err)
	}
}

// kioskURL constructs the destination from the arena web hostname. It always
// specifies plain HTTP because the built-in kiosk listener does not serve TLS.
func (web *Web) kioskURL(r *http.Request) string {
	if web.KioskPort == 0 {
		return ""
	}
	host := r.Host
	if name, _, err := net.SplitHostPort(host); err == nil {
		host = name
	} else {
		host = strings.Trim(host, "[]")
	}
	if host == "" {
		host = "localhost"
	}
	return (&url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort(host, strconv.Itoa(web.KioskPort)),
		Path:   "/filler/kiosk",
	}).String()
}

// The kiosk port only serves the form, its submit endpoint, and three
// allowlisted static resources. No management handler is registered.
func (web *Web) newKioskHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/filler/kiosk", http.StatusSeeOther)
	})
	mux.HandleFunc("GET /filler/kiosk", web.fillerKioskGetHandler)
	mux.HandleFunc("POST /filler/kiosk", web.fillerKioskPostHandler)

	staticFiles := http.StripPrefix("/static/",
		http.FileServer(http.Dir(filepath.Join(model.BaseDir, "static"))))
	for _, path := range []string{
		"/static/css/lib/bootstrap.min.css",
		"/static/css/cheesy-arena.css",
		"/static/img/favicon.ico",
	} {
		mux.Handle("GET "+path, staticFiles)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent cached forms, framing, scripts and cross-origin submits.
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Content-Security-Policy",
			"default-src 'none'; style-src 'self'; img-src 'self'; font-src 'self'; form-action 'self'; frame-ancestors 'none'; base-uri 'none'")
		if r.Method == http.MethodPost {
			// Kiosks are publicly accessible: reject browser cross-origin
			// submissions and bound request sizes before parsing a form.
			if site := r.Header.Get("Sec-Fetch-Site"); site == "cross-site" {
				http.Error(w, "Cross-origin submission denied", http.StatusForbidden)
				return
			}
			if origin := r.Header.Get("Origin"); origin != "" {
				u, err := url.Parse(origin)
				if err != nil || (u.Scheme != "http" && u.Scheme != "https") || !strings.EqualFold(u.Host, r.Host) {
					http.Error(w, "Cross-origin submission denied", http.StatusForbidden)
					return
				}
			}
			r.Body = http.MaxBytesReader(w, r.Body, 4096)
		}
		mux.ServeHTTP(w, r)
	})
}
