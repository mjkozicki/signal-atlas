// Package sensor provides the local scan lifecycle shared by independent radio services.
package sensor

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type Device struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	FirstSeen    time.Time      `json:"first_seen"`
	LastSeen     time.Time      `json:"last_seen"`
	Observations int            `json:"observations"`
	RSSI         *int           `json:"rssi"`
	Details      map[string]any `json:"details"`
}
type Result struct {
	Devices  []Device `json:"devices"`
	Warnings []string `json:"warnings"`
}
type Status struct {
	Protocol  string   `json:"protocol"`
	Provider  string   `json:"provider"`
	Available bool     `json:"available"`
	Message   string   `json:"message"`
	Readers   []string `json:"readers"`
}
type Options struct {
	Mode     string `json:"mode"`
	Duration int    `json:"duration_seconds"`
	Adapter  string `json:"adapter,omitempty"`
	Reader   string `json:"reader,omitempty"`
}

func (o *Options) Validate(protocol string) error {
	if o.Mode != "demo" && o.Mode != "live" {
		return fmt.Errorf("mode must be explicitly demo or live")
	}
	if o.Duration == 0 {
		o.Duration = 10
	}
	if o.Duration < 1 || o.Duration > 60 {
		return fmt.Errorf("duration must be 1–60 seconds")
	}
	if len(o.Reader) > 256 || len(o.Adapter) > 64 {
		return fmt.Errorf("reader or adapter name is too long")
	}
	if protocol == "bluetooth" && o.Reader != "" {
		return fmt.Errorf("reader is only supported by NFC")
	}
	if protocol == "nfc" && o.Adapter != "" {
		return fmt.Errorf("adapter is only supported by Bluetooth")
	}
	return nil
}

type Provider interface {
	Status(context.Context) Status
	Scan(context.Context, Options) (Result, error)
	Demo() Result
}
type Scan struct {
	Hidden    bool       `json:"hidden,omitempty"`
	ID        string     `json:"id"`
	Protocol  string     `json:"protocol"`
	Mode      string     `json:"mode"`
	Source    string     `json:"source"`
	State     string     `json:"state"`
	CreatedAt time.Time  `json:"created_at"`
	EndedAt   *time.Time `json:"ended_at"`
	Duration  int        `json:"duration_seconds"`
	Devices   []Device   `json:"devices"`
	Warnings  []string   `json:"warnings"`
	Error     string     `json:"error,omitempty"`
}

func ValidateResult(r *Result) error {
	if len(r.Devices) > 10000 || len(r.Warnings) > 1000 {
		return fmt.Errorf("observation count exceeds limits")
	}
	ids := map[string]bool{}
	for i := range r.Devices {
		d := &r.Devices[i]
		if d.Details == nil {
			d.Details = map[string]any{}
		}
		if d.ID == "" || len(d.ID) > 512 || ids[d.ID] {
			return fmt.Errorf("device IDs must be nonempty, bounded, and unique")
		}
		ids[d.ID] = true
		if len(d.Name) > 512 || d.Observations < 1 || d.FirstSeen.IsZero() || d.LastSeen.Before(d.FirstSeen) {
			return fmt.Errorf("invalid observation metadata for %q", d.ID)
		}
		if d.RSSI != nil && (*d.RSSI < -127 || *d.RSSI > 20) {
			return fmt.Errorf("invalid RSSI for %q", d.ID)
		}
		b, e := json.Marshal(d.Details)
		if e != nil || len(b) > 64<<10 {
			return fmt.Errorf("device details exceed 64 KB")
		}
	}
	if r.Devices == nil {
		r.Devices = []Device{}
	}
	if r.Warnings == nil {
		r.Warnings = []string{}
	}
	encoded, err := json.Marshal(r)
	if err != nil || len(encoded) > 4<<20 {
		return fmt.Errorf("scan result exceeds 4 MB or cannot be encoded")
	}
	return nil
}
