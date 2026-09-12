# Synthetic fixtures

All fixtures are synthetic; none contain captured personal devices or tags.

- `before.pcap`, `after.pcap`, and related Wi-Fi captures exercise the Wi-Fi parser
  and rules. `generate.py` regenerates them; `authorized.yaml` is the sample inventory.
- `bluetooth/demo.json` is a completed demo snapshot with two BLE observations.
- `nfc/demo.json` is a completed demo snapshot with a text and URI NDEF tag.

Import the JSON files into the matching dashboard view or CLI. Import assigns a
new snapshot ID and preserves the explicit `demo` mode:

```bash
./bin/bluetooth-scan import --file fixtures/bluetooth/demo.json --db .data/bluetooth-demo.db
./bin/nfc-scan import --file fixtures/nfc/demo.json --db .data/nfc-demo.db
```

The CLI `demo` commands produce equivalent observations with current timestamps.
The checked-in JSON timestamps are fixed for repeatable imports.
