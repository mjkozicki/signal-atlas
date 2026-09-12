package sensor

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type fakeProvider struct{ block bool }

func (fakeProvider) Status(context.Context) Status {
	return Status{Protocol: "bluetooth", Available: true, Provider: "test"}
}
func (p fakeProvider) Scan(ctx context.Context, _ Options) (Result, error) {
	if p.block {
		<-ctx.Done()
		return Result{}, ctx.Err()
	}
	return p.Demo(), nil
}
func (fakeProvider) Demo() Result {
	now := time.Now().UTC()
	r := -55
	return Result{Devices: []Device{{ID: "fixture:1", Name: "Test beacon", FirstSeen: now, LastSeen: now, Observations: 1, RSSI: &r, Details: map[string]any{"transport": "BLE"}}}, Warnings: []string{}}
}
func storeForTest(t *testing.T) *Store {
	t.Helper()
	s, e := OpenStore(filepath.Join(t.TempDir(), "sensor.db"), "bluetooth")
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { s.Close() })
	return s
}
func awaitScan(t *testing.T, s *Store, id string) *Scan {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		x, e := s.Get(id)
		if e != nil {
			t.Fatal(e)
		}
		if x.State != "running" {
			return x
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("scan did not finish")
	return nil
}
func TestLifecycleCancellationAndLease(t *testing.T) {
	s := storeForTest(t)
	m := NewManager(s, fakeProvider{block: true})
	defer m.Close()
	x, e := m.Start(Options{Mode: "live", Duration: 1})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = m.Start(Options{Mode: "live", Duration: 1}); !errors.Is(e, ErrBusy) {
		t.Fatalf("expected active lease: %v", e)
	}
	if e = m.Cancel(x.ID); e != nil {
		t.Fatal(e)
	}
	done := awaitScan(t, s, x.ID)
	if done.State != "canceled" || len(done.Devices) != 0 {
		t.Fatalf("%+v", done)
	}
	demo, e := m.Start(Options{Mode: "demo", Duration: 1})
	if e != nil {
		t.Fatal(e)
	}
	done = awaitScan(t, s, demo.ID)
	if done.State != "completed" || len(done.Devices) != 1 || done.Mode != "demo" {
		t.Fatalf("%+v", done)
	}
	if _, e = m.Start(Options{Mode: ""}); e == nil {
		t.Fatal("implicit live mode accepted")
	}
	if _, e = m.Start(Options{Mode: "live", Duration: 61}); e == nil {
		t.Fatal("unbounded duration accepted")
	}
}
func TestProtocolIsolationPersistenceAndRecovery(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sensor.db")
	s, e := OpenStore(path, "bluetooth")
	if e != nil {
		t.Fatal(e)
	}
	if other, e := OpenStore(path, "nfc"); e == nil {
		other.Close()
		t.Fatal("protocol DB isolation missing")
	}
	x, e := s.Begin(Options{Mode: "live", Duration: 1})
	if e != nil {
		t.Fatal(e)
	}
	s.DB.Exec("UPDATE sensor_lease SET expires=0")
	s.Close()
	s, e = OpenStore(path, "bluetooth")
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	restored, e := s.Get(x.ID)
	if e != nil {
		t.Fatal(e)
	}
	if restored.State != "failed" || !strings.Contains(restored.Error, "expired") {
		t.Fatalf("%+v", restored)
	}
	m := NewManager(s, fakeProvider{})
	defer m.Close()
	x, e = m.Start(Options{Mode: "demo", Duration: 1})
	if e != nil {
		t.Fatal(e)
	}
	awaitScan(t, s, x.ID)
}
func TestImportDiffAndCSV(t *testing.T) {
	s := storeForTest(t)
	m := NewManager(s, fakeProvider{})
	defer m.Close()
	x, e := m.Start(Options{Mode: "demo", Duration: 1})
	if e != nil {
		t.Fatal(e)
	}
	a := awaitScan(t, s, x.ID)
	var b Scan
	data, _ := json.Marshal(a)
	json.Unmarshal(data, &b)
	b.Hidden = true
	b.Devices[0].Name = "=HYPERLINK(\"evil\")"
	if e = s.Import(&b); e != nil {
		t.Fatal(e)
	}
	if b.Hidden {
		t.Fatal("import copied local visibility metadata")
	}
	if b.ID == a.ID {
		t.Fatal("import overwrote original snapshot")
	}
	diff, e := Diff(a, &b)
	if e != nil || len(diff) != 1 || diff[0].Kind != "changed" {
		t.Fatalf("%+v %v", diff, e)
	}
	var csv bytes.Buffer
	Export(&csv, &b, "csv")
	if !strings.Contains(csv.String(), "'=HYPERLINK") {
		t.Fatal("CSV formula not escaped")
	}
	b.Protocol = "nfc"
	if e = s.Import(&b); e == nil {
		t.Fatal("wrong protocol imported")
	}
	if _, e = Diff(a, &b); e == nil {
		t.Fatal("cross-protocol diff accepted")
	}
	b = *a
	b.Mode = "live"
	if _, e = Diff(a, &b); e == nil {
		t.Fatal("demo/live comparison accepted")
	}
	b = *a
	b.Devices = append(b.Devices, b.Devices[0])
	if e = s.Import(&b); e == nil {
		t.Fatal("duplicate IDs accepted")
	}
}
func TestHTTPBoundariesAndWorkflow(t *testing.T) {
	s := storeForTest(t)
	m := NewManager(s, fakeProvider{})
	defer m.Close()
	h := Handler(m)
	request := func(method, path, body, origin, header string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, "http://127.0.0.1:8788"+path, strings.NewReader(body))
		r.Header.Set("Origin", origin)
		r.Header.Set("X-Wifi-Scanner", header)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	body := `{"mode":"demo","duration_seconds":1}`
	if w := request("POST", "/api/scans", body, "", ""); w.Code != 403 {
		t.Fatal("missing mutation header accepted")
	}
	if w := request("POST", "/api/scans", body, "https://evil.test", "local"); w.Code != 403 {
		t.Fatal("cross-origin request accepted")
	}
	if w := request("POST", "/api/scans", body+body, "", "local"); w.Code != 400 {
		t.Fatal("trailing document accepted")
	}
	if w := request("POST", "/api/scans", `{"mode":"live","python":"evil"}`, "", "local"); w.Code != 400 {
		t.Fatal("browser-specified executable accepted")
	}
	w := request("POST", "/api/scans", body, "", "local")
	if w.Code != 202 {
		t.Fatal(w.Body.String())
	}
	var scan Scan
	json.Unmarshal(w.Body.Bytes(), &scan)
	completed := awaitScan(t, s, scan.ID)
	data, _ := json.Marshal(completed)
	if w = request("POST", "/api/import", string(data), "", "local"); w.Code != 201 {
		t.Fatal(w.Body.String())
	}
	for _, path := range []string{"/api/status", "/api/scans", "/api/scans/latest", "/api/export/" + scan.ID + "?format=json", "/api/export/" + scan.ID + "?format=csv", "/api/diff?from=" + scan.ID + "&to=latest"} {
		if w = request("GET", path, "", "", ""); w.Code != 200 {
			t.Fatal(path, w.Code, w.Body.String())
		}
	}
	r := httptest.NewRequest("GET", "http://evil.test/api/scans", nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal("host validation missing")
	}
}
func TestPythonRunnerCancellationAndBounds(t *testing.T) {
	runner := PythonRunner{Executable: "python3", Script: "import time; time.sleep(10)"}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var result Result
	if e := runner.Run(ctx, "scan", Options{Duration: 1}, &result); e == nil {
		t.Fatal("canceled helper succeeded")
	}
	var b limitedBuffer
	b.limit = 3
	if _, e := b.Write([]byte("four")); e == nil {
		t.Fatal("unbounded helper output")
	}
}

func TestNormalizeMissingDetailsAndBoundResult(t *testing.T) {
	r := Result{Devices: []Device{{ID: "unknown", FirstSeen: time.Now(), LastSeen: time.Now(), Observations: 1}}}
	if err := ValidateResult(&r); err != nil || r.Devices[0].Details == nil {
		t.Fatalf("normalize absent details: %v", err)
	}
	r.Warnings = []string{strings.Repeat("x", (4<<20)+1)}
	if err := ValidateResult(&r); err == nil {
		t.Fatal("oversized result accepted")
	}
}
