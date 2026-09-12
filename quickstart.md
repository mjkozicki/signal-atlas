# Signal Atlas quickstart

Start with synthetic Wi-Fi, Bluetooth, and NFC observations. No radio permissions,
NFC reader, Python packages, or administrator access are needed for the demo.

## Run with Docker

With Docker Engine/Desktop and Compose running, get the source and start the demo:

```bash
git clone https://github.com/mjkozicki/signal-atlas.git
cd signal-atlas
docker compose up --build -d --wait
```

If npm is installed, `npm run docker` is the equivalent root command. It builds
the image before starting the container; no host `npm ci` or Go installation is needed.

Open **http://127.0.0.1:8787**. This builds the app inside Docker; host Go/Node/Python
installations are unnecessary. Snapshots persist in a named volume. Use
`docker compose down` to stop and remove the container while keeping that data.
For imports, backups, a different port, and live-hardware limitations, see the
[Docker guide](docs/docker.md). The steps below cover running natively.

## 1. Get the source and prerequisites

If you have not already downloaded the source:

```bash
git clone https://github.com/mjkozicki/signal-atlas.git
cd signal-atlas
```

You can also download and extract a source archive from the
[upstream repository](https://github.com/mjkozicki/signal-atlas). Run commands in
the project root containing `Makefile`, `go.mod`, and `package.json`.
No parent workspace is required.

For the documented macOS/Linux workflow, install:

- **Go 1.26.4 or newer:** [Go downloads](https://go.dev/dl/).
- **Node.js 22.12+**, with npm; Node 24 or 26 is also suitable:
  [Node.js downloads](https://nodejs.org/en/download).
- **make** and Python 3 are needed for the development test suite and optional
  native-provider setup, rather than the npm build/launch commands.

Check your tools:

```bash
go version
node --version
npm --version
make --version
```

The first build downloads locked dependencies, so it requires internet access.
Go's pure-Go SQLite dependency does not require libpcap or a C compiler for the demo.
The local launch command and Makefile use a POSIX shell. On Windows, use WSL for this quickstart; native
Windows reader/adapter setup is a separate hardware-validation task.

## 2. Build and launch the demo

From the project root:

```bash
npm ci
npm run local -- --demo
```

`npm run local` builds the UI, downloads the Go modules, compiles all four
executables, then launches the services. To build without launching, use
`npm run build`. `make demo-all` remains available as a convenience wrapper.

Open **http://127.0.0.1:8787**. Keep the terminal open while using the dashboard.
The build creates four executables and starts three local services:

| Service | Address | Demo database |
| --- | --- | --- |
| Wi-Fi + dashboard | `127.0.0.1:8787` | `.data/demo.db` |
| Bluetooth | `127.0.0.1:8788` | `.data/bluetooth-demo.db` |
| NFC | `127.0.0.1:8789` | `.data/nfc-demo.db` |

Try these views:

1. **Overview / Findings:** select an AP and review its advertised configuration.
2. **Bluetooth / NFC:** inspect the synthetic device or tag; **Load demo scan** adds
   another synthetic snapshot without accessing hardware.
3. **History:** compare snapshots. **Hide**, **Show hidden snapshots**, **Restore**,
   and **Delete** let you organize the examples. Delete asks for confirmation.
4. **About:** learn RSSI/dBm, SSID/BSSID, BLE advertisements, NDEF, and scan limits.

“Setup needed” on NFC or Bluetooth is expected without native dependencies/hardware;
the demo still works. Native scans run only when you explicitly start them.

Press **Ctrl+C** in the launcher terminal to stop all three services. To restart
without rebuilding:

```bash
./bin/signal-atlas --demo
```

Snapshots survive restart. Deleted/hidden demos are not automatically recreated in an
existing database. To add a fresh synthetic Wi-Fi snapshot explicitly:

```bash
./bin/wifi-scan demo --db .data/demo.db
```

## 3. Use your own observations

Stop the demo launcher first, then start the normal workspace:

```bash
npm run local
```

The normal workspace uses separate `.data/wifi-scanner.db`, `.data/bluetooth.db`,
and `.data/nfc.db` files. The dashboard remains at port 8787.

### Wi-Fi: import a capture

Click **Import capture** and choose a PCAP/PCAPNG containing raw 802.11 or radiotap
management frames. Ethernet/IP captures cannot provide the required Wi-Fi information.
Add your managed SSIDs and exact BSSIDs under **Authorized networks**, save, and
**Reassess latest capture**. Seeing an SSID does not prove you own its AP.

CLI example with the included synthetic fixture:

```bash
./bin/wifi-scan inventory --file fixtures/authorized.yaml
./bin/wifi-scan discover --pcap fixtures/before.pcap --json
```

Native live Wi-Fi capture requires Linux, `iw`, `dumpcap`, capture permissions, and
an already configured monitor interface. See [Wi-Fi usage](README.md#passive-live-capture).

### Bluetooth / NFC: install optional native providers

Install Python 3.10+ first. Then, from the project root:

```bash
make native-setup
./bin/bluetooth-scan status
./bin/nfc-scan status
```

This creates `.venv/` and installs the pinned Bleak and pyscard packages. The Go
services find that environment automatically; `--python /path/to/python` selects
another interpreter. pyscard may require a C compiler and PC/SC development headers.
On macOS these come from the platform development tools; on Linux install your
distribution's PC/SC headers, reader driver, and `pcscd` service as needed.

- **Bluetooth:** provide a supported BLE adapter, enable Bluetooth, and allow the
  hosting application Bluetooth access when the OS requests it. `status` checks
  installation; adapter power/permission is checked when discovery starts.
- **NFC:** connect a PC/SC-compatible contactless reader and install its driver. Select
  the exact reader name if several are listed, then present a compatible tag.

Use the dashboard's live scan controls, or explicitly run:

```bash
./bin/bluetooth-scan discover --duration 10
./bin/nfc-scan discover --duration 10 --reader 'EXACT NAME FROM STATUS'
```

BLE discovery may send scan requests; it does not pair or connect to peripherals.
NFC polling energizes the reader and uses fixed select/read commands, with no tag
writing or authentication. Both stop after 1–60 seconds. Real hardware behavior still
needs validation for your OS/adapter/reader. Read the [Bluetooth](docs/bluetooth.md)
and [NFC](docs/nfc.md) guides for supported profiles and troubleshooting.

## 4. Export and protect your observations

Use **Export** in the dashboard, or create an ignored local exports folder:

```bash
mkdir -p exports
./bin/bluetooth-scan export --format json --output exports/bluetooth.json
./bin/nfc-scan export --format csv --output exports/nfc.csv
./bin/wifi-scan export --format html --output exports/wifi.html
```

Those commands use the normal databases. Add `--db` with a demo database path to
export demo data instead. Existing output files are never overwritten.

History can include network/device identifiers and tag or advertisement payloads.
Databases are local and permission-restricted, but not encrypted by the app. `.data/`,
`.venv/`, `bin/`, and `exports/` are ignored by Git. Share only observations you intend
to disclose. Deleting a snapshot does not delete originals, exports, or backups.

## Troubleshooting

| Symptom | What to do |
| --- | --- |
| `address already in use` | Stop the previous launcher/server using its terminal, then retry. Default ports are 8787–8789. |
| Bluetooth/NFC service offline | Start `./bin/signal-atlas`; starting only `wifi-scan serve` starts only the dashboard/Wi-Fi service. |
| No visible snapshots | Try Show hidden snapshots, select the intended database/workspace, or explicitly add demo data. |
| Provider missing | Run `make native-setup`, or use `--python` to select the correct environment. |
| No PC/SC reader | Check USB connection, contactless capability, driver, and PC/SC service. |
| Unsupported NFC tag | Some tags yield UID/ATR only; consult the supported profiles in the NFC guide. |
| No score / unknown field | The capture or authorized scope lacks enough information. Unknown is not zero. |
| Frontend changed but dashboard looks old | Stop the old services and run `npm run local` (add `-- --demo` for demos); it rebuilds before launch. |

For development: `make build`, `make check`, and `make test`. See
[CONTRIBUTING.md](CONTRIBUTING.md), [architecture](docs/architecture.md), and
[validation notes](docs/validation.md). Signal Atlas is [MIT licensed](LICENSE);
author and dependency credits are in [ATTRIBUTION.md](ATTRIBUTION.md).
