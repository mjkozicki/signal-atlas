package bluetooth

import (
	"context"
	_ "embed"
	"time"
	"signal-atlas/src/internal/sensor"
)

//go:embed providers/bleak.py
var helper string

type Provider struct{ runner sensor.PythonRunner }

func New(python string) *Provider {
	return &Provider{sensor.PythonRunner{Executable: sensor.PythonPath(python), Script: helper}}
}
func (p *Provider) Status(ctx context.Context) sensor.Status {
	s := sensor.Status{Protocol: "bluetooth", Provider: "Bleak", Readers: []string{}}
	if e := p.runner.Run(ctx, "status", sensor.Options{}, &s); e != nil {
		s.Message = e.Error() + ". Run make native-setup or set --python."
	}
	return s
}
func (p *Provider) Scan(ctx context.Context, o sensor.Options) (sensor.Result, error) {
	var r sensor.Result
	e := p.runner.Run(ctx, "scan", o, &r)
	return r, e
}
func (*Provider) Demo() sensor.Result {
	now := time.Now().UTC()
	a, b := -48, -71
	return sensor.Result{Devices: []sensor.Device{
		{ID: "demo:ble:climate", Name: "Studio climate sensor", RSSI: &a, FirstSeen: now.Add(-5 * time.Second), LastSeen: now, Observations: 12, Details: map[string]any{"transport": "BLE", "service_uuids": []string{"0000181a-0000-1000-8000-00805f9b34fb"}, "manufacturer_data": map[string]string{"65535": "01020304"}, "identifier_type": "Synthetic identifier", "scan_mode": "demo"}},
		{ID: "demo:ble:beacon", Name: "Workshop beacon", RSSI: &b, FirstSeen: now.Add(-3 * time.Second), LastSeen: now, Observations: 7, Details: map[string]any{"transport": "BLE", "service_uuids": []string{}, "service_data": map[string]string{"0000feaa-0000-1000-8000-00805f9b34fb": "00"}, "identifier_type": "Synthetic identifier", "scan_mode": "demo"}},
	}, Warnings: []string{"Synthetic Bluetooth demonstration; no adapter was accessed.", "Advertisement discovery does not establish device security or ownership."}}
}
