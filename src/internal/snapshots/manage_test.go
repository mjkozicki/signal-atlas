package snapshots_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"signal-atlas/src/internal/scanner"
	"signal-atlas/src/internal/sensor"
	"signal-atlas/src/internal/web"
)

func TestSnapshotManagementAcrossServices(t *testing.T) {
	for _, protocol := range []string{"wifi", "bluetooth", "nfc"} {
		t.Run(protocol, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "history.db")
			var db *sql.DB
			var handler http.Handler
			var seed func() string
			var reopen func()
			if protocol == "wifi" {
				s, err := scanner.OpenStore(path)
				if err != nil {
					t.Fatal(err)
				}
				defer func() { s.Close() }()
				db = s.DB
				handler = web.Handler(s)
				seed = func() string {
					x := &scanner.Scan{Source: "test demo", APs: []scanner.AP{}, Findings: []scanner.Finding{}}
					if err := s.Save(x); err != nil {
						t.Fatal(err)
					}
					return x.ID
				}
				reopen = func() {
					s.Close()
					var err error
					s, err = scanner.OpenStore(path)
					if err != nil {
						t.Fatal(err)
					}
					db = s.DB
					handler = web.Handler(s)
				}
			} else {
				s, err := sensor.OpenStore(path, protocol)
				if err != nil {
					t.Fatal(err)
				}
				defer func() { s.Close() }()
				db = s.DB
				handler = sensor.Handler(sensor.NewManager(s, nil))
				seed = func() string {
					x, err := s.Begin(sensor.Options{Mode: "demo", Duration: 1})
					if err != nil {
						t.Fatal(err)
					}
					x.State = "completed"
					now := time.Now()
					x.EndedAt = &now
					if err = s.Finish(x); err != nil {
						t.Fatal(err)
					}
					return x.ID
				}
				reopen = func() {
					s.Close()
					var err error
					s, err = sensor.OpenStore(path, protocol)
					if err != nil {
						t.Fatal(err)
					}
					db = s.DB
					handler = sensor.Handler(sensor.NewManager(s, nil))
				}
			}
			request := func(method, path, body string, expected int) *httptest.ResponseRecorder {
				r := httptest.NewRequest(method, "http://127.0.0.1:8787"+path, strings.NewReader(body))
				r.Header.Set("X-Wifi-Scanner", "local")
				r.Header.Set("Origin", "http://127.0.0.1:8787")
				w := httptest.NewRecorder()
				handler.ServeHTTP(w, r)
				if w.Code != expected {
					t.Fatalf("%s %s: %d %s", method, path, w.Code, w.Body.String())
				}
				return w
			}
			older, newer := seed(), seed()
			// New routes keep the owning service's write protection.
			for _, method := range []string{"PATCH", "DELETE"} {
				for _, origin := range []string{"", "https://elsewhere.test"} {
					r := httptest.NewRequest(method, "http://127.0.0.1:8787/api/scans/"+newer, strings.NewReader(`{"hidden":true}`))
					r.Header.Set("Origin", origin)
					if origin != "" {
						r.Header.Set("X-Wifi-Scanner", "local")
					}
					w := httptest.NewRecorder()
					handler.ServeHTTP(w, r)
					if w.Code != 403 {
						t.Fatal("unprotected mutation", w.Code)
					}
				}
			}
			for _, body := range []string{`{}`, `{"hidden":null}`, `{"hidden":"true"}`, `{"hidden":true,"unexpected":1}`, `{"hidden":true}{}`} {
				request("PATCH", "/api/scans/"+newer, body, 400)
			}
			request("DELETE", "/api/scans/latest", "", 400)
			request("PATCH", "/api/scans/"+newer, `{"hidden":true}`, 200)
			reopen() // Visibility must survive a service restart.
			w := request("GET", "/api/scans", "", 200)
			var visible []struct {
				ID     string `json:"id"`
				Hidden bool   `json:"hidden"`
			}
			json.Unmarshal(w.Body.Bytes(), &visible)
			if len(visible) != 1 || visible[0].ID != older {
				t.Fatalf("hidden leaked into history: %s", w.Body.String())
			}
			w = request("GET", "/api/scans/latest", "", 200)
			if !strings.Contains(w.Body.String(), older) {
				t.Fatal("latest selected hidden snapshot")
			}
			w = request("GET", "/api/scans?include_hidden=true", "", 200)
			json.Unmarshal(w.Body.Bytes(), &visible)
			if len(visible) != 2 || !visible[0].Hidden {
				t.Fatal("cannot find hidden snapshot", w.Body.String())
			}
			w = request("GET", "/api/scans/"+newer, "", 200)
			if !strings.Contains(w.Body.String(), `"hidden":true`) {
				t.Fatal("explicit retrieval lost visibility")
			}
			request("PATCH", "/api/scans/"+newer, `{"hidden":false}`, 200)
			w = request("GET", "/api/scans/latest", "", 200)
			if !strings.Contains(w.Body.String(), newer) {
				t.Fatal("restore failed")
			}
			request("PATCH", "/api/scans/"+newer, `{"hidden":true}`, 200)
			request("DELETE", "/api/scans/"+newer, "", 200)
			for _, route := range []string{"/api/scans/" + newer, "/api/export/" + newer + "?format=json", "/api/diff?from=" + older + "&to=" + newer} {
				request("GET", route, "", 404)
			}
			request("DELETE", "/api/scans/"+newer, "", 404)
			request("DELETE", "/api/scans/"+older, "", 200)
			w = request("GET", "/api/scans?include_hidden=true", "", 200)
			if strings.TrimSpace(w.Body.String()) != "[]" {
				t.Fatal("final deletion left snapshot")
			}
			request("GET", "/api/scans/latest", "", 404)
			var count int
			if err := db.QueryRow("SELECT count(*) FROM snapshot_hidden").Scan(&count); err != nil || count != 0 {
				t.Fatal("orphaned visibility", count, err)
			}
			if protocol != "wifi" {
				// Persist a running job without invoking hardware; mutation must fail closed.
				id := "running-test"
				body := fmt.Sprintf(`{"id":%q,"state":"running","protocol":%q}`, id, protocol)
				if _, err := db.Exec("INSERT INTO sensor_scans VALUES (?,?)", id, body); err != nil {
					t.Fatal(err)
				}
				request("PATCH", "/api/scans/"+id, `{"hidden":true}`, 409)
				request("DELETE", "/api/scans/"+id, "", 409)
				request("GET", "/api/scans/"+id, "", 200)
			}
		})
	}
}

func TestVisibilityFiltersBeforeHistoryLimit(t *testing.T) {
	s, err := scanner.OpenStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	oldest := &scanner.Scan{}
	if err = s.Save(oldest); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 101; i++ {
		x := &scanner.Scan{}
		if err = s.Save(x); err != nil {
			t.Fatal(err)
		}
		if err = s.SetHidden(x.ID, true); err != nil {
			t.Fatal(err)
		}
	}
	visible, err := s.List()
	if err != nil || len(visible) != 1 || visible[0].ID != oldest.ID {
		t.Fatal("hidden entries consumed visible history limit", len(visible), err)
	}
}
