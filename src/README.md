# Source guide

All application implementation lives here. Run build, development, and test
commands from the parent `signal-atlas/` project root.

| Directory | Responsibility |
| --- | --- |
| `cmd/wifi-scan/` | Go CLI entrypoint and command dispatch |
| `cmd/bluetooth-scan/`, `cmd/nfc-scan/` | Independent discovery service entrypoints |
| `cmd/signal-atlas/` | Launcher supervising all three local service processes |
| `internal/snapshots/` | Shared visibility metadata, transactional deletion, and management API routes |
| `internal/sensor/` | Shared job lifecycle, SQLite leases, HTTP/CLI, import/export/diff, Python runner |
| `internal/bluetooth/` | Bleak advertisement provider, demo observations, mocked callback tests |
| `internal/nfc/` | PC/SC polling, NDEF parser, synthetic tags, mocked APDU tests |
| `internal/scanner/` | Capture parsing, security rules, inventory, SQLite, diff, reports, and bounded live checks |
| `internal/web/` | Loopback dashboard API, fixed service proxies, and embedded frontend assets |
| `web/` | Vite frontend root, HTML entrypoint, and Svelte configuration |
| `web/src/` | Svelte dashboard, TypeScript types, and CSS |
| `web/src/components/` | Bluetooth/NFC views and the About measurement/process/glossary guide |
| `site/` | Static project information website, published separately to GitHub Pages |

## Build and run

```bash
npm ci
npm run local
```

The root `go.mod` defines the `signal-atlas` module. Go imports use
`signal-atlas/src/internal/scanner` and `signal-atlas/src/internal/web`.
Executable packages are under `./src/cmd/`. The local Go module is `signal-atlas`; it builds without a hosted repository URL.

The root `vite.config.ts` selects `src/web/` and writes the compiled dashboard to
`src/internal/web/dist/`. Go embeds that output at compile time; rebuild the Go
binary after changing the frontend. Only `.gitkeep` is tracked in the generated
directory.

`npm run build` runs `build:web`, then `build:go`: it builds the UI, downloads
Go modules, and compiles all four command packages into `bin/`. `npm run local`
runs that build before launching; add `-- --demo` for synthetic observations.
`npm run docker` builds the image before starting the container. `make build`
installs npm dependencies and delegates to the same root npm build.

For frontend development, start `./bin/signal-atlas` and `npm run dev` in separate
terminals at the project root. The Vite server proxies API calls to the Go service.

The public project website is plain HTML/CSS/JavaScript in `site/`, with separate
overview, scanner, quickstart, About, measurements, process, and glossary pages.
It does not start scanner services or access observations. See [website publishing](../docs/website.md)
for local preview and GitHub Pages deployment.

## Tests and supporting files

Go tests sit beside their implementation under `internal/`. Mocked Python provider tests sit beside their helpers; `make test` runs both suites without hardware.
They load shared synthetic captures from the project-root `fixtures/` directory.

```bash
make check
make test
go test ./src/internal/scanner -run '^$' -fuzz FuzzFrame -fuzztime 5s
```

Dependency manifests, lockfiles, and build configuration remain at the project root.
Documentation is in [`../docs/`](../docs/), and the main
[README](../README.md) covers setup, CLI usage, and platform limitations.
Generated executables and local databases remain in `bin/` and `.data/`.
