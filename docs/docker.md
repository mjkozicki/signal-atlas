# Run Signal Atlas in Docker

The included image runs the dashboard and three independent Go services in one
container. It supports synthetic demos, capture/snapshot imports, analysis, exports,
the About guide, and snapshot hide/restore/delete. Native radio providers are not
installed in this image.

## Start the demo

Install Docker Engine with Compose on Linux, or Docker Desktop on macOS/Windows.
Start Docker, then run from the project root:

```bash
docker compose up --build -d --wait
```

With npm installed, you can use the root shortcut instead:

```bash
npm run docker
```

It builds the image before starting the container and waits for all services to be
healthy. The build installs frontend and Go dependencies inside Docker. No host
`npm ci` or Go installation is needed. Extra Compose `up` options can follow `--`.

Open **http://127.0.0.1:8787**. Go, Node, and Python are not needed on the host.
The first build downloads base images and locked application dependencies. The image
builds for the Docker engine's architecture, including Linux amd64 and arm64.

Only port 8787 is published, bound to host loopback. Bluetooth and NFC APIs remain
inside the container and are reached through the dashboard's existing proxies.
The container runs as UID/GID 10001, with a read-only root filesystem, a temporary
`/tmp`, and a writable named volume at `/app/.data`.

If the native launcher is already using port 8787, stop it or choose another port:

```bash
SIGNAL_ATLAS_PORT=8797 docker compose up -d --wait
```

Then open http://127.0.0.1:8797. Keep the same variable value for subsequent commands.
On PowerShell set `$env:SIGNAL_ATLAS_PORT = '8797'` before running Compose.

## Stop, restart, and preserve snapshots

```bash
docker compose logs --tail=100 atlas
docker compose stop
docker compose up -d --wait
```

The `atlas-data` named volume holds all three SQLite databases. Compose prefixes
the volume name with its project name. Restarting, rebuilding, and `docker compose
down` preserve it. Demo seeding happens only when a demo database does not exist;
hidden/deleted snapshots stay that way after restart. Use the dashboard to add
another synthetic BLE/NFC scan, or explicitly seed Wi-Fi with:

```bash
docker compose exec atlas wifi-scan demo --db .data/demo.db
```

For a backup, stop the services and copy the entire data directory (including any
SQLite WAL files) before starting again:

```bash
docker compose stop
docker compose cp atlas:/app/.data ./signal-atlas-backup
docker compose up -d --wait
```

Store backups outside the source repository. **`docker compose down --volumes`
permanently removes this Compose project's stored observations.** Use it only for
an intentional reset. The container volume is separate from a native `.data/` folder.

## Use an empty workspace

Create an override file outside the repository, for example `/tmp/atlas-empty.yaml`:

```yaml
services:
  atlas:
    command: ["--demo=false"]
```

Then run:

```bash
docker compose -f compose.yaml -f /tmp/atlas-empty.yaml up -d --wait
```

This explicitly disables demo mode. The launcher uses `.data/wifi-scanner.db`,
`.data/bluetooth.db`, and `.data/nfc.db`; existing demo databases remain available
when you switch back. Import Wi-Fi PCAP/PCAPNG or BLE/NFC JSON through the dashboard.
Export downloads are saved by your browser on the host.

The image can also run without Compose:

```bash
docker build -t signal-atlas:local .
docker run --rm --init -p 127.0.0.1:8787:8787 \
  --mount source=signal-atlas-data,target=/app/.data \
  signal-atlas:local
```

## Live Wi-Fi, Bluetooth, and NFC

| Operation | Included container |
| --- | --- |
| Demo scans and saved-data operations | Supported; no hardware access |
| Live Wi-Fi capture | Run natively on Linux with a preconfigured monitor interface, `iw`, and `dumpcap`; import the resulting capture |
| Live Bluetooth discovery | Run the native Bleak service on the host; export/import its JSON snapshots |
| Live NFC polling | Run the native pyscard service with a PC/SC reader on the host; export/import its JSON snapshots |

On a Linux engine, a custom hardware-enabled image and deployment could use the
host's network namespace for Wi-Fi, BlueZ/D-Bus for BLE, and PC/SC socket or device
access for NFC. That requires provider packages, host permissions, and validation
with the actual adapter/reader. The supplied Compose file does not configure these
connections and does not request privileged mode or host-device access.

Docker Desktop runs Linux containers through a VM. Host networking does not by
itself make the Mac/Windows Bluetooth stack, Wi-Fi radio, or NFC reader available.
Docker documents [USB/IP](https://docs.docker.com/desktop/features/usbip/) for some
USB devices; that is a separate setup and not validated by this project. See
[Docker Desktop networking](https://docs.docker.com/desktop/features/networking/)
and the native [Bluetooth](bluetooth.md) / [NFC](nfc.md) guides.

## Network boundary and troubleshooting

`signal-atlas --container` explicitly allows the dashboard to bind to `0.0.0.0`
inside the container. Native launches still require loopback. Host/Origin checks
and mutation headers remain enforced in both modes. Keep the published port on
`127.0.0.1`; the app has no remote authentication. Do not put it behind a public
proxy. Docker explains [loopback port publishing](https://docs.docker.com/engine/network/port-publishing/).

- **Daemon unavailable:** start Docker Desktop or the Linux Docker service.
- **Port occupied:** stop the native launcher or set `SIGNAL_ATLAS_PORT` as above.
- **Bluetooth/NFC says setup needed:** expected; this image has no native providers.
- **Unhealthy container:** inspect `docker compose logs atlas`; the health check
  requires all three services to answer through the dashboard.
- **Permission denied with a custom bind mount:** make the directory writable by
  UID/GID 10001, or use the supplied named volume.

For contributors with npm installed, `python3 scripts/docker_smoke.py` invokes
`npm run docker` to build the image and exercises
all three APIs, imports/exports, request boundaries, snapshot mutations, and
persistence after container recreation. It uses an isolated Compose project, an
available localhost port, and a temporary volume removed after the test.
