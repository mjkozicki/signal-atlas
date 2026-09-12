"""Bounded PC/SC polling with UID and standard NDEF reads. No write/authenticate APDUs."""
import argparse
import importlib.util
import json
import sys
import time
import uuid
from datetime import datetime, timezone

MAX_NDEF = 4096


def stamp():
    return datetime.now(timezone.utc).isoformat()


def status():
    result = {'protocol': 'nfc', 'provider': 'PC/SC (pyscard)', 'available': False, 'readers': []}
    if importlib.util.find_spec('smartcard') is None:
        result['message'] = 'Install native providers with make native-setup, then connect a PC/SC-compatible NFC reader.'
        return result
    try:
        from smartcard.System import readers
        result['readers'] = [str(r) for r in readers()]
        result['available'] = bool(result['readers'])
        result['message'] = 'Select a contactless NFC reader. PC/SC may also list contact-only smart-card readers.' if result['readers'] else 'No PC/SC readers detected. Connect an NFC reader and check its driver.'
    except Exception as exc:
        result['message'] = f'PC/SC unavailable: {exc}'
    return result


def apdu(connection, command):
    data, sw1, sw2 = connection.transmit(list(command))
    if (sw1, sw2) != (0x90, 0x00):
        raise ValueError(f'Read/select command not supported or accessible (status {sw1:02x}{sw2:02x}).')
    return bytes(data)


def read_binary(connection, offset, length, max_read=240):
    if offset < 0 or offset + length > MAX_NDEF + 2:
        raise ValueError('NDEF read exceeds limit.')
    out = bytearray()
    while len(out) < length:
        pos = offset + len(out)
        want = min(240, max_read, length - len(out))
        if want < 1:
            raise ValueError('Invalid maximum read length.')
        part = apdu(connection, [0x00, 0xB0, pos >> 8, pos & 255, want])
        if len(part) != want:
            raise ValueError('Truncated NDEF file read.')
        out.extend(part)
    return bytes(out)


def type4_ndef(connection):
    apdu(connection, bytes.fromhex('00a4040007d276000085010100'))
    apdu(connection, bytes.fromhex('00a4000c02e103'))
    cc = read_binary(connection, 0, 15)
    if int.from_bytes(cc[:2], 'big') < 15 or cc[7:9] != b'\x04\x06':
        raise ValueError('Unsupported Type 4 capability container.')
    if cc[13] != 0:
        raise ValueError('NDEF file requires read authorization.')
    maximum = int.from_bytes(cc[11:13], 'big')
    max_read = int.from_bytes(cc[3:5], 'big')
    if max_read < 15:
        raise ValueError('Unsupported Type 4 maximum read length.')
    apdu(connection, bytes.fromhex('00a4000c02') + cc[9:11])
    length = int.from_bytes(read_binary(connection, 0, 2), 'big')
    if length > MAX_NDEF or length + 2 > maximum:
        raise ValueError('NDEF file length exceeds capability or 4096-byte limit.')
    return read_binary(connection, 2, length, max_read)


def extract_tlv(data):
    offset = 0
    while offset < len(data):
        kind = data[offset]
        offset += 1
        if kind == 0:
            continue
        if kind == 0xFE:
            return b''
        if offset >= len(data):
            raise ValueError('Truncated Type 2 TLV length.')
        length = data[offset]
        offset += 1
        if length == 255:
            if offset + 2 > len(data):
                raise ValueError('Truncated extended TLV length.')
            length = int.from_bytes(data[offset:offset+2], 'big')
            offset += 2
        if offset + length > len(data):
            raise ValueError('Type 2 TLV exceeds readable memory.')
        if kind == 3:
            return bytes(data[offset:offset+length])
        offset += length
    return b''


def type2_ndef(connection):
    cc = apdu(connection, [0xFF, 0xB0, 0, 3, 16])
    if len(cc) != 16 or cc[0] != 0xE1 or cc[3] >> 4 != 0:
        raise ValueError('No freely readable Type 2 capability container.')
    size = cc[2] * 8
    if not 0 < size <= min(MAX_NDEF, 1008):
        raise ValueError('Type 2 memory exceeds supported one-byte page addressing.')
    memory = bytearray()
    for page in range(4, 4 + (size + 3)//4, 4):
        block = apdu(connection, [0xFF, 0xB0, 0, page, 16])
        if len(block) != 16:
            raise ValueError('Truncated Type 2 memory read.')
        memory.extend(block)
    return extract_tlv(memory[:size])


def inspect_reader(reader):
    connection = reader.createConnection()
    connection.connect()
    warnings = []
    try:
        details = {'reader': str(reader), 'atr': bytes(connection.getATR()).hex(), 'technology': 'PC/SC contactless candidate'}
        try:
            uid = apdu(connection, [0xFF, 0xCA, 0, 0, 0]).hex()
            if not uid:
                raise ValueError('Empty UID response')
            details['uid'] = uid
        except Exception as exc:
            warnings.append(f'UID unavailable: {exc}')
        try:
            ndef = type4_ndef(connection)
            details['technology'] = 'NFC Type 4 NDEF'
            details['ndef_hex'] = ndef.hex()
        except Exception as type4_error:
            # This pseudo-APDU memory profile is documented for ACR122 readers.
            if 'ACR122' in str(reader).upper():
                try:
                    ndef = type2_ndef(connection)
                    details['technology'] = 'NFC Type 2 NDEF (ACR122 profile)'
                    details['ndef_hex'] = ndef.hex()
                except Exception as type2_error:
                    warnings.append(f'NDEF unavailable: {type2_error}')
            else:
                warnings.append(f'NDEF unavailable: {type4_error}')
        details['read_notes'] = warnings
        details['identity_note'] = 'UIDs can be randomized or duplicated; ATR is not a unique tag identity.'
        details['identifier_type'] = 'Reader-reported UID' if details.get('uid') else 'Per-presentation identifier (UID unavailable)'
        identifier = str(reader) + ':' + details.get('uid', 'unresolved:' + uuid.uuid4().hex)
        now = stamp()
        return {'id': identifier, 'name': 'NFC tag ' + details.get('uid', '(UID unavailable)'), 'first_seen': now,
                'last_seen': now, 'observations': 1, 'rssi': None, 'details': details}
    finally:
        connection.disconnect()


def scan(duration, reader_name):
    from smartcard.System import readers
    from smartcard.scard import (SCardEstablishContext, SCardReleaseContext, SCardGetStatusChange,
                                 SCARD_SCOPE_USER, SCARD_S_SUCCESS, SCARD_E_TIMEOUT,
                                 SCARD_STATE_PRESENT, SCARD_STATE_EMPTY, SCARD_STATE_UNAVAILABLE,
                                 SCARD_STATE_UNKNOWN)
    choices = readers()
    if not choices:
        raise ValueError('No PC/SC reader found. Connect a supported NFC reader.')
    if reader_name:
        choices = [r for r in choices if str(r) == reader_name]
        if not choices:
            raise ValueError('Selected reader is not available; choose an exact name from status.')
    elif len(choices) != 1:
        raise ValueError('Multiple PC/SC readers detected; select an exact reader name.')
    reader = choices[0]
    rc, context = SCardEstablishContext(SCARD_SCOPE_USER)
    if rc != SCARD_S_SUCCESS:
        raise RuntimeError(f'Cannot establish PC/SC context: {rc}')
    devices = {}
    current_state = 0
    present_id = None
    deadline = time.monotonic() + duration
    warnings = ['NFC polling energizes the reader and exchanges read/select commands; no tag writes or authentication are performed.']
    try:
        while time.monotonic() < deadline:
            rc, states = SCardGetStatusChange(context, 200, [(str(reader), current_state)])
            if rc == SCARD_E_TIMEOUT:
                if present_id in devices:
                    devices[present_id]['last_seen'] = stamp()
                continue
            if rc != SCARD_S_SUCCESS:
                raise RuntimeError(f'PC/SC polling failed: {rc}')
            _, current_state, _ = states[0]
            if current_state & (SCARD_STATE_UNAVAILABLE | SCARD_STATE_UNKNOWN):
                raise RuntimeError('The selected reader became unavailable.')
            if current_state & SCARD_STATE_EMPTY:
                present_id = None
            if current_state & SCARD_STATE_PRESENT and present_id is None:
                try:
                    item = inspect_reader(reader)
                    previous = devices.get(item['id'])
                    if previous:
                        item['first_seen'] = previous['first_seen']
                        item['observations'] += previous['observations']
                    devices[item['id']] = item
                    present_id = item['id']
                except Exception as exc:
                    if len(warnings) < 100:
                        warnings.append(f'Tag inspection failed: {exc}')
                    present_id = 'unreadable'
            if present_id in devices:
                devices[present_id]['last_seen'] = stamp()
        return {'devices': list(devices.values()), 'warnings': warnings}
    finally:
        SCardReleaseContext(context)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--action', choices=['status', 'scan'], required=True)
    parser.add_argument('--duration', type=int, default=10)
    parser.add_argument('--reader', default='')
    parser.add_argument('--adapter', default='')
    args = parser.parse_args()
    if not 0 <= args.duration <= 60:
        raise ValueError('Duration exceeds 60 seconds.')
    result = status() if args.action == 'status' else scan(args.duration, args.reader)
    print(json.dumps(result))


if __name__ == '__main__':
    try:
        main()
    except Exception as exc:
        print(f'{type(exc).__name__}: {exc}', file=sys.stderr)
        sys.exit(1)
