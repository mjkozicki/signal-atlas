"""Generate deterministic, synthetic radiotap management captures; no radio access."""
from pathlib import Path
import struct

ROOT = Path(__file__).resolve().parent

def ie(kind, payload):
    return bytes([kind, len(payload)]) + payload

def rsn(akms, pmf):
    suite = lambda kind: b'\x00\x0f\xac' + bytes([kind])
    return struct.pack('<H', 1) + suite(4) + struct.pack('<H', 1) + suite(4) + struct.pack('<H', len(akms)) + b''.join(suite(a) for a in akms) + struct.pack('<H', pmf)

def beacon(ssid, mac, freq, ch, rssi, akms, pmf):
    rt = struct.pack('<BBHIHHb', 0, 0, 13, 0x28, freq, 0, rssi)
    addr = bytes.fromhex(mac.replace(':', ''))
    header = b'\x80\x00\x00\x00' + b'\xff'*6 + addr + addr + b'\x00\x00'
    fixed = struct.pack('<QHH', 0, 100, 0x11 if akms else 1)
    elements = ie(0, ssid.encode()) + ie(3, bytes([ch]))
    if akms:
        elements += ie(48, rsn(akms, pmf))
    elements += ie(45, b'\x00'*26) + ie(255, b'\x23'+b'\x00'*21)
    return rt + header + fixed + elements

def write(name, packets):
    raw = struct.pack('<IHHIIII', 0xa1b2c3d4, 2, 4, 0, 0, 65535, 127)
    for n, packet in enumerate(packets):
        raw += struct.pack('<IIII', 1789239600+n, 0, len(packet), len(packet)) + packet
    (ROOT/name).write_bytes(raw)

before = [
    beacon('Studio', '02:00:00:00:01:01', 5180, 36, -42, [8], 192),
    beacon('Studio', '02:00:00:00:01:02', 6135, 37, -56, [8], 192),
    beacon('Studio · IoT', '02:00:00:00:02:01', 2437, 6, -61, [2], 128),
    beacon('Studio · Guest', '02:00:00:00:03:01', 5745, 149, -51, [2,8], 128),
    beacon('Studio', '02:00:00:00:09:01', 5220, 44, -69, [2], 0),
    beacon("Neighbor's Wi-Fi", '02:00:00:00:08:01', 2437, 6, -77, [], 0),
]
after = before[:2] + [
    beacon('Studio · IoT', '02:00:00:00:02:01', 2437, 6, -61, [2], 192),
    beacon('Studio · Guest', '02:00:00:00:03:01', 5745, 149, -51, [8], 192),
] + before[5:]
write('before.pcap', before)
write('after.pcap', after)
print('Wrote synthetic before.pcap and after.pcap')
