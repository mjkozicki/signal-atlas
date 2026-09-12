package nfc

import (
	"context"
	_ "embed"
	"encoding/hex"
	"time"
	"signal-atlas/src/internal/sensor"
)

//go:embed providers/pcsc.py
var helper string

type Provider struct{ runner sensor.PythonRunner }

func New(python string) *Provider {
	return &Provider{sensor.PythonRunner{Executable: sensor.PythonPath(python), Script: helper}}
}
func (p *Provider) Status(ctx context.Context) sensor.Status {
	s := sensor.Status{Protocol: "nfc", Provider: "PC/SC", Readers: []string{}}
	if e := p.runner.Run(ctx, "status", sensor.Options{}, &s); e != nil {
		s.Message = e.Error() + ". Run make native-setup or set --python."
	}
	return s
}
func (p *Provider) Scan(ctx context.Context, o sensor.Options) (sensor.Result, error) {
	var r sensor.Result
	if e := p.runner.Run(ctx, "scan", o, &r); e != nil {
		return r, e
	}
	for i := range r.Devices {
		d := &r.Devices[i]
		if raw, ok := d.Details["ndef_hex"].(string); ok {
			b, e := hex.DecodeString(raw)
			if e == nil {
				var records []Record
				records, e = ParseNDEF(b)
				if e == nil {
					d.Details["ndef_records"] = records
				}
			}
			if e != nil {
				d.Details["ndef_error"] = e.Error()
				r.Warnings = append(r.Warnings, "A tag's NDEF message could not be decoded; raw bytes remain available.")
			}
		}
	}
	return r, nil
}
func (*Provider) Demo() sensor.Result {
	now := time.Now().UTC()
	return sensor.Result{Devices: []sensor.Device{
		{ID: "demo:nfc:welcome", Name: "Welcome tag", FirstSeen: now.Add(-4 * time.Second), LastSeen: now, Observations: 1, Details: map[string]any{"reader": "Synthetic USB NFC reader", "uid": "04a1b2c3d4e5f6", "technology": "NFC Type 2 NDEF", "identifier_type": "Synthetic identifier", "ndef_records": []Record{{TNF: 1, Type: "T", Text: "Welcome to the studio", Language: "en", PayloadHex: "02656e57656c636f6d6520746f207468652073747564696f"}, {TNF: 1, Type: "U", URI: "https://example.com/studio", PayloadHex: "046578616d706c652e636f6d2f73747564696f"}}}},
	}, Warnings: []string{"Synthetic NFC demonstration; no reader was accessed.", "NDEF content is displayed as data. Links and tag commands are never opened or executed automatically."}}
}
