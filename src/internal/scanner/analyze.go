package scanner

import (
	"fmt"
	"slices"
	"strings"
)

func Analyze(s *Scan, inventory Inventory) {
	s.RuleVersion = RuleVersion
	s.Findings = []Finding{}
	s.Score = nil
	total, owned := 0, 0
	incomplete := false
	for i := range s.APs {
		a := &s.APs[i]
		a.Authorized = inventory.Owns(*a)
		penalty := 0
		add := func(id, sev, title, risk, fix string, confidence int, evidence ...string) {
			s.Findings = append(s.Findings, Finding{ID: id + ":" + a.BSSID, RuleID: id, Severity: sev, Title: title, BSSID: a.BSSID, SSID: a.SSID, Evidence: evidence, Risk: risk, Remediation: fix, Confidence: confidence})
			penalty += map[string]int{"critical": 40, "high": 25, "medium": 10, "low": 3}[sev]
		}
		if !a.Authorized {
			for _, n := range inventory.Authorized {
				if n.SSID == a.SSID {
					add("WIFI-INVENTORY-001", "high", "Unrecognized AP advertising an owned SSID", "This could be an unrecorded access point or an impersonating AP; an SSID match alone does not prove a rogue device.", "Compare this BSSID with your controller inventory and investigate its physical location before trusting it.", 65, "SSID matches authorized inventory", "BSSID "+a.BSSID+" is absent from that network's allowlist")
				}
			}
			continue
		}
		owned++
		sec := a.Security
		if sec.Protocol == "Open" {
			add("WIFI-OPEN-001", "critical", "Network advertises no link encryption", "Nearby observers may read unencrypted wireless traffic. A captive portal does not provide link encryption.", "Use WPA3-Personal/Enterprise, or Enhanced Open (OWE) for a public guest network.", 99, "Privacy bit = 0; no WPA or RSN element")
		}
		if sec.Protocol == "WEP / unknown legacy" {
			add("WIFI-LEGACY-001", "high", "Legacy or unrecognized privacy configuration", "The privacy bit is set without a recognized WPA/RSN element. This may indicate WEP, but the capture cannot confirm it.", "Verify the AP configuration and replace WEP with WPA3 or WPA2-AES where required.", 75, "Privacy bit = 1; no recognized WPA/RSN element")
		}
		if strings.Contains(sec.Protocol, "WPA1") || slices.Contains(sec.Ciphers, "TKIP") {
			add("WIFI-TKIP-001", "high", "Legacy WPA or TKIP advertised", "Legacy protocols weaken the network's available connection security.", "Disable WPA1 and TKIP; use AES-CCMP/GCMP and migrate compatible clients to WPA3.", 99, "Protocol: "+sec.Protocol, "Ciphers: "+strings.Join(sec.Ciphers, ", "))
		}
		if sec.Protocol == "WPA2/WPA3 transition" {
			add("WIFI-TRANSITION-001", "medium", "WPA2 fallback remains available", "Clients may connect with WPA2 instead of WPA3's stronger authentication.", "Migrate compatible clients to WPA3-only; isolate devices that still require WPA2.", 99, "AKM: "+strings.Join(sec.AKM, ", "))
		}
		if sec.Protocol != "Open" && sec.PMF != "unknown" && sec.PMF != "required" && sec.Protocol != "WEP / unknown legacy" {
			add("WIFI-PMF-001", "high", "Protected Management Frames are not required", "Clients can connect without protection for certain management frames.", "Require PMF after confirming client compatibility; isolate legacy devices if necessary.", 99, "PMF: "+sec.PMF)
		}
		if a.Band == "6" && (sec.PMF != "required" || (sec.Protocol != "WPA3" && sec.Protocol != "OWE" && sec.Protocol != "WPA3-Enterprise")) {
			add("WIFI-6GHZ-001", "high", "Verify 6 GHz security configuration", "The advertised security does not match the expected WPA3 or Enhanced Open configuration with required PMF.", "Confirm the capture frequency and require WPA3 or OWE with PMF on the 6 GHz radio.", 90, fmt.Sprintf("Frequency: %d MHz", a.Frequency), "Protocol: "+sec.Protocol, "PMF: "+sec.PMF)
		}
		if sec.Protocol == "Unknown" {
			incomplete = true
			add("WIFI-DATA-001", "low", "Security information is incomplete", "Incomplete or unsupported information elements prevent a reliable security assessment.", "Collect a fresh capture and confirm the AP's security settings in its controller.", 100, "RSN/WPA element could not be fully decoded")
		}
		total += max(0, 100-penalty)
	}
	if owned > 0 && !incomplete {
		v := total / owned
		s.Score = &v
	}
	slices.SortStableFunc(s.Findings, func(a, b Finding) int {
		rank := map[string]int{"critical": 0, "high": 1, "medium": 2, "low": 3}
		return rank[a.Severity] - rank[b.Severity]
	})
}
