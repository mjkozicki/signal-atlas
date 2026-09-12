# Validation

Validated on macOS arm64 with Go 1.26.4 and Node 26.

- Svelte/TypeScript check: no errors or warnings.
- Production Vite build and native Go binary build pass.
- Go unit/integration suite passes with the race detector; `go vet ./...` passes.
- Cross-compilation for Linux amd64 passes.
- Synthetic classic-PCAP, raw 802.11, and PCAPNG parsing; RSN length checks, PMF flags, hidden SSID, 6 GHz frequency/channel, and malformed/empty capture handling are covered.
- Rules tests confirm neighbor exclusion, unknown-BSSID investigation findings, expected before/after scores, and honest absent-AP comparison language.
- Persistence/reopen, inventory validation, active-confirmation rejection, connection allowlist rejection, rate limiting, HTML escaping, and export structure are covered.
- HTTP tests exercise import, inventory update, reassessment, diff, exports, invalid input, missing mutation header, cross-origin writes, DNS-rebinding Host rejection, and non-loopback bind rejection.
- CLI smoke workflow imports both fixtures, inspects APs, compares snapshots, exports all formats, and verifies that active scanning without explicit confirmation is rejected.
- Frame parser fuzzing completed about 100,000 cases after a bounds-check fix. A truncated radiotap regression seed remains in the test source.
- Browser verification confirmed dashboard rendering, AP selection, and the evidence/remediation view. Automated file-picker verification was interrupted by the browser tool; capture upload is covered at the HTTP layer, but the full native picker flow was not verified.

No live wireless capture or active network probes were executed. Linux hardware, adapter permissions, and interface binding need validation on the target sensor. Controller integrations and broader active audits are not part of this release.

## Earlier source-layout validation

The source reorganization was verified with these commands from the
`signal-atlas/` project root; all passed:

```bash
make build
make check
make test
GOOS=linux GOARCH=amd64 go build -o /tmp/wifi-scanner-linux-check ./src/cmd/wifi-scan
```

Go tests live beside the implementation under `src/internal/` and load shared
captures from the root `fixtures/` directory. TypeScript checks include
`src/web/src/`; the production UI is generated in `src/internal/web/dist/` and
embedded in the rebuilt binary. That source move did not change CLI commands,
HTTP routes, database locations, or capture formats.

A temporary-database smoke check also imported `fixtures/before.pcap` with the
rebuilt CLI, confirmed six APs and four findings, and started the rebuilt HTTP
server. The embedded HTML, CSS, JavaScript, and latest-scan API all loaded
successfully. No live capture or active network check was needed for this move.

## Bluetooth and NFC components

Validated on macOS arm64 with Go 1.26.4, Node 26, Python 3.14.6, Bleak 3.0.2,
and pyscard 2.3.1:

- All four native executables build; all command packages cross-compile for Linux
  amd64. Svelte/TypeScript reports zero errors and warnings; `go vet ./...` passes.
- The Go race suite covers asynchronous completion/cancellation, persistent scan
  leases, busy rejection, interrupted-job recovery, protocol database isolation,
  import validation, comparison semantics, CSV formula escaping, HTTP origin/Host
  restrictions, and provider output bounds.
- Nine mocked Python tests cover BLE callbacks and Linux adapter arguments, status
  without scanner creation, Type 4 and ACR122 Type 2 APDUs, advertised read-size
  limits, malformed TLVs, and reader disconnect after failure. No real adapter or
  reader is accessed by these tests.
- NDEF tests cover text, URI, UTF-16, malformed records, and bounds. A five-second
  fuzz run executed 143,218 inputs without failures.
- Combined-service HTTP checks completed synthetic scans through the dashboard
  gateway, imported/exported JSON, downloaded CSV, compared identical snapshots,
  and confirmed cross-origin mutation rejection. Each protocol uses its own database.
- Headless Chrome verified Bluetooth/NFC navigation, demo buttons, device details,
  NDEF text/URI display, history comparison, and disabled NFC live controls without
  a reader. Desktop and 390-pixel mobile screenshots were inspected; no page errors
  were reported. The native browser-control tool was unavailable; these checks used
  a temporary Playwright installation outside the project.
- Native status checks confirm the installed Bleak provider and report no connected
  PC/SC reader. Status did not create a Bluetooth scanner or poll tags.
- The combined launcher was stopped and restarted successfully with its child
  services. Demo history persisted across restart.

No live BLE scan or physical NFC tag read was performed. OS Bluetooth permission,
reader drivers, real tag interoperability, and native discovery on Linux/Windows
remain hardware-validation tasks. Unsupported tag profiles report read notes;
this implementation does not claim universal NFC compatibility.

## About guide

The About page documents measurements, scan workflows, glossary terms, and data
limits for all three services. Its score description was checked against the
implemented rule deductions and integer mean. Svelte/TypeScript checks pass with
zero errors or warnings; the production frontend and embedded Wi-Fi executable build.

Headless Chrome checks cover direct About/section links, reload at a section,
keyboard operation of the RSSI example and its relative-power calculation,
glossary search/category filters/empty state, navigation back to Overview, and
390/320-pixel layouts. The guide remains readable with an unavailable API and
initiates no mutation or scan requests. Screenshots were inspected on desktop and
mobile, including the final definition-column layout. No hardware scans were run.

## Snapshot visibility and deletion

Builds, Svelte checks (zero warnings/errors), `go vet`, and the full Go race suite
pass. Shared management integration tests exercise all three APIs: protected
mutations, strict visibility JSON, concrete IDs, persistence across reopening,
default latest selection, hidden retrieval/restoration, deletion, missing-record
exports/comparisons, removal of the final snapshot, and rejection of running-job
mutations. Hidden snapshots are filtered before the history limit; importing a
hidden sensor snapshot creates a visible copy.

Headless Chrome verified Hide, Show hidden snapshots, Restore, history-row actions,
and snapshot-specific deletion prompts on Wi-Fi, Bluetooth, and NFC. Dismissing
confirmation retained the snapshot; accepting removed only the test-created entry.
All pre-existing snapshots were preserved. Desktop/mobile screenshots were inspected,
with no browser errors or horizontal overflow in the mobile check. These checks used
synthetic/capture fixtures and did not start hardware scanning.

## Standalone Signal Atlas preparation

The project was extracted into its own repository with only project source,
synthetic fixtures, documentation, manifests, and upstream notices. No original
workspace history, databases, virtual environment, node_modules, or executables
were included in the initial source inventory.

A fresh source export passed `make build`, `make check`, `make test`, and
`make smoke` on macOS arm64. The UI check reported zero errors or warnings.
Linux and Windows amd64 command cross-compilation passed; this does not establish
native hardware or launcher compatibility on Windows (the quickstart uses WSL).
The clean export's launcher started all three services and seeded only synthetic
observations without installing Python providers. Headless Chrome verified the
Signal Atlas branding, author attribution, and Bluetooth/NFC demo views.

Local documentation links resolve, copied license files contain text, and the source
inventory was checked for local data paths, generated dependencies, and common
credential patterns. These checks are bounded inspections, not a guarantee that
arbitrary future contributions contain no sensitive data.

The GitHub Actions workflow is configured for Linux/macOS build, type/vet checks,
race tests, and hardware-free CLI smoke tests. No remote repository or public release
was created during that initial local preparation. See the
[Actions history](https://github.com/mjkozicki/signal-atlas/actions) for published
revisions and their remote validation results.

## Docker support

Validated on Docker Desktop for macOS arm64, Docker Engine 29.7.2, and Compose 5.5.0:

- The multi-stage image builds the production frontend and all four Linux arm64
  executables. The runtime uses UID/GID 10001 and includes project/upstream notices.
- `python3 scripts/docker_smoke.py` passes with a read-only root filesystem and a
  named data volume. It verifies embedded HTML/assets, all three APIs through the
  published localhost port, exports/imports, Host/Origin/mutation-header rejection,
  hide/restore/delete, and persistence after deleting and recreating the container.
- Explicit `--demo=false` switches to empty normal databases; switching back preserves
  demo-workspace observations. An empty Compose command is not used to disable demos.
- The full Go race suite and targeted `go vet` checks pass. Listen-address tests
  verify that wildcard binding requires explicit container mode. Native loopback
  restrictions and browser request checks remain covered.
- A separate demo container serves the dashboard and three seeded histories without
  native providers. The test's temporary Compose project and volume were removed.

No containerized radio or reader access was tested. The shipped image supports demos
and stored-data workflows; use the native workflow for live hardware. Linux amd64
container validation is configured in CI. The earlier published source revision
passed [Linux/macOS CI](https://github.com/mjkozicki/signal-atlas/actions/runs/34721368356).
Check the [current workflow results](https://github.com/mjkozicki/signal-atlas/actions/workflows/ci.yml)
for the Docker job's result on each published revision.

## Root npm build and launch commands

`npm run local -- --help` passed on macOS arm64: the frontend build completed,
Go modules downloaded, all four executables were compiled, and the launcher
received the forwarded argument. Synthetic CLI import/export/comparison checks passed.
The Makefile delegates to this same full npm build without recompiling Go a second time.

The Docker smoke check now starts its isolated test project through `npm run docker`.
The image build, service health, all three APIs, snapshot operations, persistence,
and demo/normal workspace switching passed. Its temporary test volume was removed.
CI installs npm dependencies and exercises `npm run build` on Linux/macOS; the
Docker job exercises the root Docker shortcut without installing host Go dependencies.

## Public About guide and multiple pages

The public site has seven focused pages. `python3 scripts/check_site.py` validates
links across pages, assets, section anchors, old-bookmark destinations, canonical
URLs, headings, accessible label targets, and exact agreement of the 32 glossary
definitions with the app's About component. All seven pages returned HTTP 200 from
the local static server. Both JavaScript files pass syntax checks. RSSI calculations
were checked at −90, −70, −60, −50, and −30 dBm; glossary matching was checked for
case/whitespace handling, category selection, empty searches, and missing terms.

The guide uses static content with optional interactive examples. It makes no
scanner API requests. Pages publishing runs the static checks before deployment.
