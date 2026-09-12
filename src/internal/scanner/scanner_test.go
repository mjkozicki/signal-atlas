package scanner

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/gopacket/gopacket/pcapgo"
)

func fixture(t *testing.T, name string) *Scan {
	t.Helper()
	f, e := os.Open("../../../fixtures/" + name)
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	s, e := ReadCapture(f, "fixture")
	if e != nil {
		t.Fatal(e)
	}
	_, inv := Demo()
	Analyze(s, inv)
	return s
}
func TestCaptureAndRules(t *testing.T) {
	s := fixture(t, "before.pcap")
	if len(s.APs) != 6 {
		t.Fatalf("AP count %d", len(s.APs))
	}
	if len(s.Findings) != 4 {
		t.Fatalf("findings %+v", s.Findings)
	}
	if s.Score == nil || *s.Score != 85 {
		t.Fatalf("score %v", s.Score)
	}
	for _, a := range s.APs {
		if a.BSSID == "02:00:00:00:01:02" && (a.Band != "6" || a.Channel != 37 || a.Security.PMF != "required" || a.RSSI == nil || *a.RSSI != -56) {
			t.Fatalf("bad 6 GHz decode: %+v", a)
		}
	}
	for _, f := range s.Findings {
		if f.SSID == "Neighbor's Wi-Fi" {
			t.Fatal("neighbor was assessed")
		}
		if len(f.Evidence) == 0 || f.Confidence == 0 || f.Remediation == "" {
			t.Fatal("finding missing evidence")
		}
	}
	after := fixture(t, "after.pcap")
	if len(after.Findings) != 0 || *after.Score != 100 {
		t.Fatalf("remediated capture: %+v", after)
	}
	d := Diff(s, after)
	if len(d.Changes) < 6 {
		t.Fatalf("missing comparison changes: %+v", d)
	}
	for _, c := range d.Changes {
		if c.Kind == "finding absent" && c.BSSID == "02:00:00:00:09:01" && !strings.Contains(c.Detail, "unverified") {
			t.Fatal("absence incorrectly treated as remediation")
		}
	}
}
func TestNoInventoryNoScore(t *testing.T) {
	s := fixture(t, "before.pcap")
	Analyze(s, Inventory{})
	if s.Score != nil || len(s.Findings) != 0 {
		t.Fatal("unscoped APs must not contribute to score")
	}
}
func TestCaptureNGAndMalformed(t *testing.T) {
	f, e := os.Open("../../../fixtures/before.pcap")
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	r, e := pcapgo.NewReader(f)
	if e != nil {
		t.Fatal(e)
	}
	raw, ci, e := r.ReadPacketData()
	if e != nil {
		t.Fatal(e)
	}
	if _, err := parseFrame(raw, layers.LinkTypeIEEE80211Radio, ci.Timestamp); err != nil {
		t.Fatalf("radiotap: %v", err)
	}
	var b bytes.Buffer
	w, e := pcapgo.NewNgWriter(&b, layers.LinkTypeIEEE80211Radio)
	if e != nil {
		t.Fatal(e)
	}
	if e = w.WritePacket(ci, raw); e != nil {
		t.Fatal(e)
	}
	if e = w.Flush(); e != nil {
		t.Fatal(e)
	}
	s, e := ReadCapture(&b, "pcapng")
	if e != nil || len(s.APs) != 1 {
		t.Fatalf("pcapng decode: %+v %v", s, e)
	}
	raw = raw[:len(raw)-1]
	if _, e = parseFrame(raw, layers.LinkTypeIEEE80211Radio, ci.Timestamp); e == nil {
		t.Fatal("truncated IE accepted")
	}
	var eth bytes.Buffer
	p := pcapgo.NewWriter(&eth)
	p.WriteFileHeader(65535, layers.LinkTypeEthernet)
	if _, e = ReadCapture(&eth, "ethernet"); e == nil {
		t.Fatal("Ethernet capture accepted")
	}
	var empty bytes.Buffer
	p = pcapgo.NewWriter(&empty)
	p.WriteFileHeader(65535, layers.LinkTypeIEEE802_11)
	s, e = ReadCapture(&empty, "empty")
	if e != nil || len(s.Warnings) == 0 {
		t.Fatal("empty capture must include warning")
	}
}
func TestRSNLengthAndPMF(t *testing.T) {
	rsn := []byte{1, 0, 0, 15, 172, 4, 1, 0, 0, 15, 172, 4, 1, 0, 0, 15, 172, 8, 192, 0}
	s, e := parseRSN(rsn, false)
	if e != nil || s.Protocol != "WPA3" || s.PMF != "required" {
		t.Fatalf("%+v %v", s, e)
	}
	for _, n := range []int{0, 1, 7, 9, 13, 17, 19} {
		if _, e = parseRSN(rsn[:n], false); e == nil {
			t.Fatalf("accepted truncated RSN length %d", n)
		}
	}
	bad := append([]byte{}, rsn...)
	bad[18] = 64
	if _, e = parseRSN(bad, false); e == nil {
		t.Fatal("MFPR without MFPC accepted")
	}
}
func TestAuthorizationFailsClosed(t *testing.T) {
	inv := Inventory{Authorized: []AuthorizedNetwork{{SSID: "Home", BSSIDs: []string{"02:11:22:33:44:55"}, Targets: []Target{{IP: "192.168.1.1", Ports: []int{443}}}}}}
	if _, e := AuthorizeActive(inv, "Home", Connection{SSID: "Home", BSSID: "02:11:22:33:44:55"}); e != nil {
		t.Fatal(e)
	}
	for _, c := range []Connection{{}, {SSID: "Home", BSSID: "02:11:22:33:44:56"}, {SSID: "Other", BSSID: "02:11:22:33:44:55"}} {
		if _, e := AuthorizeActive(inv, "Home", c); e == nil {
			t.Fatal("unauthorized connection accepted")
		}
	}
	for _, ip := range []string{"8.8.8.8", "localhost", "127.0.0.1", "::1"} {
		bad := Inventory{Authorized: []AuthorizedNetwork{{SSID: "Home", BSSIDs: []string{"02:11:22:33:44:55"}, Targets: []Target{{IP: ip, Ports: []int{443}}}}}}
		if e := bad.Validate(); e == nil {
			t.Fatalf("unsafe target %s accepted", ip)
		}
	}
	store, e := OpenStore(":memory:")
	if e != nil {
		t.Fatal(e)
	}
	defer store.Close()
	if _, e = ActiveAudit(context.Background(), store, "Home", "unused", false); e == nil {
		t.Fatal("active operation without confirmation")
	}
	if e = store.AcquireAudit("Home"); e != nil {
		t.Fatal(e)
	}
	if e = store.AcquireAudit("Home"); e == nil {
		t.Fatal("rate limit bypass")
	}
}
func TestPersistenceAndExports(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.db")
	store, e := OpenStore(path)
	if e != nil {
		t.Fatal(e)
	}
	s, inv := Demo()
	if e = store.SetInventory(inv); e != nil {
		t.Fatal(e)
	}
	if e = store.Save(s); e != nil {
		t.Fatal(e)
	}
	id := s.ID
	store.Close()
	store, e = OpenStore(path)
	if e != nil {
		t.Fatal(e)
	}
	defer store.Close()
	restored, e := store.Get(id)
	if e != nil || restored.ID != id {
		t.Fatal("scan did not persist")
	}
	all, e := store.List()
	if e != nil || len(all) != 1 {
		t.Fatal("history missing")
	}
	actual, e := store.Inventory()
	if e != nil || len(actual.Authorized) != 3 {
		t.Fatal("inventory missing")
	}
	restored.APs[0].SSID = "<script>alert(1)</script>"
	for _, format := range []string{"json", "sarif", "html"} {
		var b bytes.Buffer
		if e = Export(&b, restored, format); e != nil {
			t.Fatal(e)
		}
		if format == "html" {
			if strings.Contains(b.String(), "<script>") {
				t.Fatal("HTML report permits injection")
			}
		} else {
			var value map[string]any
			if e = json.Unmarshal(b.Bytes(), &value); e != nil {
				t.Fatal(e)
			}
			if format == "sarif" && value["version"] != "2.1.0" {
				t.Fatal("invalid SARIF version")
			}
		}
	}
}
func FuzzFrame(f *testing.F) {
	f.Add([]byte{0x80, 0})
	f.Add([]byte{0, 0, 8, 0, 16, 0, 0, 0}) // Truncated radiotap FHSS regression.
	f.Add(make([]byte, 36))
	f.Add([]byte{0, 0, 8, 0, 0, 0, 0, 0})
	f.Fuzz(func(t *testing.T, b []byte) {
		if len(b) > 65535 {
			t.Skip()
		}
		parseFrame(b, layers.LinkTypeIEEE802_11, time.Time{})
		parseFrame(b, layers.LinkTypeIEEE80211Radio, time.Time{})
	})
}
func TestRawFrameOpenHidden(t *testing.T) {
	raw := make([]byte, 38)
	raw[0] = 0x80
	raw[16] = 2
	binary.LittleEndian.PutUint16(raw[32:34], 100)
	a, e := parseFrame(raw, layers.LinkTypeIEEE802_11, time.Now())
	if e != nil || !a.Hidden || a.Security.Protocol != "Open" {
		t.Fatalf("%+v %v", a, e)
	}
	var buf bytes.Buffer
	w := pcapgo.NewWriter(&buf)
	w.WriteFileHeader(65535, layers.LinkTypeIEEE802_11)
	w.WritePacket(gopacket.CaptureInfo{Timestamp: time.Now(), CaptureLength: len(raw), Length: len(raw)}, raw)
	if _, e = ReadCapture(&buf, "raw"); e != nil {
		t.Fatal(e)
	}
}
