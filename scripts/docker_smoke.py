#!/usr/bin/env python3
"""Exercise Docker using an isolated project/volume and synthetic observations."""
import json
import os
from pathlib import Path
import re
import socket
import subprocess
import tempfile
from urllib.error import HTTPError
from urllib.request import build_opener, ProxyHandler, Request
import uuid

ROOT = Path(__file__).resolve().parents[1]


def main():
    project = 'signal-atlas-smoke-' + uuid.uuid4().hex[:12]
    with socket.socket() as sock:
        sock.bind(('127.0.0.1', 0))
        port = sock.getsockname()[1]
    env = {**os.environ, 'SIGNAL_ATLAS_PORT': str(port)}
    command = ['docker', 'compose', '-f', str(ROOT / 'compose.yaml'), '-p', project]
    base = f'http://127.0.0.1:{port}'
    opener = build_opener(ProxyHandler({}))

    def compose(*args, check=True):
        return subprocess.run(command + list(args), cwd=ROOT, env=env, check=check)

    def request(path, method='GET', data=None, headers=None, expected=200):
        supplied = {'X-Wifi-Scanner': 'local', 'Origin': base}
        if headers:
            supplied.update(headers)
        if isinstance(data, dict):
            data = json.dumps(data).encode()
            supplied['Content-Type'] = 'application/json'
        req = Request(base + path, data=data, headers=supplied, method=method)
        try:
            response = opener.open(req, timeout=10)
        except HTTPError as error:
            response = error
        with response:
            body = response.read()
            assert response.status == expected, (path, response.status, body)
            return body

    def api(path, **kwargs):
        return json.loads(request(path, **kwargs))

    try:
        subprocess.run(
            ['npm', 'run', 'docker', '--', '--wait-timeout', '90'], cwd=ROOT,
            env={**env, 'COMPOSE_FILE': str(ROOT / 'compose.yaml'),
                 'COMPOSE_PROJECT_NAME': project}, check=True,
        )
        html = request('/').decode()
        assert 'Signal Atlas' in html
        assets = re.findall(r'(?:src|href)="(/assets/[^" ]+)"', html)
        assert assets, 'embedded production assets missing'
        for asset in assets:
            assert request(asset)
        paths = ['/api/scans', '/api/bluetooth/scans', '/api/nfc/scans']
        snapshots = {}
        for path in paths:
            rows = api(path)
            assert len(rows) == 1, (path, rows)
            snapshot = rows[0]['id']
            snapshots[path] = snapshot
            export = path.replace('/scans', '/export/') + snapshot + '?format=json'
            assert api(export)['id'] == snapshot
            for headers in ({'Host': 'evil.example'}, {'Origin': 'https://evil.example'},
                            {'X-Wifi-Scanner': ''}):
                request(path + '/' + snapshot, method='PATCH', data={'hidden': True},
                        headers=headers, expected=403)
            api(path + '/' + snapshot, method='PATCH', data={'hidden': True})
            assert api(path) == []
        compose('down')  # Recreate the container while preserving its named volume.
        compose('up', '-d', '--wait', '--wait-timeout', '90')
        for path, snapshot in snapshots.items():
            assert api(path) == [], 'hidden demo was reseeded on restart'
            rows = api(path + '?include_hidden=true')
            assert len(rows) == 1 and rows[0]['id'] == snapshot
            api(path + '/' + snapshot, method='PATCH', data={'hidden': False})
            assert api(path)[0]['id'] == snapshot
            api(path + '/' + snapshot, method='DELETE')
            assert api(path + '?include_hidden=true') == []
        imported = api('/api/import', method='POST', data=(ROOT / 'fixtures/before.pcap').read_bytes())
        assert imported['access_points']
        for protocol in ('bluetooth', 'nfc'):
            fixture = json.loads((ROOT / 'fixtures' / protocol / 'demo.json').read_text())
            imported = api(f'/api/{protocol}/import', method='POST', data=fixture, expected=201)
            assert imported['devices']
        with tempfile.TemporaryDirectory(prefix='atlas-compose-') as directory:
            override = Path(directory) / 'empty.yaml'
            override.write_text('services:\n  atlas:\n    command: ["--demo=false"]\n')
            compose('-f', str(override), 'up', '-d', '--wait', '--wait-timeout', '90')
            for path in paths:
                assert api(path) == [], 'normal workspace reused demo observations'
        compose('up', '-d', '--wait', '--wait-timeout', '90')
        for path in paths:
            assert len(api(path)) == 1, 'demo workspace was lost during mode switch'
        print('Docker: embedded UI, all three APIs, imports/exports, request boundaries, '
              'hide/restore/delete, volume persistence, and workspace switching passed; '
              'no hardware scanned.', flush=True)
    except BaseException:
        compose('logs', '--no-color', '--tail=100', check=False)
        raise
    finally:
        # The unique project owns only disposable synthetic test observations.
        compose('down', '--volumes', '--remove-orphans')


if __name__ == '__main__':
    main()
