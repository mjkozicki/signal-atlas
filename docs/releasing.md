# Preparing a public repository or source release

The repository root is self-contained. It includes MIT licensing, attribution,
upstream notices, the quickstart, locked manifests, CI configuration, and synthetic
fixtures. The Go module is `signal-atlas`; it does not require a hosted import path
to build. Protocol executables remain `wifi-scan`, `bluetooth-scan`, and `nfc-scan`;
the suite launcher is `signal-atlas`.

## Verify the source

```bash
make build
make check
make test
make smoke
python3 scripts/docker_smoke.py
git status --short
```

Keep `src/internal/web/dist/.gitkeep` in Git so Go package discovery works before
the UI build. Generated bundles are embedded in the Go executable during `make build`,
not committed. Keep synthetic fixture captures; do not add local `.data/`, `.venv/`,
`node_modules/`, `bin/`, exports, or real wireless captures.

Dependency notices are recorded in `THIRD_PARTY_NOTICES.md` and `licenses/`. Regenerate
them after dependency updates. If distributing executables, include the project
license and upstream notices alongside them. The optional Python environment is not
part of the source release; redistributing that environment adds its own upstream
license/source obligations. Hardware support claims must match actual validation.

The Docker build includes the project and upstream notices beside the binaries.
Debian package copyright files remain under `/usr/share/doc/`. Base images and OS
packages have their own licenses. Review these when distributing a container image;
the application MIT license does not replace their terms. See [Docker setup](docker.md).

## Publish when ready

The upstream repository is [mjkozicki/signal-atlas](https://github.com/mjkozicki/signal-atlas),
with clone URL `https://github.com/mjkozicki/signal-atlas.git`.
The [project website](https://mjkozicki.github.io/signal-atlas/) is deployed from
`src/site/` by the Pages workflow; see [website publishing](website.md).

Check `git remote -v` to confirm the intended upstream before pushing the reviewed
`main` branch. For a new standalone copy, create an empty repository and add its
clone URL as `origin` first. GitHub Actions runs the Linux/macOS checks and the
Docker build/API smoke test on pushes and pull requests. Local checks cannot
establish a new revision's remote runner result in advance.

The initial standalone history contains only Signal Atlas, without unrelated parent
workspace history. Configure repository description/topics and issue settings on the
hosting service. The npm manifest is intentionally `private: true`: this application
is distributed as a source repository, not published automatically to npm.

For a source archive from a reviewed commit:

```bash
git archive --format=zip --prefix=signal-atlas/ --output=../signal-atlas-source.zip HEAD
```

This includes committed source and notices, while excluding generated dependencies
and local data. Rebuild and run the smoke checks from an extracted archive before
publishing a release artifact.
