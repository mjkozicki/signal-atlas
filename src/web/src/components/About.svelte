<script lang="ts">
  import { BookOpen, Radio, Wifi, Bluetooth, Nfc, Search, ArrowUpRight, LockKeyhole } from '@lucide/svelte';
  let signal = $state(-60);
  let query = $state('');
  let category = $state('All');
  const categories = ['All', 'Wi-Fi', 'Bluetooth', 'NFC', 'Reports'];
  const terms = [
    { category: 'Wi-Fi', name: 'SSID · network name', text: 'The name a Wi-Fi network advertises, such as Studio. Several access points can share one SSID. A matching name alone does not establish ownership; hidden SSIDs omit the name from some announcements.' },
    { category: 'Wi-Fi', name: 'AP · access point', text: 'A radio endpoint that clients use to join a wireless network. A home router often contains several AP interfaces, alongside routing and other functions. An AP count is not a count of physical routers or connected people.' },
    { category: 'Wi-Fi', name: 'BSSID · basic service set identifier', text: 'The MAC-style identifier for a Wi-Fi basic service set, shown as six hexadecimal pairs. One physical AP can advertise multiple BSSIDs. This app authorizes an AP only when both its SSID and exact BSSID match your inventory.' },
    { category: 'Wi-Fi', name: 'MAC address', text: 'A link-layer network identifier, often written like 02:11:22:33:44:55. It is different from an IP address. Addresses can be changed or spoofed; they are not proof of identity or ownership.' },
    { category: 'Wi-Fi', name: 'RF · radio frequency', text: 'Radio signals used for wireless communication. The RF environment view summarizes the access points in a capture. Its curves are arranged by network name, not along a calibrated frequency axis.' },
    { category: 'Wi-Fi', name: 'Beacon & probe response', text: 'Wi-Fi management frames that announce a network and its capabilities. A beacon is a periodic announcement; a probe response answers a discovery request. Reading a response in a capture does not mean this scanner sent the request.' },
    { category: 'Wi-Fi', name: 'PCAP / PCAPNG & radiotap', text: 'PCAP and PCAPNG are packet-capture file formats. This app needs raw 802.11 frames or radiotap headers carrying Wi-Fi metadata such as signal and frequency. An ordinary Ethernet/IP capture cannot provide these radio and security announcements.' },
    { category: 'Wi-Fi', name: 'Monitor mode', text: 'An adapter mode for receiving wireless frames on a selected channel. Live Wi-Fi capture here requires a preconfigured Linux monitor interface. The app does not change adapter mode or hop between channels.' },
    { category: 'Wi-Fi', name: 'WPA / WPA2 / WPA3 & RSN', text: 'Wi-Fi security families. RSN means Robust Security Network; its advertised information describes supported authentication, ciphers, and protection settings. The label reflects the announcement, not a test of a client connection or password strength.' },
    { category: 'Wi-Fi', name: 'AKM, PSK, SAE & Enterprise', text: 'AKM means Authentication and Key Management. PSK uses a pre-shared key, usually derived from a shared password. SAE is the password-authenticated exchange used by WPA3-Personal. Enterprise commonly uses 802.1X/EAP and an authentication server; beacons do not reveal whether clients validate server certificates correctly.' },
    { category: 'Wi-Fi', name: 'Transition mode', text: 'An advertised compatibility mode accepting both WPA2-Personal and WPA3-Personal clients. Seeing SAE alongside PSK does not mean every connected client uses WPA3.' },
    { category: 'Wi-Fi', name: 'Cipher · CCMP, GCMP, TKIP', text: 'The algorithm suite protecting Wi-Fi data. CCMP and GCMP use AES; TKIP is a legacy suite. Advertised ciphers show capabilities, not which cipher every client negotiated.' },
    { category: 'Wi-Fi', name: 'PMF · Protected Management Frames', text: 'Protection for certain management frames. Capable means supported but not necessarily required; required means clients must use it; disabled means the advertised capability is absent. Unknown means the capture cannot establish the setting. PMF does not prevent all wireless interference.' },
    { category: 'Wi-Fi', name: 'Open, OWE & WEP / unknown legacy', text: 'Open indicates no advertised WPA/RSN data protection. OWE (Opportunistic Wireless Encryption) adds encryption without authenticating the network to the user. A privacy flag without decodable WPA/RSN is labeled WEP / unknown legacy here because that flag alone cannot conclusively identify WEP.' },
    { category: 'Wi-Fi', name: 'IP address, port & TCP reachability', text: 'An IP address identifies a network endpoint; a port selects a service on it, such as TCP 443. The optional CLI audit only checks whether explicitly allowed TCP endpoints accept a connection. Reachability is not evidence that a service is vulnerable.' },
    { category: 'Bluetooth', name: 'BLE · Bluetooth Low Energy', text: 'The Bluetooth transport used by this discovery service. Bluetooth Classic discovery, pairing, connections, and inspection of connected-device data are outside its scope.' },
    { category: 'Bluetooth', name: 'Advertisement & scan response', text: 'Short discovery messages that can carry a name, service identifiers, or data. Active BLE scanning may transmit scan requests to obtain additional advertisement data. This app does not pair with or connect to peripherals.' },
    { category: 'Bluetooth', name: 'Device identifier & private address', text: 'An OS-reported identifier for an observed advertiser. macOS uses CoreBluetooth UUIDs; other backends commonly report Bluetooth addresses. Randomized addresses and OS behavior can make one device appear under different identifiers.' },
    { category: 'Bluetooth', name: 'Service UUID & GATT', text: 'A UUID (Universally Unique Identifier) names a service type or other object. An advertised service UUID suggests a supported function; it is not a unique device identity. GATT organizes services and characteristics after connection, and this scanner does not read those characteristics.' },
    { category: 'Bluetooth', name: 'Manufacturer data & service data', text: 'Advertisement bytes associated with a company identifier or service UUID. The app preserves them as hexadecimal data. A company identifier is a protocol field, not a verified manufacturer attribution; custom payload meaning requires its format specification.' },
    { category: 'Bluetooth', name: 'TX power', text: 'Transmit power reported in an advertisement, when available, in dBm. It describes a transmit-side value; RSSI is measured at the receiver. These values alone do not provide a calibrated distance estimate.' },
    { category: 'NFC', name: 'NFC · Near Field Communication', text: 'A close-range exchange between a reader and a presented tag. This workflow requires a compatible contactless reader; a computer’s Wi-Fi or Bluetooth adapter cannot substitute for one.' },
    { category: 'NFC', name: 'PC/SC & reader', text: 'PC/SC is the smart-card interface used to communicate with supported readers. A listed reader can be contact-only, so availability does not guarantee NFC support. Select the exact contactless reader when more than one is present.' },
    { category: 'NFC', name: 'UID & ATR', text: 'UID is a reader-reported tag identifier; it may be duplicated or randomized. ATR (Answer To Reset) is reader-reported card information, not a unique identity or the tag’s text. When a UID cannot be read, this app assigns an identifier for that presentation.' },
    { category: 'NFC', name: 'NDEF · NFC Data Exchange Format', text: 'A standard container for tag records. Text records contain text and a language code; URI records hold an identifier such as a web address. Unknown record types retain payload bytes. Reading a URI does not verify the destination or open it.' },
    { category: 'NFC', name: 'Type 2 / Type 4 & APDU', text: 'Tag types define different communication and storage profiles. This app reads a supported Type 4 NDEF profile and has an ACR122 Type 2 fallback. APDU means Application Protocol Data Unit: a command/response exchanged with the reader or card. Only fixed select/read operations are used here.' },
    { category: 'NFC', name: 'Hex, payload & read notes', text: 'Hexadecimal writes a byte as two base-16 digits, such as 04 or ff. A payload is the carried data. Read notes explain unsupported or inaccessible content; an NDEF read failure can coexist with a successfully observed UID. There is no NFC RSSI measurement in this app.' },
    { category: 'Reports', name: 'Snapshot, demo, live & imported', text: 'A snapshot saves one scan’s observations. Demo is synthetic and does not access hardware. Live is collected by a native provider. Imported means loaded from a file; Bluetooth/NFC imports preserve the file’s declared demo/live mode without verifying its provenance.' },
    { category: 'Reports', name: 'Authorized, neighbor & review AP', text: 'Authorized means an exact SSID/BSSID inventory match. Neighbor means outside the inventory. An unrecognized BSSID using an owned SSID is flagged for review, since it may be an unrecorded AP or impersonation. The app does not prove which explanation is correct.' },
    { category: 'Reports', name: 'Finding, severity, confidence & evidence', text: 'A finding is a rule’s interpretation of observed configuration. Severity expresses the rule’s potential impact. Confidence is a rule-assigned indicator of support from the evidence, not a measured probability of compromise. Evidence lists what triggered the rule; remediation explains what to check or change.' },
    { category: 'Reports', name: 'Hidden, restored & deleted snapshots', text: 'Hide removes a snapshot from normal history and the default latest selection while retaining it locally. Show hidden snapshots lets you inspect and restore it. Delete permanently removes the saved snapshot after confirmation; capture files, exports, inventory, and backups remain. This is not secure disk erasure. Running scans must finish or be canceled first.' },
    { category: 'Reports', name: 'JSON, CSV, HTML & SARIF', text: 'JSON preserves structured snapshots. Bluetooth/NFC CSV exports provide spreadsheet rows. Wi-Fi HTML reports are readable and printable; SARIF is a structured format for analysis findings. Exports can contain identifiers and broadcast or tag data.' },
  ];
  let filtered = $derived(terms.filter(t => (category === 'All' || t.category === category) && `${t.name} ${t.text}`.toLowerCase().includes(query.trim().toLowerCase())));
  let relativePower = $derived(new Intl.NumberFormat(undefined, { maximumSignificantDigits: 2 }).format(10 ** ((signal + 60) / 10)));
</script>

<div class="page-heading"><div><p class="eyebrow">A GUIDE TO YOUR OBSERVATIONS</p><h1>About Signal Atlas</h1><p class="subtitle">Understand the numbers, learn the terminology, and follow each scan from discovery to a saved snapshot.</p></div><span class="guide-icon"><BookOpen size={28}/></span></div>
<div class="about-jumps" aria-label="About page sections"><a href="#measurements">Measurements</a><a href="#scan-process">Scan process</a><a href="#glossary">Glossary</a><a href="#data-limits">Data & limits</a></div>
<div class="protocol-cards">
  <article class="panel"><Wifi size={23}/><h2>Wi-Fi</h2><p>Inspect access-point announcements and assess advertised configuration within your authorized inventory.</p></article>
  <article class="panel"><Bluetooth size={23}/><h2>Bluetooth</h2><p>Discover BLE advertisements and inspect names, received signal, and broadcast metadata.</p></article>
  <article class="panel"><Nfc size={23}/><h2>NFC</h2><p>Present a tag to a reader to inspect its identifier and supported, readable NDEF records.</p></article>
</div>

<section id="measurements" class="about-section" aria-labelledby="measurements-title">
  <div class="section-heading"><p class="eyebrow">01 · READ THE NUMBERS</p><h2 id="measurements-title">What the measurements mean</h2><p>Every value describes the selected observation and the sensor that collected it.</p></div>
  <div class="measurement-grid">
    <article class="panel signal-explainer"><div class="card-heading"><h3>RSSI & dBm</h3><Radio size={21}/></div><p>RSSI means Received Signal Strength Indicator. Here, the signal value is reported in dBm: power relative to one milliwatt on a logarithmic scale. A less negative value is stronger: −50 dBm is stronger than −70 dBm.</p>
      <div class="signal-demo"><span class="example-label">ILLUSTRATION · NOT A LIVE READING</span><label for="example-rssi">Example received signal <output for="example-rssi">{signal} dBm</output></label><input id="example-rssi" type="range" min="-90" max="-30" step="5" bind:value={signal} aria-valuetext={`${signal} dBm`}/><div class="signal-scale"><span>−90 · weaker</span><span>−30 · stronger</span></div><p><strong>{relativePower}×</strong> the received power of −60 dBm. A 10 dB increase means ten times the power; it does not mean ten times the speed.</p></div>
      <p class="small-note">Walls, orientation, interference, distance, and adapter calibration affect readings. Signal alone cannot establish throughput, reliability, or distance. NFC does not report RSSI here. <a href="https://www.cisco.com/c/en/us/support/docs/wireless-mobility/wireless-lan-wlan/71113-rrm-new.html" target="_blank" rel="noreferrer">Radio measurement reference ↗</a></p>
    </article>
    <article class="panel"><h3>Band, channel & channel width</h3><dl class="explanations"><div><dt>Band · GHz</dt><dd>The frequency range, such as 2.4, 5, or 6 GHz. GHz means billions of cycles per second; it is not a data rate.</dd></div><div><dt>Frequency · MHz</dt><dd>A radio frequency in millions of cycles per second. For example, 2412 MHz is a frequency in the 2.4 GHz band.</dd></div><div><dt>Channel</dt><dd>A numbered frequency assignment within a band. Read the band together with the channel; channel numbers can repeat across bands.</dd></div><div><dt>Width · MHz</dt><dd>The amount of spectrum a channel uses, such as 20, 40, or 80 MHz. Wider channels occupy more spectrum; they do not guarantee faster service. This parser may report unknown or only a basic width hint.</dd></div><div><dt>Wi-Fi generation</dt><dd>A capability hint from decoded announcements. It is not the speed negotiated by a client; advanced generations and features may be unrecognized.</dd></div></dl></article>
    <article class="panel"><h3>Counts, timestamps & RF charts</h3><p><strong>First / last seen</strong> bound the observations within a snapshot, not the lifetime of a device. Wi-Fi uses capture timestamps; Bluetooth/NFC use the local collection clock. The dashboard displays times in your browser’s local timezone.</p><ul><li><strong>Wi-Fi observations:</strong> accepted beacon/probe-response observations grouped by BSSID.</li><li><strong>Bluetooth observations:</strong> provider advertisement callbacks, not a raw RF packet count.</li><li><strong>NFC observations:</strong> tag presentations; leaving a tag on the reader does not repeatedly increment the count.</li></ul><p>The Wi-Fi curves show received signal by AP, ordered by name. Primary channel occupancy counts APs, not busy-airtime percentage. Counts alone cannot identify interference or a best channel.</p><p class="small-note">The app does not measure internet speed, latency, packet loss, noise floor, or signal-to-noise ratio (SNR). An unknown value or “—” means unavailable, not zero.</p></article>
    <article class="panel"><h3>Configuration score · 0–100</h3><p>Wi-Fi scoring covers only inventory-matched APs. Each starts at 100 and loses points for its findings, with a minimum of zero.</p><div class="penalties"><span><b>−40</b>Critical</span><span><b>−25</b>High</span><span><b>−10</b>Medium</span><span><b>−3</b>Low</span></div><p class="score-example">Example: one high and one medium finding gives an AP <strong>65 / 100</strong>.</p><p>The displayed score is the mean of authorized AP scores, rounded down. With no authorized APs, or incomplete security decoding on an authorized AP, the score is unknown.</p><p>A high score means fewer findings under the current rules. It does not certify password strength, firmware, client behavior, or overall security. Bluetooth and NFC have no security score.</p></article>
  </div>
</section>

<section id="scan-process" class="about-section" aria-labelledby="process-title"><div class="section-heading"><p class="eyebrow">02 · FOLLOW THE PROCESS</p><h2 id="process-title">From discovery to comparison</h2><p>One dashboard, three independent local services, and separate histories.</p></div>
  <ol class="process-steps"><li><span>1</span><div><h3>Choose a source</h3><p>Import a Wi-Fi capture, select a native Bluetooth/NFC scan, or load synthetic demo data. Opening a page or loading a demo does not start a hardware scan.</p></div></li><li><span>2</span><div><h3>Collect a bounded observation</h3><p>Live scans last 1–60 seconds. The selected channel, nearby advertisers, reader compatibility, and device visibility determine what can be observed.</p></div></li><li><span>3</span><div><h3>Decode and save</h3><p>Wi-Fi announcements become AP records and scoped findings. BLE advertisements become device metadata. Supported NFC payloads become tag records. Saved snapshots retain the source and observations.</p></div></li><li><span>4</span><div><h3>Inspect, compare, and export</h3><p>Select an entry for details. Compare snapshots collected under similar conditions. Bluetooth/NFC comparisons require the same demo/live mode. An absent identifier may reflect range, rotation, or a missed presentation.</p></div></li></ol>
  <div class="protocol-cards process-cards"><article class="panel"><h3>Wi-Fi capture</h3><p>Import PCAP/PCAPNG containing wireless management frames. Native capture requires Linux and an existing monitor interface; it listens on that interface’s current channel, without association or injection.</p><p>Add exact SSIDs and BSSIDs under Authorized networks. Reassess to create a new snapshot with the current inventory; earlier findings remain unchanged.</p></article><article class="panel"><h3>BLE discovery</h3><p>Check provider setup and OS Bluetooth permission, choose a duration, and start discovery. The OS may send scan requests, but the service makes no peripheral connections or pairing attempts.</p><p>Only advertising devices visible to the OS can appear. Provider availability does not confirm that the adapter is powered on or permission has been granted.</p></article><article class="panel"><h3>NFC polling</h3><p>Attach a compatible PC/SC contactless reader, select it, and present a tag during the scan. Polling energizes the reader and exchanges fixed select/read commands.</p><p>The supported Type 4 and ACR122 Type 2 profiles read UID/NDEF where accessible. Unsupported tags may still yield an identifier and a read note. No writing or authentication is performed.</p></article></div>
  <div class="panel lifecycle"><h3>Bluetooth / NFC scan states</h3><dl><div><dt>Running</dt><dd>Collection is in progress; results appear when it finishes.</dd></div><div><dt>Completed</dt><dd>A snapshot was saved, even if no devices or tags were observed.</dd></div><div><dt>Failed</dt><dd>A provider, permission, reader, or data error stopped the scan. Read the recorded error.</dd></div><div><dt>Canceled</dt><dd>You stopped the job. Partial observations are not saved.</dd></div></dl><p class="small-note">One scan can run per Bluetooth/NFC database at a time. Failed scans never fall back to demo data. A service interrupted mid-scan is marked failed when its expired job lease is recovered.</p></div>
</section>

<section id="glossary" class="about-section" aria-labelledby="glossary-title"><div class="section-heading"><p class="eyebrow">03 · LOOK UP A TERM</p><h2 id="glossary-title">The network glossary</h2><p>Plain-language definitions for the labels you see in the dashboard and reports.</p></div><div class="glossary-tools"><label class="search"><Search size={17}/><input aria-label="Search glossary" type="search" placeholder="Try SSID, PMF, UUID, or NDEF…" bind:value={query}/></label><div class="glossary-filters" aria-label="Glossary categories">{#each categories as c}<button class:chosen={category === c} aria-pressed={category === c} onclick={() => category = c}>{c}</button>{/each}</div></div><p class="glossary-count" role="status">{filtered.length} {filtered.length === 1 ? 'term' : 'terms'}</p><div class="glossary-list">{#each filtered as term (term.name)}<details class="panel term"><summary><span>{term.name}</span><small>{term.category}</small></summary><p>{term.text}</p></details>{:else}<div class="panel glossary-empty"><h3>No matching terms</h3><p>Try a shorter search or choose All categories.</p><button class="secondary" onclick={() => { query = ''; category = 'All'; }}>Clear filters</button></div>{/each}</div></section>

<section id="data-limits" class="about-section" aria-labelledby="limits-title"><div class="section-heading"><p class="eyebrow">04 · INTERPRET WITH CONTEXT</p><h2 id="limits-title">Your data and the limits of a snapshot</h2></div><div class="panel data-notes"><LockKeyhole size={24}/><div><h3>Stored on this computer</h3><p>Each service saves its history locally in SQLite. There is no cloud sync. Wi-Fi snapshots retain AP identifiers and findings; Bluetooth and NFC also retain broadcast or tag payloads. Databases have restricted file permissions but are not encrypted by the app. Exported files carry their data wherever you share them.</p><h3>Evidence has a scope</h3><p>Unknown metadata stays unknown. A quiet scan, a strong signal, a familiar name, or a high configuration score cannot establish ownership, authenticity, or complete security. Physical BLE scanning and NFC reader/tag interoperability still require validation with your hardware.</p><h3>Discovery and optional active checks</h3><p>The Wi-Fi HTTP dashboard does not run active audits. The separate Linux CLI TCP reachability workflow requires explicit confirmation, an exact connected-network match, and allowed private address/port pairs. It does not test credentials or exploit services.</p><h3>Open source &amp; credits</h3><p>Signal Atlas was created by Michael Kozicki, with implementation and documentation assistance from OpenAI Codex. Original project code is MIT licensed. Go, gopacket, modernc.org/sqlite, Svelte, Lucide/Feather, Bleak, and pyscard retain their upstream licenses; attribution and license notices are included with the source.</p><h3>Read further</h3><div class="reference-links"><a href="https://www.wireshark.org/docs/dfref/w/wlan.html" target="_blank" rel="noreferrer">Wireshark WLAN fields <ArrowUpRight size={14}/></a><a href="https://www.bluetooth.com/bluetooth-le-primer/" target="_blank" rel="noreferrer">Bluetooth SIG LE primer <ArrowUpRight size={14}/></a><a href="https://nfc-forum.org/build/specifications/data-exchange-format-ndef-technical-specification/" target="_blank" rel="noreferrer">NFC Forum NDEF <ArrowUpRight size={14}/></a></div><p class="small-note">Reference links open external websites in a new tab. Tag-provided URLs are displayed as data and are never opened automatically.</p></div></div></section>
<footer class="main-footer"><span><BookOpen size={14}/> Signal Atlas · Guide to measurements and discovery</span><a href="#about" onclick={() => window.scrollTo({ top: 0 })}>Back to top ↑</a></footer>

<style>
.guide-icon{display:grid;place-items:center;width:54px;height:54px;background:#e4f1ec;border-radius:14px;color:#087f75;flex-shrink:0}
.about-jumps{display:flex;flex-wrap:wrap;gap:9px;margin:0 0 26px}
.about-jumps a{padding:9px 15px;background:#fff;border:1px solid var(--line);border-radius:7px;font-size:12px;color:#385d6b}
.about-jumps a:hover{background:#eaf4ef}
.protocol-cards{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:18px}
.protocol-cards article{padding:24px}
.protocol-cards :global(svg){color:var(--teal);margin-bottom:17px}
.protocol-cards p{margin-bottom:0}
.about-section{margin-top:42px;scroll-margin-top:25px}
.section-heading{margin-bottom:21px}
.section-heading h2{font-size:23px;letter-spacing:-.6px}
.section-heading p:last-child{color:#61768a;font-size:14px}
.measurement-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:20px}
.measurement-grid>article{padding:27px}
.measurement-grid h3,.process-cards h3{font-size:17px}
.panel p,.panel li{font-size:14px;color:#526a7b;line-height:1.8}
.panel li{margin-bottom:8px}
.panel ul{padding-left:20px}
.panel p:last-child{margin-bottom:0}
.panel strong{color:#2b4c5b}
.card-heading{display:flex;align-items:start;justify-content:space-between;color:var(--teal)}
.signal-demo{background:#ecf5f1;border:1px solid #d6e8df;padding:22px;border-radius:9px;margin:22px 0}
.example-label{display:block;letter-spacing:1.3px;font-size:9px;color:#587c71;margin-bottom:21px}
.signal-demo label{display:flex;justify-content:space-between;align-items:center;color:#3d6559;font-size:13px;gap:10px}
.signal-demo output{font-size:24px;font-weight:650;white-space:nowrap;color:#087f75}
.signal-demo input{width:100%;margin:23px 0 12px;accent-color:#087f75;cursor:pointer}
.signal-scale{display:flex;justify-content:space-between;color:#678479;font-size:11px}
.signal-demo p{margin:20px 0 0;font-size:13px}
.panel .small-note{font-size:12px;color:#667d8e}
.small-note a{color:#087f75;text-decoration:underline}
.explanations{margin:0}
.explanations>div{padding:13px 0;border-bottom:1px solid #edf1f4;display:block}
.explanations>div:last-child{border:0;padding-bottom:0}
.explanations dt{color:#365b69;font-weight:600;font-size:13px;margin-bottom:6px}
.explanations dd{max-width:none;text-align:left;margin:0;color:#526a7b;font-size:14px;line-height:1.7}
.penalties{display:grid;grid-template-columns:repeat(4,1fr);gap:10px;margin:22px 0}
.penalties span{background:#f4f7fa;padding:14px 6px;text-align:center;border-radius:7px;font-size:11px;color:#677c8e}
.penalties b{display:block;font-size:23px;color:#3b5564;margin-bottom:5px}
.score-example{border-left:3px solid #3f9e86;padding:11px 15px;background:#f0f7f3}
.process-steps{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:24px;padding:0;list-style:none;margin:0 0 27px}
.process-steps li>span{display:grid;place-items:center;width:33px;height:33px;border-radius:50%;background:#dcece5;color:#087f75;font-weight:650;margin-bottom:16px}
.process-steps h3{font-size:15px}
.process-steps p{font-size:13px;color:#5b7385}
.process-cards p+p{margin-top:14px}
.lifecycle{padding:25px;margin-top:20px}
.lifecycle dl{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:22px}
.lifecycle dl>div{display:block;border:0;padding:0}
.lifecycle dt{font-weight:600;color:#375b69;font-size:13px}
.lifecycle dd{max-width:none;text-align:left;margin:9px 0 0;font-size:13px;color:#526a7b;line-height:1.7}
.glossary-tools{display:flex;align-items:center;flex-wrap:wrap;gap:16px}
.glossary-tools .search{flex:1;min-width:240px;background:white}
.glossary-tools input{width:100%;font-size:13px}
.glossary-filters{display:flex;gap:5px;flex-wrap:wrap}
.glossary-filters button{padding:9px 12px;border:1px solid #dae5e8;border-radius:6px;font-size:12px}
.glossary-filters button.chosen{background:#dcefe6;color:#166a56;border-color:#bddcca}
.glossary-count{font-size:12px;color:#667d8e;margin:17px 0 12px}
.glossary-list{display:grid;gap:9px}
.term summary{display:list-item;padding:18px 23px;cursor:pointer;font-size:14px;font-weight:550;color:#345466}
.term summary small{float:right;font-weight:400;color:#6d8291;font-size:11px;margin-left:14px;line-height:1.8}
.term[open] summary{border-bottom:1px solid #edf1f4}
.term p{padding:17px 23px;margin:0;max-width:950px}
.glossary-empty{padding:28px}
.data-notes{padding:28px;display:flex;gap:22px}
.data-notes>:global(svg){flex-shrink:0;color:#087f75}
.data-notes h3:not(:first-child){margin-top:25px}
.reference-links{display:flex;gap:13px 23px;flex-wrap:wrap;margin-bottom:14px}
.reference-links a{display:inline-flex;align-items:center;gap:5px;color:#087f75;font-size:13px;text-decoration:underline}

@media(max-width:1150px){.protocol-cards{grid-template-columns:1fr}
.protocol-cards article{padding:22px}
.measurement-grid{grid-template-columns:1fr}
.process-steps,.lifecycle dl{grid-template-columns:1fr 1fr}
.guide-icon{display:none}
}

@media(max-width:540px){.measurement-grid>article,.lifecycle,.data-notes{padding:20px}
.about-jumps a{padding:8px 11px}
.section-heading h2{font-size:21px}
.panel p,.panel li,.explanations dd{font-size:13px}
.signal-demo{padding:16px}
.signal-demo label{align-items:start;flex-direction:column}
.process-steps{grid-template-columns:1fr;gap:12px}
.process-steps li{display:flex;gap:15px}
.process-steps li>span{flex-shrink:0}
.lifecycle dl{grid-template-columns:1fr}
.term summary{padding:17px;font-size:13px}
.term summary small{float:none;display:block;margin:7px 0 0 15px}
.term p{padding:17px}
.data-notes{display:block}
.data-notes>:global(svg){margin-bottom:17px}
.glossary-tools .search{min-width:0;width:100%;flex-basis:100%}
.glossary-filters button{padding:8px 10px}
}

</style>
