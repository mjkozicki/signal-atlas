package web

import (
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"strings"
	"time"

	"signal-atlas/src/internal/scanner"
	"signal-atlas/src/internal/snapshots"
)

//go:embed all:dist
var assets embed.FS

func Handler(store *scanner.Store) http.Handler {
	mux := http.NewServeMux()
	snapshots.Register(mux, store)
	mux.Handle("/api/bluetooth/", serviceProxy("bluetooth"))
	mux.Handle("/api/nfc/", serviceProxy("nfc"))
	send := func(w http.ResponseWriter, value any) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(value)
	}
	fail := func(w http.ResponseWriter, e error) {
		status := http.StatusBadRequest
		if errors.Is(e, sql.ErrNoRows) {
			status = 404
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(map[string]string{"error": e.Error()})
	}
	mux.HandleFunc("GET /api/scans", func(w http.ResponseWriter, r *http.Request) {
		s, e := store.List(r.URL.Query().Get("include_hidden") == "true")
		if e != nil {
			fail(w, e)
			return
		}
		send(w, s)
	})
	mux.HandleFunc("GET /api/scans/{id}", func(w http.ResponseWriter, r *http.Request) {
		s, e := store.Get(r.PathValue("id"))
		if e != nil {
			fail(w, e)
			return
		}
		send(w, s)
	})
	mux.HandleFunc("GET /api/inventory", func(w http.ResponseWriter, r *http.Request) {
		i, e := store.Inventory()
		if e != nil {
			fail(w, e)
			return
		}
		send(w, i)
	})
	mux.HandleFunc("PUT /api/inventory", func(w http.ResponseWriter, r *http.Request) {
		var inv scanner.Inventory
		d := json.NewDecoder(r.Body)
		d.DisallowUnknownFields()
		if e := d.Decode(&inv); e != nil {
			fail(w, e)
			return
		}
		if e := store.SetInventory(inv); e != nil {
			fail(w, e)
			return
		}
		send(w, inv)
	})
	mux.HandleFunc("POST /api/reassess", func(w http.ResponseWriter, r *http.Request) {
		s, e := store.Get("latest")
		if e != nil {
			fail(w, e)
			return
		}
		inv, e := store.Inventory()
		if e != nil {
			fail(w, e)
			return
		}
		scanner.Analyze(s, inv)
		s.CreatedAt = time.Now().UTC()
		s.Source = "reassessment of " + s.ID
		if e = store.Save(s); e != nil {
			fail(w, e)
			return
		}
		send(w, s)
	})
	mux.HandleFunc("POST /api/import", func(w http.ResponseWriter, r *http.Request) {
		s, e := scanner.ReadCapture(r.Body, "imported capture")
		if e != nil {
			fail(w, e)
			return
		}
		inv, e := store.Inventory()
		if e != nil {
			fail(w, e)
			return
		}
		scanner.Analyze(s, inv)
		if e = store.Save(s); e != nil {
			fail(w, e)
			return
		}
		send(w, s)
	})
	mux.HandleFunc("GET /api/diff", func(w http.ResponseWriter, r *http.Request) {
		a, e := store.Get(r.URL.Query().Get("from"))
		if e != nil {
			fail(w, e)
			return
		}
		b, e := store.Get(r.URL.Query().Get("to"))
		if e != nil {
			fail(w, e)
			return
		}
		send(w, scanner.Diff(a, b))
	})
	mux.HandleFunc("GET /api/export/{id}", func(w http.ResponseWriter, r *http.Request) {
		s, e := store.Get(r.PathValue("id"))
		if e != nil {
			fail(w, e)
			return
		}
		format := r.URL.Query().Get("format")
		typ := map[string]string{"json": "application/json", "sarif": "application/sarif+json", "html": "text/html; charset=utf-8"}[format]
		if typ == "" {
			fail(w, fmt.Errorf("choose json, sarif, or html"))
			return
		}
		w.Header().Set("Content-Type", typ)
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", s.ID+"."+format))
		if e = scanner.Export(w, s, format); e != nil {
			return
		}
	})
	sub, _ := fs.Sub(assets, "dist")
	mux.Handle("/", http.FileServer(http.FS(sub)))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, e := net.SplitHostPort(r.Host)
		if e != nil {
			host = r.Host
		}
		if host != "127.0.0.1" && host != "localhost" && host != "[::1]" && host != "::1" {
			http.Error(w, "localhost host required", 403)
			return
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
		origin := r.Header.Get("Origin")
		if origin != "" && origin != "http://"+r.Host {
			http.Error(w, "same-origin requests only", 403)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") && r.Method != "GET" && r.Method != "HEAD" {
			if r.Header.Get("X-Wifi-Scanner") != "local" || r.Header.Get("Sec-Fetch-Site") == "cross-site" {
				http.Error(w, "local application header required", 403)
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, 32<<20)
		}
		mux.ServeHTTP(w, r)
	})
}
func validateListenAddress(addr string, container bool) error {
	h, _, e := net.SplitHostPort(addr)
	if e != nil {
		return e
	}
	ip := net.ParseIP(h)
	if ip == nil || (!ip.IsLoopback() && !(container && h == "0.0.0.0")) {
		return fmt.Errorf("server must bind to a literal loopback address, e.g. 127.0.0.1:8787")
	}
	return nil
}

func Serve(store *scanner.Store, addr string, container bool) error {
	if e := validateListenAddress(addr, container); e != nil {
		return e
	}
	s := http.Server{Addr: addr, Handler: Handler(store), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 16}
	fmt.Printf("Wi-Fi scanner: http://%s\n", addr)
	return s.ListenAndServe()
}
