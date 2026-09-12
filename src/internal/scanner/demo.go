package scanner

import (
	"time"
)

// Demo is explicitly synthetic and never represents nearby networks.
func Demo() (*Scan, Inventory) {
	now := time.Now().UTC()
	s := &Scan{CreatedAt: now, Source: "demo · synthetic lab", APs: []AP{}, Warnings: []string{"Synthetic demonstration data; no wireless adapter was accessed."}}
	type sample struct {
		ssid, bssid, band string
		ch, freq, rssi    int
		protocol, pmf     string
		akm               []string
	}
	for _, x := range []sample{
		{"Studio", "02:00:00:00:01:01", "5", 36, 5180, -42, "WPA3", "required", []string{"SAE"}},
		{"Studio", "02:00:00:00:01:02", "6", 37, 6135, -56, "WPA3", "required", []string{"SAE"}},
		{"Studio · IoT", "02:00:00:00:02:01", "2.4", 6, 2437, -61, "WPA2", "capable", []string{"PSK"}},
		{"Studio · Guest", "02:00:00:00:03:01", "5", 149, 5745, -51, "WPA2/WPA3 transition", "capable", []string{"PSK", "SAE"}},
		{"Studio", "02:00:00:00:09:01", "5", 44, 5220, -69, "WPA2", "disabled", []string{"PSK"}},
		{"Neighbor's Wi-Fi", "02:00:00:00:08:01", "2.4", 6, 2437, -77, "WPA2", "disabled", []string{"PSK"}},
		{"Workshop", "02:00:00:00:08:02", "2.4", 1, 2412, -82, "WPA2", "capable", []string{"PSK"}},
	} {
		r := x.rssi
		w := 80
		if x.band == "2.4" {
			w = 20
		}
		s.APs = append(s.APs, AP{SSID: x.ssid, BSSID: x.bssid, Vendor: "Unknown", Band: x.band, Channel: x.ch, Frequency: x.freq, Width: w, RSSI: &r, Generation: "6", BeaconInterval: 100, Security: Security{Protocol: x.protocol, PMF: x.pmf, AKM: x.akm, Ciphers: []string{"CCMP"}}, FirstSeen: now.Add(-time.Minute), LastSeen: now, Observations: 42})
	}
	inv := Inventory{Authorized: []AuthorizedNetwork{{SSID: "Studio", BSSIDs: []string{"02:00:00:00:01:01", "02:00:00:00:01:02"}}, {SSID: "Studio · IoT", BSSIDs: []string{"02:00:00:00:02:01"}}, {SSID: "Studio · Guest", BSSIDs: []string{"02:00:00:00:03:01"}}}}
	Analyze(s, inv)
	return s, inv
}
