"""BLE callbacks exercised with a fake scanner; no OS adapter is created."""
import asyncio
import importlib.util
from pathlib import Path
from types import SimpleNamespace
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location('ble_provider', Path(__file__).with_name('bleak.py'))
provider = importlib.util.module_from_spec(spec)
spec.loader.exec_module(provider)

class DiscoveryTests(unittest.TestCase):
    def test_callbacks_and_adapter(self):
        seen = {}
        class Scanner:
            def __init__(self, **kwargs): seen.update(kwargs)
            async def __aenter__(self):
                device = SimpleNamespace(address='AA:BB:CC:DD:EE:FF', name='Fallback')
                adv = SimpleNamespace(local_name='Sensor', rssi=-55, service_uuids=['180f'], manufacturer_data={76: b'\x01\x02'}, service_data={'180f': b'\x64'}, tx_power=-4)
                seen['detection_callback'](device, adv)
                adv.rssi = -60
                seen['detection_callback'](device, adv)
            async def __aexit__(self, *args): pass
        with patch.dict('sys.modules', {'bleak': SimpleNamespace(BleakScanner=Scanner)}), patch.object(provider.sys, 'platform', 'linux'):
            result = asyncio.run(provider.scan(0, 'hci1'))
        self.assertEqual(seen['bluez'], {'adapter': 'hci1'})
        self.assertEqual(seen['scanning_mode'], 'active')
        self.assertEqual(len(result['devices']), 1)
        device = result['devices'][0]
        self.assertEqual(device['observations'], 2)
        self.assertEqual(device['rssi'], -60)
        self.assertEqual(device['details']['manufacturer_data'], {'76': '0102'})
        self.assertEqual(device['details']['service_data'], {'180f': '64'})
        self.assertLessEqual(device['first_seen'], device['last_seen'])

    def test_adapter_rejected_on_macos(self):
        with patch.dict('sys.modules', {'bleak': SimpleNamespace(BleakScanner=None)}), patch.object(provider.sys, 'platform', 'darwin'):
            with self.assertRaisesRegex(ValueError, 'Linux'):
                asyncio.run(provider.scan(0, 'hci0'))

    def test_status_does_not_construct_scanner(self):
        with patch.object(provider.importlib.util, 'find_spec', return_value=object()):
            self.assertTrue(provider.status()['available'])
        with patch.object(provider.importlib.util, 'find_spec', return_value=None):
            self.assertFalse(provider.status()['available'])

if __name__ == '__main__': unittest.main()
