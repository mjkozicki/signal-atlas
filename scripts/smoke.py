#!/usr/bin/env python3
"""Hardware-free CLI smoke checks with temporary databases and output files."""
import json
from pathlib import Path
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parents[1]


def run(binary, *args):
    executable = ROOT / 'bin' / binary
    return subprocess.check_output([str(executable), *map(str, args)], cwd=ROOT, text=True)


def main():
    with tempfile.TemporaryDirectory(prefix='signal-atlas-smoke-') as directory:
        temp = Path(directory)
        wifi = json.loads(run('wifi-scan', 'demo', '--db', temp / 'wifi.db'))
        assert wifi['access_points'] and wifi['findings']
        run('wifi-scan', 'export', '--db', temp / 'wifi.db', '--format', 'html', '--output', temp / 'wifi.html')
        assert '<html' in (temp / 'wifi.html').read_text().lower()
        for protocol in ('bluetooth', 'nfc'):
            db = temp / f'{protocol}.db'
            scan = json.loads(run(f'{protocol}-scan', 'demo', '--db', db))
            assert scan['mode'] == 'demo' and scan['state'] == 'completed' and scan['devices']
            exported = temp / f'{protocol}.json'
            run(f'{protocol}-scan', 'export', '--db', db, '--format', 'json', '--output', exported)
            imported = json.loads(run(f'{protocol}-scan', 'import', '--db', db, '--file', exported))
            assert imported['id'] != scan['id']
            assert json.loads(run(f'{protocol}-scan', 'diff', '--db', db, '--from', scan['id'], '--to', imported['id'])) == []
            print(f'{protocol}: demo / export / import / comparison passed')
        print('Wi-Fi: demo / HTML export passed; no hardware scanned')


if __name__ == '__main__':
    main()
