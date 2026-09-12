# Contributing to Signal Atlas

Start with [quickstart.md](quickstart.md). This repository is self-contained; run
commands from its root. Go code lives under `src/cmd/` and `src/internal/`; the
Svelte application is in `src/web/src/`.

```bash
npm ci
npm run build
make check
make test
```

Tests use synthetic captures, mocked providers, and temporary databases. They do
not require wireless hardware. Document live-hardware checks separately with the OS,
adapter/reader, permissions, tag profile, and what was actually observed.

For UI work, start `./bin/signal-atlas --demo` and run `npm run dev` in a second
terminal. Vite serves at `http://127.0.0.1:5178` and proxies APIs to the local services.
Run `npm run build` again to embed UI changes into all Go executables, or use
`npm run local -- --demo` to rebuild and start the demo services together.
`npm run docker` builds and starts the Docker demo without a host Go toolchain.

For container changes, run `python3 scripts/docker_smoke.py` with Docker running.
This uses an isolated Compose project and deletes only its own test volume. See
[Docker setup](docs/docker.md) for runtime architecture and hardware limits.

For website edits, run `python3 scripts/check_site.py` and check the JavaScript
syntax as described in [website publishing](docs/website.md). Keep the public About
pages aligned with `src/web/src/components/About.svelte`; all glossary definitions
are checked against that component before deployment.

Keep changes focused, preserve explicit demo/live labeling, and keep unknown values
unknown. Include a test when changing parsers, job lifecycles, persistence, API
boundaries, or rules. Update the About guide and docs when user-visible behavior changes.
Use synthetic fixtures; do not commit private captures, identifiers, databases,
credentials, or generated dependencies.

Open an issue with reproduction steps and environment details, or a pull request
with the problem, resulting behavior, and checks run. Contributions to original
project files are submitted under the project's MIT license. Preserve third-party
notices and add attribution when introducing dependencies or copied code.

After dependency changes, refresh [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)
using `python3 scripts/collect_notices.py --python .venv/bin/python` after `npm ci`
and `make native-setup`. Review changed upstream notices before redistribution.
The npm package stays `private` to prevent accidental registry publication; that
setting does not change the source license or Git repository visibility.
