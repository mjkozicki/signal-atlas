# NFC service

`nfc-scan` polls an attached **PC/SC contactless reader** using pyscard. It runs as a
separate process with its own SQLite database and API. NFC is a close-range tag
presentation workflow; it does not discover distant devices like Bluetooth.
A laptop's Bluetooth/Wi-Fi adapter is not an NFC reader.

## Setup and use

From `signal-atlas/`:

```bash
make build
make native-setup
./bin/nfc-scan status
./bin/nfc-scan discover --duration 10
./bin/nfc-scan discover --duration 30 --reader 'EXACT NAME FROM STATUS'
./bin/nfc-scan serve                       # 127.0.0.1:8789
```

Connect a USB PC/SC-compatible contactless reader and install its required driver.
macOS uses the PC/SC framework. Linux requires the PC/SC service (`pcscd`), reader
driver (commonly CCID), and build headers when installing pyscard. Windows uses the
Smart Card service and the reader's PC/SC driver. Python 3.10+ and a C compiler may
be needed to install pyscard; `make native-setup` installs both native providers into
`.venv`. Windows users can create a virtual environment and install
`requirements-native.txt` with their Python interpreter directly.

`status` lists PC/SC readers without scanning tags. PC/SC may include contact-only
smart-card readers, so a listed reader is not proof of NFC support. With multiple
readers, select the exact reader name. The dashboard provides that selector.
A missing driver, unplugged reader, inaccessible tag, or unsupported command is
reported explicitly. Dependencies installed successfully on macOS arm64; actual
USB readers and physical tags have not yet been tested with this implementation.

```bash
./bin/nfc-scan demo --db .data/nfc-demo.db
./bin/nfc-scan history
./bin/nfc-scan inspect --scan latest
./bin/nfc-scan export --format json --output nfc.json
./bin/nfc-scan export --format csv --output nfc.csv
./bin/nfc-scan import --file nfc.json
./bin/nfc-scan diff --from SCAN_ID --to latest
```

The default database is `.data/nfc.db`; each command accepts `--db PATH` and native
commands accept startup-only `--python PATH`. The **NFC** dashboard view works when
this service and `wifi-scan serve` are running. `make serve-all` starts all three;
`make demo-all` seeds separate databases without using any radio or reader.

## Supported read operations

- Reader identity, ATR, and reader-reported UID using the fixed `FF CA 00 00 00`
  command when supported. A missing UID receives a per-presentation identifier.
- NFC Forum Type 4 NDEF: select the standard NDEF application and capability
  container, require freely readable access, select the advertised file, then use
  bounded `READ BINARY` commands. The supported profile uses a two-byte NLEN and
  a standard NDEF file-control TLV in the first 15 capability-container bytes.
- ACR122 Type 2 fallback: fixed `FF B0` memory reads, capability-container checks,
  and NDEF TLV extraction. This fallback is enabled only for reader names containing
  `ACR122`. It supports up to 1008 bytes with one-byte page addressing and does not
  interpret dynamic lock/memory-control layouts or sector selection.
- NDEF payloads up to 4096 bytes and 64 unchunked records. Text records decode UTF-8
  or UTF-16; URI records decode the standard prefix table. Other records preserve
  type, ID, and payload hex. Malformed NDEF retains raw data and a parse warning.

These are specific read profiles, not universal NFC compatibility. Type 1, Type 3,
Type 5, extended Type 4 files, protected sectors, MIFARE Classic authentication,
card emulation, payment applications, tag writing, and arbitrary APDU execution
are not implemented. Some tags will yield a UID/ATR and an NDEF-unavailable note.

Polling energizes the reader and sends select/read commands. It reads once per
presentation and updates presence timestamps until removal; observation count is
the number of presentations, not an RF packet count. Cancellation terminates the
provider subprocess and stores a canceled state. Results appear after the bounded
1–60 second scan completes; unfinished tag observations are not saved.

UIDs can be duplicated or randomized; ATR describes card characteristics and is
not a unique identity. No ownership, authenticity, or security conclusion is drawn.
Tag text and URLs are displayed as data; the app never opens tag URLs automatically.
Raw NDEF payloads and decoded records are retained in the local, unencrypted database.

See [architecture.md](architecture.md) for APIs and persistence. Technical references:
[pyscard user guide](https://pyscard.sourceforge.io/user-guide.html),
[PC/SC API](https://pcsclite.apdu.fr/api/group__API.html), and
[ACR122U API manual](https://downloads.acs.com.hk/drivers/en/API-ACR122U-2.02.pdf).

## Manage saved snapshots

Use Hide or Delete beside a snapshot or in History. Show hidden snapshots reveals
hidden entries so they can be restored. Hidden entries are excluded from normal
history and the default latest selection. Delete confirms the exact snapshot and
cannot be undone; it does not delete exported files. Running scans must finish or
be canceled first. Visibility persists across restart, including for demo data.
