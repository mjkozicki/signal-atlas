"""BLE advertisement discovery only. No connection, pairing, or GATT operations."""
import argparse
import asyncio
import importlib.util
import json
import platform
import sys
from datetime import datetime, timezone


def stamp():
    return datetime.now(timezone.utc).isoformat()


def status():
    available = importlib.util.find_spec('bleak') is not None
    return {'protocol': 'bluetooth', 'provider': 'Bleak / ' + platform.system(),
            'available': available, 'readers': [],
            'message': 'Provider installed. Adapter power and Bluetooth permission are checked when a scan starts.' if available else 'Install native providers with make native-setup. Bluetooth requires Bleak and OS permission.'}


async def scan(duration, adapter):
    from bleak import BleakScanner
    devices = {}
    overflow = False

    def observe(device, advertisement):
        nonlocal overflow
        identity = device.address
        if identity not in devices and len(devices) >= 10000:
            overflow = True
            return
        now = stamp()
        old = devices.get(identity)
        rssi = advertisement.rssi
        if rssi == 127 or not -127 <= rssi <= 20:
            rssi = None
        devices[identity] = {
            'id': identity,
            'name': (advertisement.local_name or device.name or 'Unnamed BLE device')[:512],
            'first_seen': old['first_seen'] if old else now, 'last_seen': now,
            'observations': old['observations'] + 1 if old else 1, 'rssi': rssi,
            'details': {
                'transport': 'BLE',
                'identifier_type': 'CoreBluetooth UUID' if sys.platform == 'darwin' else 'Bluetooth address',
                'service_uuids': sorted(advertisement.service_uuids),
                'manufacturer_data': {str(k): bytes(v).hex() for k, v in advertisement.manufacturer_data.items()},
                'service_data': {k: bytes(v).hex() for k, v in advertisement.service_data.items()},
                'tx_power': advertisement.tx_power,
                'identity_note': 'Identifier observed by this OS; address rotation or platform UUIDs prevent guaranteed physical-device tracking.',
                'scan_mode': 'OS-managed active discovery; no peripheral connections',
            },
        }

    kwargs = {}
    if adapter:
        if sys.platform != 'linux':
            raise ValueError('Named adapters are supported only by the Linux BlueZ provider.')
        kwargs['bluez'] = {'adapter': adapter}
    async with BleakScanner(detection_callback=observe, scanning_mode='active', **kwargs):
        await asyncio.sleep(duration)
    warnings = ['BLE advertisements only; Bluetooth Classic, pairing security, and GATT characteristics are not inspected.',
                'Discovery can transmit scan requests. Observed identifiers are not proof of ownership or a stable identity.']
    if overflow:
        warnings.append('Device limit reached; additional advertisements were not retained.')
    return {'devices': sorted(devices.values(), key=lambda d: (d['name'], d['id'])), 'warnings': warnings}


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--action', choices=['status', 'scan'], required=True)
    parser.add_argument('--duration', type=int, default=10)
    parser.add_argument('--adapter', default='')
    parser.add_argument('--reader', default='')
    args = parser.parse_args()
    if not 0 <= args.duration <= 60:
        raise ValueError('Duration exceeds 60 seconds.')
    result = status() if args.action == 'status' else asyncio.run(scan(args.duration, args.adapter))
    print(json.dumps(result))


if __name__ == '__main__':
    try:
        main()
    except Exception as exc:
        print(f'{type(exc).__name__}: {exc}', file=sys.stderr)
        sys.exit(1)
