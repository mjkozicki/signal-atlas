package sensor

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
	"signal-atlas/src/internal/snapshots"
)

func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
func Fail(w http.ResponseWriter, e error) {
	status := 400
	if errors.Is(e, sql.ErrNoRows) {
		status = 404
	}
	if errors.Is(e, ErrBusy) {
		status = 409
	}
	JSON(w, status, map[string]string{"error": e.Error()})
}
func Decode(r *http.Request, v any) error {
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if e := d.Decode(v); e != nil {
		return e
	}
	if e := d.Decode(new(any)); e != io.EOF {
		return fmt.Errorf("expected exactly one JSON document")
	}
	return nil
}
func LocalOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, e := net.SplitHostPort(r.Host)
		if e != nil {
			host = r.Host
		}
		if host != "127.0.0.1" && host != "localhost" && host != "::1" && host != "[::1]" {
			http.Error(w, "localhost host required", 403)
			return
		}
		if origin := r.Header.Get("Origin"); origin != "" && origin != "http://"+r.Host {
			http.Error(w, "same-origin requests only", 403)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		if r.Method != "GET" && r.Method != "HEAD" {
			if r.Header.Get("X-Wifi-Scanner") != "local" || r.Header.Get("Sec-Fetch-Site") == "cross-site" {
				http.Error(w, "local application header required", 403)
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, 4<<20)
		}
		next.ServeHTTP(w, r)
	})
}
func Handler(m *Manager) http.Handler {
	mux := http.NewServeMux()
	snapshots.Register(mux, m.Store)
	mux.HandleFunc("GET /api/status", func(w http.ResponseWriter, r *http.Request) { JSON(w, 200, m.Provider.Status(r.Context())) })
	mux.HandleFunc("GET /api/scans", func(w http.ResponseWriter, r *http.Request) {
		x, e := m.Store.List(r.URL.Query().Get("include_hidden") == "true")
		if e != nil {
			Fail(w, e)
			return
		}
		JSON(w, 200, x)
	})
	mux.HandleFunc("POST /api/scans", func(w http.ResponseWriter, r *http.Request) {
		var o Options
		if e := Decode(r, &o); e != nil {
			Fail(w, e)
			return
		}
		s, e := m.Start(o)
		if e != nil {
			Fail(w, e)
			return
		}
		JSON(w, 202, s)
	})
	mux.HandleFunc("GET /api/scans/{id}", func(w http.ResponseWriter, r *http.Request) {
		s, e := m.Store.Get(r.PathValue("id"))
		if e != nil {
			Fail(w, e)
			return
		}
		JSON(w, 200, s)
	})
	mux.HandleFunc("POST /api/scans/{id}/cancel", func(w http.ResponseWriter, r *http.Request) {
		if e := m.Cancel(r.PathValue("id")); e != nil {
			Fail(w, e)
			return
		}
		JSON(w, 202, map[string]string{"state": "cancellation requested"})
	})
	mux.HandleFunc("POST /api/import", func(w http.ResponseWriter, r *http.Request) {
		var s Scan
		if e := Decode(r, &s); e != nil {
			Fail(w, e)
			return
		}
		if e := m.Store.Import(&s); e != nil {
			Fail(w, e)
			return
		}
		JSON(w, 201, s)
	})
	mux.HandleFunc("GET /api/diff", func(w http.ResponseWriter, r *http.Request) {
		a, e := m.Store.Get(r.URL.Query().Get("from"))
		if e != nil {
			Fail(w, e)
			return
		}
		b, e := m.Store.Get(r.URL.Query().Get("to"))
		if e != nil {
			Fail(w, e)
			return
		}
		d, e := Diff(a, b)
		if e != nil {
			Fail(w, e)
			return
		}
		JSON(w, 200, d)
	})
	mux.HandleFunc("GET /api/export/{id}", func(w http.ResponseWriter, r *http.Request) {
		s, e := m.Store.Get(r.PathValue("id"))
		if e != nil {
			Fail(w, e)
			return
		}
		format := r.URL.Query().Get("format")
		types := map[string]string{"json": "application/json", "csv": "text/csv; charset=utf-8"}
		if types[format] == "" {
			Fail(w, fmt.Errorf("format must be json or csv"))
			return
		}
		w.Header().Set("Content-Type", types[format])
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", s.Protocol+"-"+s.ID+"."+format))
		Export(w, s, format)
	})
	return LocalOnly(mux)
}
func Listen(ctx context.Context, m *Manager, addr string) error {
	host, _, e := net.SplitHostPort(addr)
	if e != nil {
		return e
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("bind to a literal loopback address")
	}
	srv := &http.Server{Addr: addr, Handler: Handler(m), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 16}
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			shutdown, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			srv.Shutdown(shutdown)
		case <-done:
		}
	}()
	fmt.Printf("%s service: http://%s\n", strings.ToUpper(m.Store.Protocol), addr)
	e = srv.ListenAndServe()
	if e == http.ErrServerClosed {
		return nil
	}
	return e
}
