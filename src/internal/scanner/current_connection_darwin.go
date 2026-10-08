//go:build darwin

package scanner

import (
	"context"
	"fmt"
)

const currentConnectionScript = `
import CoreWLAN
import Darwin
import Foundation

guard let wifi = CWWiFiClient.shared().interface(),
      let ssid = wifi.ssid(),
      let bssid = wifi.bssid() else {
    fputs("No associated Wi-Fi network or permission to read its SSID. Check macOS Location Services permissions.\n", stderr)
    exit(1)
}

let connection: [String: String] = [
    "ssid": ssid,
    "bssid": bssid,
    "interface": wifi.interfaceName ?? ""
]
let data = try JSONSerialization.data(withJSONObject: connection, options: [.sortedKeys])
print(String(data: data, encoding: .utf8)!)
`

func CurrentConnection(ctx context.Context) (Connection, error) {
	out, err := command(ctx, "swift", "-e", currentConnectionScript)
	if err != nil {
		return Connection{}, fmt.Errorf("read current Wi-Fi connection with CoreWLAN: %w", err)
	}
	connection, err := ParseCurrentConnection(out)
	if err != nil {
		return Connection{}, err
	}
	return connection, nil
}
