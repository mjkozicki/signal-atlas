# Bluetooth service

`bluetooth-scan` discovers **Bluetooth Low Energy advertisements** through Bleak.
It runs independently of Wi-Fi and NFC, with its own process, SQLite database, API,
and scan lifecycle. It does not discover Bluetooth Classic devices or inspect GATT,
pairing, encryption strength, or device ownership.

## Setup and use

From `signal-atlas/`:

```bash
make build
make native-setup
./bin/bluetooth-scan status
./bin/bluetooth-scan discover --duration 10
./bin/bluetooth-scan serve                 # 127.0.0.1:8788
```

`make native-setup` installs the pinned Bleak and pyscard dependencies into `.venv`.
Native providers need Python 3.10+; pyscard may also need a C compiler and PC/SC
headers. The Go executables automatically find the project virtual environment;
use `--python /absolute/path/to/python` to select another interpreter at startup.
Demo, history, imports, exports, and comparisons need no Python dependencies.

Bleak uses CoreBluetooth on macOS, BlueZ/D-Bus on Linux, and WinRT on Windows.
macOS requires Bluetooth permission for the hosting application; a rejected
permission or powered-off adapter produces a failed scan with the provider error.
On Linux, enable Bluetooth and ensure the current account can access BlueZ.
Optionally use `--adapter hci1`. Adapter names are rejected on other platforms.
Windows needs a supported BLE adapter and OS permissions. These platform backends
are provided by Bleak; physical-device discovery has not been verified in this repo.

`status` checks dependency installation without creating a Bluetooth scanner. It
cannot establish adapter power or OS permission. A live scan performs those checks.
Discovery uses OS-managed **active scanning**, which can transmit scan requests;
it makes no peripheral connections and does not pair devices.

```bash
./bin/bluetooth-scan demo --db .data/bluetooth-demo.db
./bin/bluetooth-scan history
./bin/bluetooth-scan inspect --scan latest
./bin/bluetooth-scan export --format json --output bluetooth.json
./bin/bluetooth-scan export --format csv --output bluetooth.csv
./bin/bluetooth-scan import --file bluetooth.json
./bin/bluetooth-scan diff --from SCAN_ID --to latest
```

Commands accept `--db PATH`; the default is `.data/bluetooth.db`. Output files are
created with mode 0600 and never overwrite existing files. Imports create a new
snapshot ID, preserve the declared demo/live mode, and label the source as imported.
Compare only completed snapshots with the same mode. The dashboard exposes these
operations under **Bluetooth** when this service and `wifi-scan serve` are running.
`make serve-all` launches all services; `make demo-all` seeds isolated demo databases.

## Observations and limits

Each observed identifier includes name, first/last callback times, advertisement
count, latest RSSI when valid, advertised service UUIDs, manufacturer bytes, service
data, and advertised transmit power when provided by the OS. Data bytes are hex.
Unknown metadata is not synthesized. Callback count is not an RF packet count.

macOS reports CoreBluetooth UUIDs rather than MAC addresses. Address randomization,
OS caching, scan response behavior, and visibility vary by platform. An identifier
is not a persistent physical-device identity. Missing observations in a later scan
do not prove a device has left or a security issue was remediated. There are no BLE
security scores or inferred vulnerability findings.

Live scan durations are 1–60 seconds. Results are saved on completion, with at most
10,000 identifiers and 4 MB of provider output. A database lease permits one active
scan per service database, including across CLI and service processes. Canceling
stops the provider subprocess and records a canceled scan; partial observations are
not retained. Failed scans retain their error, with no substitution of demo data.

Data is local, permission-restricted, and unencrypted. BLE identifiers and broadcast
payloads remain in snapshots until those snapshots or the database are deleted. Back up SQLite with its
WAL/SHM sidecars or while the service is stopped.

API routes and job states are documented in [architecture.md](architecture.md).
Provider reference: [Bleak scanner API](https://bleak.readthedocs.io/en/stable/api/scanner.html)
and [BlueZ scanner arguments](https://bleak.readthedocs.io/en/latest/api/args.html).

## Manage saved snapshots

Use Hide or Delete beside a snapshot or in History. Show hidden snapshots reveals
hidden entries so they can be restored. Hidden entries are excluded from normal
history and the default latest selection. Delete confirms the exact snapshot and
cannot be undone; it does not delete exported files. Running scans must finish or
be canceled first. Visibility persists across restart, including for demo data.
