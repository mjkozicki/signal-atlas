# Attribution

**Signal Atlas** was created by **Michael Kozicki**, with implementation and
documentation assistance from OpenAI Codex. Copyright © 2026 Michael Kozicki.
The original project code, documentation, and synthetic fixtures are provided
under the [MIT License](LICENSE).

The project began with a Wi-Fi scanner design discussed in this
[shared design conversation](https://chatgpt.com/share/6aa5b3ac-71b8-83e9-8a51-8e0b33e1b0fa),
then expanded into independent Wi-Fi, BLE, and NFC services. The repository includes
the implementation and documentation needed to use it without access to that link.

## Open-source building blocks

- **gopacket** contributors: packet capture parsing.
- **modernc.org/sqlite** and its dependencies: pure-Go SQLite support; SQLite itself
  includes public-domain code.
- **Go** authors: the standard library and runtime.
- **Svelte**, **Vite**, and **TypeScript** contributors: the dashboard and build tools.
- **Lucide** contributors and **Cole Bemis / Feather**: interface icons. Lucide’s
  license includes the Feather attribution for derived icons.
- **Bleak** contributors: optional native Bluetooth Low Energy discovery.
- **pyscard** contributors: optional PC/SC reader access.

See [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md) and [licenses/](licenses/) for
versions and reproduced upstream notices. These components retain their own licenses;
the MIT license for Signal Atlas does not replace them. In particular, the optional
pyscard dependency is distributed under the LGPL 2.1 terms supplied with it. It is
installed separately into the user's virtual environment, not vendored in this source.

Technical documentation from Wireshark, Bluetooth SIG, NFC Forum, PC/SC, and adapter
providers informed the implementation; references are linked in the relevant guides.
No endorsement or affiliation is implied. All checked-in captures, tag records, and
advertisement fixtures are synthetic.
