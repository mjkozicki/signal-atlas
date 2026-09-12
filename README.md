# Signal Atlas

Explore the signals around you.

A local Wi-Fi, Bluetooth Low Energy, and NFC discovery suite. Go services provide capture analysis and SQLite history, with a Svelte 5 / TypeScript dashboard embedded in the Wi-Fi executable. The dashboard is a local companion to the scanner, not a cloud service.

**[Project website](https://mjkozicki.github.io/signal-atlas/)** · [Quickstart](quickstart.md) · [Attribution](ATTRIBUTION.md) · [MIT License](LICENSE) · [Contributing](CONTRIBUTING.md)

## Start

With Docker running, use `npm run docker` (or `docker compose up --build -d --wait`
without npm) and open
**http://127.0.0.1:8787**. All three demo services run in one container and preserve
snapshots in a named volume. See [Docker setup and hardware limits](docs/docker.md).

To build and run natively:

Requires **Go 1.26.4+**, **Node 22.12+** (or Node 24/26), and npm. Go dependencies use pure-Go SQLite; libpcap and a C compiler are not required for offline analysis.

```bash
git clone https://github.com/mjkozicki/signal-atlas.git
cd signal-atlas
npm ci
npm run local -- --demo
```

Open **http://127.0.0.1:8787**. The launcher starts Wi-Fi, Bluetooth, and NFC services with separate synthetic demo databases. It does not access hardware until you explicitly start a live scan. Restart with `./bin/signal-atlas --demo` after the initial build. `make demo` still starts the Wi-Fi-only demo.

For your own captures:

```bash
npm run local
```

Root commands:

| Command | What it does |
| --- | --- |
| `npm run build` | Builds the UI, downloads Go modules, and compiles all four executables into `bin/` |
| `npm run local` | Runs the full build, then starts all three native services with normal databases |
| `npm run local -- --demo` | Runs the full build, then starts all three services with synthetic demo databases |
| `npm run docker` | Builds the UI and Go executables inside Docker, starts the demo container, and waits for it to become healthy |

Run `npm ci` once after cloning, and again when frontend dependencies change.
The Docker command only needs npm and Docker on the host; it installs dependencies
inside the image. Local mode stays in the foreground; press Ctrl+C to stop it.
Docker runs in the background; stop it with `docker compose down`.
`make build` remains an alias that runs `npm ci` followed by `npm run build`.

Import a `.pcap` or `.pcapng` with the dashboard's **Import capture** button. Add your owned SSIDs and exact BSSIDs under **Authorized networks**, then **Reassess latest capture**. Each reassessment creates a snapshot; older results are preserved. Imports need raw 802.11 or radiotap link types. Ordinary Ethernet/IP captures cannot supply Wi-Fi security information.

## Hide, restore, or delete snapshots

Use **Hide** beside the selected snapshot or in a history row to remove it from
normal views, including the default latest snapshot. Enable **Show hidden snapshots**
to inspect hidden entries and **Restore** them. Visibility is saved in the local
database and survives restart. The latest 100 entries are returned for the selected
visibility filter.

**Delete** asks you to confirm the specific snapshot, then permanently removes its
saved observations and findings from that service. Original capture files, exported
reports, inventory settings, and other snapshots remain. Deletion is not secure disk
erasure and does not remove backups. Running Bluetooth/NFC jobs must finish or be
canceled before they can be hidden or deleted. Hide/delete works for demo snapshots;
restarting the demo launcher does not recreate them in an existing database. Load a
new demo explicitly if you want another sample.

## About page

Open **About** in the dashboard for an interactive RSSI/dBm explanation, frequency
and channel measurements, Wi-Fi scoring, scan workflows, and a searchable glossary
covering Wi-Fi, Bluetooth, NFC, and reports. It also explains data storage and what
a snapshot can and cannot establish. The guide is available without any saved scans
at http://127.0.0.1:8787/#about.

The same information is available on the public [About guide](https://mjkozicki.github.io/signal-atlas/about.html),
with separate [Measurements](https://mjkozicki.github.io/signal-atlas/measurements.html),
[Scan process](https://mjkozicki.github.io/signal-atlas/process.html), and
[Glossary](https://mjkozicki.github.io/signal-atlas/glossary.html) pages.

## Bluetooth and NFC

Each protocol has an independent executable, database, and local API. The dashboard
at port 8787 presents all three. Bluetooth and NFC include bounded asynchronous
scans, cancellation, provider status, observation details, history comparison,
JSON import/export, and CSV export. Synthetic mode is labeled separately from live data.

| Service | Port | Default database | Native provider |
| --- | --- | --- | --- |
| Wi-Fi | 8787 | `.data/wifi-scanner.db` | Linux monitor capture; offline captures on other OSes |
| Bluetooth | 8788 | `.data/bluetooth.db` | Bleak BLE advertisements |
| NFC | 8789 | `.data/nfc.db` | pyscard / PC/SC contactless reader |

```bash
make native-setup                  # Python virtual environment + pinned providers
./bin/bluetooth-scan status         # No RF scan
./bin/nfc-scan status               # Lists PC/SC readers
./bin/signal-atlas                     # Start all services after make build
# Explicit native operations, when ready:
./bin/bluetooth-scan discover --duration 10
./bin/nfc-scan discover --duration 10 --reader 'EXACT READER NAME'
```

BLE discovery may transmit scan requests, but never connects or pairs. NFC requires
an attached reader and uses fixed UID/NDEF read commands; tag writing and authentication
are not implemented. Native providers need Python 3.10+, OS permission/driver support,
and suitable hardware. Demo mode and all stored-data operations work without them.
Read the [Bluetooth setup and limits](docs/bluetooth.md) and
[NFC setup and supported tag profiles](docs/nfc.md) before using hardware.

## Wi-Fi CLI

Every command accepts `--db PATH`; the default is `.data/wifi-scanner.db`. Commands and exports work without running the server.

```bash
./bin/wifi-scan inventory --file fixtures/authorized.yaml
./bin/wifi-scan discover --pcap fixtures/before.pcap --json
./bin/wifi-scan discover --pcap fixtures/after.pcap --band 5
./bin/wifi-scan inspect --ssid Studio
./bin/wifi-scan inspect --bssid 02:00:00:00:01:01
./bin/wifi-scan security --network Studio
./bin/wifi-scan audit --network Studio
./bin/wifi-scan diff SCAN_ID_1 SCAN_ID_2
./bin/wifi-scan diff yesterday
./bin/wifi-scan export --format json --output report.json
./bin/wifi-scan export --format sarif --output report.sarif
./bin/wifi-scan export --format html --output report.html
```

Export files are created with mode 0600; existing files are not overwritten. HTML reports are self-contained and printable. SARIF 2.1.0 reports include rule IDs, severity, evidence, confidence, and stable AP/rule fingerprints. Historical queries show the latest 100 visible scans; hidden snapshots remain in SQLite until explicitly deleted.

## Passive live capture

```bash
./bin/wifi-scan discover --interface mon0 --duration 15s
```

Live capture requires **Linux**, `iw`, `dumpcap`, appropriate capture permissions, and a **preconfigured monitor interface**. The scanner validates that the interface exists and is in monitor mode, listens for beacon/probe-response frames, and stops after 1–60 seconds, 32 MB, or 250,000 packets. It never changes adapter mode, associates to a network, injects frames, or performs channel hopping. A scan covers the adapter's current channel. No broad sudo installation or privilege changes are performed automatically.

macOS and Windows support offline PCAP/PCAPNG analysis and the dashboard. Native live **Wi-Fi capture** on these platforms is not implemented; the scanner reports this explicitly instead of substituting demo data. Export a suitable wireless capture from a supported sensor and import it locally.

## Authorization and bounded active checks

Passive analysis scores only APs matching **both** an owned SSID and an exact BSSID. An unexpected BSSID advertising an owned SSID produces an *unrecognized AP* finding with 65% confidence, not a claim that impersonation has been proven.

Inventory example:

```yaml
authorized:
  - ssid: Home
    bssids:
      - '02:11:22:33:44:55'
    targets:
      - ip: 192.168.1.1
        ports: [443]
```

```bash
./bin/wifi-scan inventory --file authorized.yaml
./bin/wifi-scan audit --network Home --active --confirm-active --interface wlan0
```

Active checks are **TCP reachability only**, implemented separately from passive findings. They require:

1. Explicit `--active` and `--confirm-active` for the run.
2. Linux `iw` validation of the currently connected SSID and BSSID against the inventory.
3. Explicit private IPv4 targets and ports (at most eight pairs), on the connected interface's subnet.
4. A socket bound to that interface and its source address; inability to bind fails closed.
5. Connection revalidation before each target, a two-second connection timeout, 250 ms pacing, and a persistent five-minute per-network rate limit.
6. A recorded start event before network activity, per-result records, and a completion event. Interrupted runs have no completion event.

No active operation is exposed by the HTTP API. A successful TCP connection is not labeled an exposed-admin vulnerability. DHCP, DNS, segmentation, client isolation, credential validation, UPnP, IPv6 auditing, and controller integration are future work.

## Wi-Fi capabilities

- PCAP and PCAPNG import (single consistent radiotap or raw 802.11 link type), SSID/BSSID deduplication, first/last observation, counts, channel/band, signal where present, basic HT/VHT width and Wi-Fi generation hints.
- WPA/RSN decoding: WPA1, WPA2, SAE/WPA3, transition mode, OWE, enterprise AKMs, ciphers, and PMF capability/requirement.
- Versioned rules with evidence, risk, remediation, confidence, and a reproducible configuration score.
- Dashboard: AP inspection, search and band filters, finding filters, RF signal and primary-channel observations, scan history/diff, inventory editing, and three export formats.
- Loopback-only HTTP serving, same-origin/custom-header mutation protection, CSP, request limits, bounded container/frame parsing, and SQLite persistence.
- Synthetic before/after captures and repeatable parser, rule, authorization, storage, report, HTTP, and fuzz tests.

## Interpretation limits

- Score = mean of authorized AP scores, starting at 100 with deductions of 40/25/10/3 for critical/high/medium/low findings, clamped at zero per AP. It summarizes observed configuration, not overall network security. With no owned APs, or an owned AP with an undecodable security configuration, the score is unknown.
- Beacon observations cannot establish password strength, client-side certificate validation, firmware age, isolation, management exposure, or actual client security. Unknown data stays unknown.
- A privacy bit without WPA/RSN is reported as **WEP / unknown legacy**, rather than conclusively WEP. Hidden SSIDs and WPA2 alone are not automatically treated as vulnerabilities.
- WPA2/3-Enterprise is intentionally ambiguous except when Suite-B-192 is observed. Advanced RSN overrides, multiple-BSSID profiles, full 6E/7 operation/width details, MLO, vendor OUI lookup, supported rates, channel utilization, and clients are not implemented.
- AP changes within a capture are summarized by the latest observation. Hidden beacons can retain an SSID from an earlier probe response. Cross-scan first/last observations are available through snapshots; a separate long-term RSSI-series index is future work.
- RF graphs are received-signal observations and primary-channel counts, not airtime utilization or an automatic channel recommendation. Missing APs in a later capture are not assumed remediated.
- Wi-Fi database records retain SSIDs/BSSIDs, but no client identities or packet payloads. The separate Bluetooth and NFC databases retain observed identifiers and advertised/tag payloads. Live temporary capture files are removed after analysis. Imported originals remain wherever you placed them. The database is local and permission-restricted, not encrypted; use OS disk encryption where appropriate. SQLite WAL/SHM sidecars should stay with the database during backup.

## Development and validation

```bash
npm ci
npm run check
npm run build
make test
make check
# optional short parser fuzz run
go test ./src/internal/scanner -run '^$' -fuzz FuzzFrame -fuzztime 5s
# Regenerate the synthetic PCAP fixtures
make fixtures
```

The production UI is embedded at Go build time. `npm run build` always builds the
UI before the Go executables; `npm run local` rebuilds before launching. The lower-level
`npm run build:web` builds only the UI, and `npm run build:go` downloads Go modules
and builds the executables using the current UI bundle. Docker uses the UI-only
command in its Node stage and compiles Go in its Go stage. For UI development, run
the Go server and `npm run dev` in separate terminals; Vite listens on
127.0.0.1:5178 and proxies `/api` to port 8787.

```text
signal-atlas/
├── src/
│   ├── cmd/                wifi-scan, bluetooth-scan, nfc-scan, and signal-atlas launcher
│   ├── internal/
│   │   ├── scanner/        Capture, rules, inventory, SQLite, reports, live checks
│   │   ├── sensor/         Shared Bluetooth/NFC lifecycle, storage, CLI, and API
│   │   ├── bluetooth/      BLE provider and embedded Python helper
│   │   ├── nfc/            PC/SC provider, NDEF parser, and embedded Python helper
│   │   └── web/            Dashboard API, service proxies, and embedded UI
│   └── web/
│       ├── src/            Svelte dashboard, TypeScript, and styles
│       └── index.html      Vite entry page
├── fixtures/               Synthetic PCAPs, BLE/NFC snapshots, and example inventory
├── docs/                   Architecture, roadmap, and validation notes
├── bin/                    Generated executables (ignored)
├── .data/                  Local SQLite databases (ignored)
├── go.mod / go.sum         Go module and dependency lock
├── package*.json           Frontend dependencies and scripts
└── Makefile                Build, run, and validation commands
```

Run all commands from the `signal-atlas/` project root. Source code lives under
`src/`; shared fixtures, documentation, dependency manifests, and build configuration
remain at the root. Vite builds `src/web/` into `src/internal/web/dist/`, which Go
embeds in `bin/wifi-scan`. The nested `src/web/src/` directory is the frontend's
source directory, relative to its Vite root.

See the [source guide](src/README.md), [architecture and roadmap](docs/architecture.md),
and [validation notes](docs/validation.md) for details.

Technical references: [dumpcap capture controls](https://www.wireshark.org/docs/man-pages/dumpcap.html), [Linux iw interfaces](https://wireless.docs.kernel.org/en/latest/en/users/documentation/iw.html), [Wireshark WLAN fields](https://www.wireshark.org/docs/dfref/w/wlan.html), [SQLite WAL](https://www.sqlite.org/wal.html), and [Svelte documentation](https://svelte.dev/docs/svelte/overview).

## License and credits

Copyright © 2026 Michael Kozicki. Original project code, documentation, and synthetic
fixtures are available under the [MIT License](LICENSE). See [ATTRIBUTION.md](ATTRIBUTION.md)
and [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md) for contributor and dependency credits.
