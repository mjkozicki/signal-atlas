# Project information website

The public site is **https://mjkozicki.github.io/signal-atlas/**.
It introduces the project and includes the local app's About information as
focused reading pages, alongside scanner capabilities and setup instructions.

| Page | Content |
| --- | --- |
| `index.html` | Short project overview and routes into the site |
| `scanners.html` | Wi-Fi, Bluetooth, NFC, workspace screenshot, and capabilities |
| `quickstart.html` | Docker/native setup and root npm commands |
| `about.html` | Guide overview, local storage, data limits, credits, and references |
| `measurements.html` | RSSI/dBm illustration, bands/channels, counts, charts, and scores |
| `process.html` | Collection, decoding, snapshots, protocol workflows, and scan states |
| `glossary.html` | All 32 app definitions, with search and category filters |

Main navigation is shared across the site; About pages also have guide navigation.
Old homepage section bookmarks redirect to the corresponding new pages. Each page
has its own title, description, canonical URL, and active navigation state.

## Source and local preview

The site is authored in `src/site/` as static HTML, CSS, and JavaScript enhancements
for copying commands, exploring an illustrative RSSI value, filtering definitions,
and following old section bookmarks. It has no build dependencies, scanner API calls, account
system, cookies, or analytics scripts. The only clipboard write follows a click
on a Copy button. Commands remain selectable when JavaScript or the clipboard API
is unavailable. Definitions remain readable without JavaScript, and the RSSI panel
retains a labeled static example. No guide control starts a scan. The app screenshot is an existing image from synthetic demo
validation, with the current Signal Atlas branding.

From the repository root:

```bash
python3 -m http.server 8810 --bind 127.0.0.1 --directory src/site
```

Open http://127.0.0.1:8810. Use relative local asset links so the site works beneath
GitHub's `/signal-atlas/` project path. Keep the site content consistent with the
native application and quickstart. Never include real capture data in site assets.

Validate the site from the repository root:

```bash
python3 scripts/check_site.py
node --check src/site/site.js
node --check src/site/guide.js
```

The checker verifies links across pages, section anchors, assets, accessible label
references, main headings, canonical URLs, and glossary parity with
`src/web/src/components/About.svelte`. When changing the app guide, update its
corresponding public reading page. Measurement explanations and workflow text are
ported from that component; maintain the same interpretation limits and score rules.
The static website is excluded
from the Docker build context and is not embedded in the scanner dashboard.

## GitHub Pages publishing

The repository Pages source is **GitHub Actions**. The workflow in
`.github/workflows/pages.yml` publishes only `src/site/` when that directory or
the workflow changes on `main`. Changes to the checker or app About guide also
trigger validation and publishing. It can be started manually through Actions.
The workflow uses pinned official configure/upload/deploy Pages actions and the
`github-pages` environment, with read-only source access and scoped deployment
permissions. Static validation runs before the upload/deployment. No separately
maintained deployment branch or custom domain is needed.

The public website is informational. GitHub Pages does not run the Go services,
hold local snapshots, or provide live radio access. Those remain in the local app.
GitHub provides the hosting infrastructure; the source contains no added tracking.

See GitHub's [custom Pages workflow guide](https://docs.github.com/en/pages/getting-started-with-github-pages/using-custom-workflows-with-github-pages).
