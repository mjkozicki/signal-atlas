# Architecture and roadmap

The [source design](https://chatgpt.com/share/6aa5b3ac-71b8-83e9-8a51-8e0b33e1b0fa) describes a phased product. Signal Atlas combines that Wi-Fi inspection/security workflow with independent Bluetooth and NFC services, a shared local dashboard, and snapshot management.

## Source layout and build flow

Application code is contained in [`../src/`](../src/README.md):

- `src/cmd/wifi-scan/` owns the CLI entrypoint and imports the internal Go packages.
- `src/internal/scanner/` owns parsing, analysis, inventory, persistence, reports, and live checks.
- `src/internal/web/` owns the HTTP service and embeds its adjacent `dist/` directory.
- `src/web/` is the Vite root; `src/web/src/` holds the Svelte application.

The Go module stays at the project root, so internal imports use
`signal-atlas/src/internal/...`. The `src/internal` packages can be imported only
by Go code beneath `src/`. Frontend manifests and Vite/TypeScript configuration also
remain at the project root, allowing all build commands to run there.

`npm run build` compiles the dashboard into `src/internal/web/dist/`, downloads
Go modules, then builds
all four packages in `./src/cmd/` into `bin/`. Generated bundles are ignored; the
tracked `dist/.gitkeep` lets Go package discovery and tests run before the frontend
is built. Shared capture fixtures remain in `fixtures/`, outside production code.

`make build` runs `npm ci` and delegates to this npm build. `npm run local` builds
before starting the native launcher; `npm run docker` runs Compose with `--build`
before waiting for a healthy container. The Dockerfile uses `build:web` in the Node
stage and builds Go separately in the Go stage, so it does not need Go in the Node image.

## Boundaries

`ReadCapture` normalizes PCAP/PCAPNG management frames into observations. `CaptureLive` is a Linux adapter that feeds the same parser. The parser never calls the OS network stack. Radiotap metadata and RSN fields are checked before indexing; container lengths are checked before the PCAP reader can allocate from them. Inputs are bounded to 64 MB (32 MB HTTP), 250,000 packets, 65,535 bytes per classic-PCAP packet, and 4 MB per PCAPNG block. Mixed-link PCAPNG fails explicitly.

`Analyze` is deterministic for observations plus inventory. It never invents vendor, client, or configuration data. Every rule has a stable ID and every snapshot stores its rules version, findings, and score. Historical findings are not retroactively rewritten when the inventory changes.

SQLite preserves scan contents unless a snapshot is explicitly deleted, and uses a current inventory setting, an audit-event table, and a persistent audit-rate-limit table. Inventory changes and their audit event commit in one transaction. Snapshot JSON is stored whole for reproducibility; normalized tables can be added when cross-scan querying warrants them.

`ActiveAudit` cannot run without an explicit confirmation flag, exact associated SSID/BSSID verification, explicit private IPv4/port targets, an interface-local subnet check, and interface-bound sockets. There is no active HTTP endpoint. Authorization reflects the operator's inventory assertion plus local connection evidence; it is not cryptographic proof of physical AP ownership. Initial and per-check connection verification reduces drift, but RF identity can still be spoofed.

The HTTP service binds to literal loopback addresses by default and checks Host,
Origin, and a custom mutation header. Explicit `--container` mode permits the
dashboard to bind to `0.0.0.0` inside Docker; Compose publishes it on host loopback.
Bluetooth/NFC remain on container loopback. There is no CORS support, remote
authentication, or cloud deployment. Local processes with access to this OS account
can use the API; loopback binding is not protection against hostile local software.

The multi-stage Docker build compiles the UI with Node and embeds it into the Go
binaries. The runtime runs the existing launcher and all three services under a
non-root account, stores databases in a named volume, and includes project/dependency
notices. Native providers are omitted. See [Docker operation](docker.md).

The Svelte application is an embedded SPA. SvelteKit server rendering would add a second runtime without helping a local capture dashboard, so v0.1 uses Svelte/Vite with the Go API. Views use snapshots; Bluetooth/NFC job state is polled while an actual job is running. There are no cloud credentials or telemetry.

## Roadmap

1. Expand fixture coverage with sanitized real hardware captures: RSN overrides, multi-BSSID, EHT/HE operation, unusual radiotap namespaces, supported rates, richer width inference, and enterprise AKMs.
2. Add native Wi-Fi provider adapters for macOS and Windows with clear capability reporting, plus an external Linux sensor protocol.
3. Add a read-only controller integration (UniFi first) and a separate authoritative inventory provenance model. No credentials are collected until the integration exists.
4. Add normalized first/last-seen and RSSI histories, policy profiles, exclusions, and richer comparison context.
5. Implement each additional active check independently with bounded targets, test fixtures, authorization checks, and a distinction between direct evidence and inference.
6. Add client posture only from authoritative sources, with privacy-preserving identifiers and retention controls.

Not included: deauthentication, credential capture, password cracking, packet injection, automated network association, or unrestricted target scanning.

## Independent Bluetooth and NFC services

`src/cmd/bluetooth-scan/` and `src/cmd/nfc-scan/` use the shared `sensor` package.
Each process owns its SQLite database, provider, and loopback API. The `signal-atlas`
launcher starts the three executables and stops its children when it exits.
`make demo-all` seeds `.data/demo.db`, `.data/bluetooth-demo.db`, and
`.data/nfc-demo.db`; `make serve-all` uses the default live-workspace databases.
Neither launcher mode starts a hardware scan automatically.

The dashboard proxies `/api/bluetooth/*` and `/api/nfc/*` to fixed loopback ports
8788 and 8789. Browser requests still pass the dashboard's Host, Origin, and custom
mutation-header checks. The proxies cannot select arbitrary hosts. Each independent
API also enforces loopback and same-origin restrictions. Offline service failures
appear in the corresponding dashboard view without preventing Wi-Fi use. Custom
service `--addr` values are supported for standalone API use; the dashboard expects
the default service ports.

Providers expose `Status`, `Scan`, and `Demo` methods. Embedded Python source invokes
Bleak or pyscard through a startup-selected interpreter, without shell evaluation.
HTTP callers cannot choose an interpreter, source code, or arbitrary APDUs. Context
cancellation terminates the child; stdout is capped at 4 MB, stderr at 32 KB, and
execution at scan duration plus 10 seconds. Status checks have a 10-second bound.
Native implementations and synthetic providers return the same observation schema.

Starting a scan atomically persists a `running` snapshot and database lease before
launching the worker. The lease rejects overlapping scans across processes sharing
the same database. Completion atomically saves the snapshot and releases the lease.
The persisted states are `running`, `completed`, `failed`, and `canceled`. Expired
leases mark interrupted jobs failed on subsequent database access. A restart may
therefore require waiting until the original duration plus 30 seconds expires.
Only the process running a job can cancel it; a standalone CLI job cannot be canceled
through a different service process. Completed observation contents are immutable; visibility is separate metadata, and snapshots can be explicitly deleted. No partial
results are streamed or retained from failed/canceled provider runs.

### Bluetooth/NFC API

These paths are identical on each independent service. For the dashboard, prefix
with `/api/bluetooth` or `/api/nfc` in place of `/api`.

| Method and route | Behavior |
| --- | --- |
| `GET /api/status` | Dependency/reader status without RF discovery |
| `GET /api/scans` | Latest 100 visible snapshots |
| `POST /api/scans` | Start a job; return HTTP 202 with scan ID |
| `GET /api/scans/{id}` | Job/snapshot state; `latest` is accepted |
| `POST /api/scans/{id}/cancel` | Request cancellation; return HTTP 202 |
| `POST /api/import` | Import a completed protocol-matching JSON snapshot under a new ID |
| `GET /api/diff?from=ID&to=ID` | Compare completed snapshots of the same demo/live mode |
| `GET /api/export/{id}?format=json` | JSON snapshot download (`csv` also supported) |

A scan request must explicitly set `mode` to `live` or `demo`:

```json
{"mode":"demo","duration_seconds":10}
```

The optional `adapter` field is Bluetooth-only (named adapters require Linux); the
optional `reader` field is NFC-only and must match an enumerated reader for live
scanning. All mutations require `X-Wifi-Scanner: local`. The API accepts one strict
JSON document of at most 4 MB. Durations are 1–60 seconds, defaulting to 10. Busy
leases return 409; missing snapshots return 404; provider errors are recorded in
the asynchronous job's failed state. Frontend polling runs only while a job is active.

Imports are untrusted observations, not proof that a sensor collected the data.
They preserve the declared mode and carry an imported-source label. Comparisons
match observed identifiers, not physical devices; added/not-observed and metadata
or RSSI changes never imply ownership or remediation. CSV export protects textual
cells against spreadsheet formula interpretation. NDEF text and URIs are rendered
as escaped data, without HTML injection or automatic navigation.

See [Bluetooth](bluetooth.md) and [NFC](nfc.md) for supported native profiles and
retention details. Real hardware qualification remains necessary for each target
OS, adapter, reader, and tag combination.

## Snapshot visibility and deletion

The shared `src/internal/snapshots/` package adds an idempotent `snapshot_hidden`
metadata table to each existing database. It does not rewrite old snapshot JSON or
change capture/inventory formats. Hidden entries are filtered in SQL before the
100-entry history limit. `latest` selects the most recent visible entry; explicit
IDs can still retrieve/export hidden snapshots. Hiding is organization, not access control.

All three APIs support these operations (the BLE/NFC dashboard proxies preserve them):

| Request | Result |
| --- | --- |
| `GET /api/scans?include_hidden=true` | Latest 100 entries including hidden snapshots, with a `hidden` flag |
| `PATCH /api/scans/ID` with `{"hidden":true}` | Hide a concrete snapshot |
| `PATCH /api/scans/ID` with `{"hidden":false}` | Restore it |
| `DELETE /api/scans/ID` | Permanently delete the snapshot and visibility metadata |

Mutations use the existing same-origin and local-header checks. IDs must be concrete;
`latest` cannot be mutated. Missing entries return 404; running jobs return 409.
The running-state check and mutation occur in one transaction. Completed, failed,
and canceled sensor snapshots can be removed. The dashboard confirms deletion with
the snapshot source, timestamp, and ID, then refreshes selection and comparison state.
A hidden marker from an imported snapshot does not hide the new imported copy.

Deletion removes the selected database row, not original captures, exported reports,
inventory settings, audit events, other snapshots, or backups. SQLite may retain
bytes in free pages or WAL files; this is not a secure-erasure facility. There is no
trash/recovery feature after deletion. Startup demo seeding checks database-file
existence, so hiding/deleting demo snapshots is not undone on an ordinary restart.
