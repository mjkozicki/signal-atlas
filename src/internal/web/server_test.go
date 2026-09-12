package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"signal-atlas/src/internal/scanner"
)

func TestAPIWorkflowAndBoundary(t *testing.T) {
	s, e := scanner.OpenStore(":memory:")
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	h := Handler(s)
	request := func(method, path string, body []byte, origin, header string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, "http://127.0.0.1:8787"+path, bytes.NewReader(body))
		if origin != "" {
			r.Header.Set("Origin", origin)
		}
		r.Header.Set("X-Wifi-Scanner", header)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	w := request("GET", "/api/scans", nil, "", "")
	if w.Code != 200 || strings.TrimSpace(w.Body.String()) != "[]" {
		t.Fatal(w.Body.String())
	}
	_, inv := scanner.Demo()
	body, _ := json.Marshal(inv)
	if w = request("PUT", "/api/inventory", body, "", ""); w.Code != 403 {
		t.Fatal("mutation accepted without custom header")
	}
	if w = request("PUT", "/api/inventory", body, "https://evil.example", "local"); w.Code != 403 {
		t.Fatal("cross-origin write accepted")
	}
	if w = request("PUT", "/api/inventory", body, "http://127.0.0.1:8787", "local"); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	pcap, e := os.ReadFile("../../../fixtures/before.pcap")
	if e != nil {
		t.Fatal(e)
	}
	w = request("POST", "/api/import", pcap, "", "local")
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	var scan scanner.Scan
	if e = json.Unmarshal(w.Body.Bytes(), &scan); e != nil {
		t.Fatal(e)
	}
	if len(scan.Findings) != 4 {
		t.Fatal("import not analyzed")
	}
	for _, format := range []string{"json", "sarif", "html"} {
		w = request("GET", "/api/export/"+scan.ID+"?format="+format, nil, "", "")
		if w.Code != http.StatusOK || w.Header().Get("Content-Disposition") == "" {
			t.Fatal("export failed")
		}
	}
	if w = request("POST", "/api/import", []byte("malformed"), "", "local"); w.Code != 400 {
		t.Fatal("invalid capture accepted")
	}
	if w = request("POST", "/api/reassess", nil, "", "local"); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	if w = request("GET", "/api/diff?from="+scan.ID+"&to=latest", nil, "", ""); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	r := httptest.NewRequest("GET", "http://evil.example/api/scans", nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal("DNS rebinding host accepted")
	}
	if e = Serve(s, "0.0.0.0:8787", false); e == nil {
		t.Fatal("non-loopback server accepted")
	}
}

func TestContainerListenBoundary(t *testing.T) {
	for _, tc := range []struct {
		addr               string
		container, allowed bool
	}{
		{"127.0.0.1:8787", false, true},
		{"[::1]:8787", false, true},
		{"0.0.0.0:8787", false, false},
		{"0.0.0.0:8787", true, true},
		{"192.168.1.2:8787", true, false},
		{"localhost:8787", true, false},
		{":8787", true, false},
		{"[::]:8787", true, false},
	} {
		if err := validateListenAddress(tc.addr, tc.container); (err == nil) != tc.allowed {
			t.Errorf("address %q, container=%v: %v", tc.addr, tc.container, err)
		}
	}
}
