package scanner

import (
	"fmt"
	"net"
	"slices"
	"strings"
	"time"
)

const RuleVersion = "1.0.0"

type Security struct {
	Protocol string   `json:"protocol"`
	AKM      []string `json:"akm"`
	Ciphers  []string `json:"ciphers"`
	PMF      string   `json:"pmf"`
}
type AP struct {
	SSID           string    `json:"ssid"`
	BSSID          string    `json:"bssid"`
	Vendor         string    `json:"vendor"`
	Band           string    `json:"band"`
	Channel        int       `json:"channel"`
	Frequency      int       `json:"frequency"`
	Width          int       `json:"width,omitempty"`
	RSSI           *int      `json:"rssi"`
	Generation     string    `json:"generation"`
	Hidden         bool      `json:"hidden"`
	BeaconInterval int       `json:"beacon_interval"`
	Security       Security  `json:"security"`
	FirstSeen      time.Time `json:"first_seen"`
	LastSeen       time.Time `json:"last_seen"`
	Observations   int       `json:"observations"`
	Authorized     bool      `json:"authorized"`
}
type Finding struct {
	ID          string   `json:"id"`
	RuleID      string   `json:"rule_id"`
	Severity    string   `json:"severity"`
	Title       string   `json:"title"`
	BSSID       string   `json:"bssid"`
	SSID        string   `json:"ssid"`
	Evidence    []string `json:"evidence"`
	Risk        string   `json:"why_it_matters"`
	Remediation string   `json:"remediation"`
	Confidence  int      `json:"confidence"`
}
type Scan struct {
	Hidden      bool      `json:"hidden,omitempty"`
	ID          string    `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	Source      string    `json:"source"`
	RuleVersion string    `json:"rule_version"`
	APs         []AP      `json:"access_points"`
	Findings    []Finding `json:"findings"`
	Score       *int      `json:"score"`
	Warnings    []string  `json:"warnings"`
}
type Target struct {
	IP    string `json:"ip" yaml:"ip"`
	Ports []int  `json:"ports" yaml:"ports"`
}
type AuthorizedNetwork struct {
	SSID    string   `json:"ssid" yaml:"ssid"`
	BSSIDs  []string `json:"bssids" yaml:"bssids"`
	Targets []Target `json:"targets,omitempty" yaml:"targets,omitempty"`
}
type Inventory struct {
	Authorized []AuthorizedNetwork `json:"authorized" yaml:"authorized"`
}

func (i *Inventory) Validate() error {
	if len(i.Authorized) > 100 {
		return fmt.Errorf("maximum 100 authorized networks")
	}
	seen := map[string]bool{}
	for n := range i.Authorized {
		a := &i.Authorized[n]
		if a.SSID == "" || len(a.SSID) > 32 || len(a.BSSIDs) == 0 {
			return fmt.Errorf("each network needs an SSID (1–32 bytes) and exact BSSIDs")
		}
		if seen[a.SSID] {
			return fmt.Errorf("duplicate SSID %q", a.SSID)
		}
		seen[a.SSID] = true
		for k, b := range a.BSSIDs {
			m, e := net.ParseMAC(b)
			if e != nil || len(m) != 6 || m[0]&1 != 0 {
				return fmt.Errorf("invalid unicast BSSID %q", b)
			}
			a.BSSIDs[k] = strings.ToLower(m.String())
		}
		count := 0
		for _, t := range a.Targets {
			ip := net.ParseIP(t.IP)
			if ip == nil || ip.To4() == nil || !ip.IsPrivate() {
				return fmt.Errorf("active targets must be literal private IPv4 addresses")
			}
			for _, p := range t.Ports {
				if p < 1 || p > 65535 {
					return fmt.Errorf("invalid port")
				}
				count++
			}
		}
		if len(a.Targets) > 0 && count == 0 {
			return fmt.Errorf("active target entries need at least one port")
		}
		if count > 8 {
			return fmt.Errorf("maximum 8 explicitly allowed address/port pairs per network")
		}
	}
	return nil
}
func (i Inventory) Owns(a AP) bool {
	for _, n := range i.Authorized {
		if n.SSID == a.SSID && slices.Contains(n.BSSIDs, strings.ToLower(a.BSSID)) {
			return true
		}
	}
	return false
}
