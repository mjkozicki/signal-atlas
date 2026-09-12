<script lang="ts">
  import { onMount } from 'svelte';
  import SnapshotActions from './SnapshotActions.svelte';
  let showHidden = $state(false);
  let historyVersion = 0;
  import { Bluetooth, Nfc, Play, Square, RefreshCw, Download, Upload, Search, ChevronRight, AlertTriangle, Check, Radio, History, Server, X } from '@lucide/svelte';
  type Device = { id: string; name: string; first_seen: string; last_seen: string; observations: number; rssi: number | null; details: Record<string, unknown> };
  type Scan = { hidden?: boolean; id: string; protocol: string; mode: string; source: string; state: string; created_at: string; ended_at: string | null; duration_seconds: number; devices: Device[]; warnings: string[]; error?: string };
  type Status = { protocol: string; provider: string; available: boolean; message: string; readers: string[] };
  type Change = { id: string; name: string; kind: string; detail: string };
  let { protocol }: { protocol: 'bluetooth' | 'nfc' } = $props();
  let scans = $state<Scan[]>([]), scan = $state<Scan | null>(null), selected = $state<Device | null>(null), status = $state<Status | null>(null);
  let loading = $state(true), busy = $state(false), error = $state(''), notice = $state(''), offline = $state(false), query = $state('');
  let duration = $state(10), reader = $state(''), adapter = $state(''), tab = $state('Devices'), from = $state(''), to = $state(''), changes = $state<Change[] | null>(null);
  let fileInput: HTMLInputElement;
  let title = $derived(protocol === 'bluetooth' ? 'Bluetooth' : 'NFC');
  let running = $derived(scans.find(s => s.state === 'running'));
  let devices = $derived((scan?.devices ?? []).filter(d => `${d.name} ${d.id}`.toLowerCase().includes(query.toLowerCase())));
  let completed = $derived(scans.filter(s => s.state === 'completed'));
  const when = (date: string) => new Date(date).toLocaleString([], { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit', second: '2-digit' });
  async function api(path: string, options?: RequestInit) {
    const response = await fetch(`/api/${protocol}${path}`, { ...options, headers: { 'X-Wifi-Scanner': 'local', ...options?.headers } });
    let data; try { data = await response.json(); } catch { throw new Error(`Service returned HTTP ${response.status}.`); }
    if (!response.ok) throw new Error(data.error || `HTTP ${response.status}`);
    return data;
  }
  function choose(id?: string) { scan = scans.find(s => s.id === id) ?? scans.find(s => !s.hidden) ?? scans[0] ?? null; selected = scan?.devices.find(d => d.id === selected?.id) ?? scan?.devices[0] ?? null; }
  async function history(preferred?: string) {
    const version = ++historyVersion;
    const updated = await api(`/scans?include_hidden=${showHidden}`);
    if (version !== historyVersion) return;
    scans = updated; offline = false;
    choose(preferred ?? scan?.id ?? scans[0]?.id);
    if (!completed.some(s => s.id === from)) from = completed[1]?.id ?? '';
    if (!completed.some(s => s.id === to)) to = completed[0]?.id ?? '';
  }
  async function refresh() {
    loading = true; error = '';
    try { await history(); status = await api('/status'); if (protocol === 'nfc' && status?.readers.length === 1) reader = status.readers[0]; }
    catch(e) { error = String(e instanceof Error ? e.message : e); offline = true; }
    finally { loading = false; }
  }
  onMount(() => {
    let alive = true, polling = false;
    refresh();
    const timer = setInterval(async () => { if (running && !polling && !busy && !loading) { polling = true; try { const id = scan?.id, version = historyVersion; const updated: Scan[] = await api(`/scans?include_hidden=${showHidden}`); if (alive && version === historyVersion) { scans = updated; choose(id ?? updated[0]?.id); } } catch(e) { if (alive) error = String(e); } finally { polling = false; } } }, 1000);
    return () => { alive = false; clearInterval(timer); };
  });
  async function work(action: () => Promise<void>) { busy = true; error = ''; notice = ''; try { await action(); } catch(e) { error = e instanceof Error ? e.message : String(e); } finally { busy = false; } }
  async function start(mode: 'demo' | 'live') {
    await work(async () => { const created: Scan = await api('/scans', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ mode, duration_seconds: duration, ...(protocol === 'nfc' ? { reader } : { adapter }) }) }); await history(created.id); tab = 'Devices'; notice = mode === 'demo' ? 'Created a synthetic scan. No hardware was accessed.' : 'Scan started. Results appear when discovery finishes.'; });
  }
  async function cancel() { if (running) await work(async () => { await api(`/scans/${running!.id}/cancel`, { method: 'POST' }); notice = 'Cancellation requested.'; }); }
  async function importFile(event: Event) {
    const input = event.currentTarget as HTMLInputElement, file = input.files?.[0]; if (!file) return;
    await work(async () => { if (file.size > 4*1024*1024) throw new Error('Snapshots must be 4 MB or smaller.'); const saved = await api('/import', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: await file.text() }); await history(saved.id); notice = `Imported ${file.name}.`; }); input.value = '';
  }
  async function manageSnapshot(s: Scan, action: 'hide' | 'restore' | 'delete') {
    ++historyVersion;
    await work(async () => {
      await api(`/scans/${s.id}`, action === 'delete' ? { method: 'DELETE' } : { method: 'PATCH', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ hidden: action === 'hide' }) });
      changes = null; from = ''; to = ''; await history();
      notice = action === 'delete' ? 'Snapshot deleted.' : action === 'hide' ? 'Snapshot hidden. Enable Show hidden snapshots to restore it.' : 'Snapshot restored.';
    });
  }
  async function compare() { await work(async () => { changes = await api(`/diff?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}`); }); }
  function entries(d: Device): [string, unknown][] { return Object.entries(d.details ?? {}).filter(([key]) => key !== 'ndef_records'); }
  function records(d: Device): { type: string; text?: string; uri?: string; language?: string; payload_hex?: string }[] {
    const value = d.details?.ndef_records;
    if (!Array.isArray(value)) return [];
    return value.filter(r => r && typeof r === 'object' && typeof r.type === 'string').map(r => ({
      type: r.type,
      text: typeof r.text === 'string' ? r.text : undefined,
      uri: typeof r.uri === 'string' ? r.uri : undefined,
      language: typeof r.language === 'string' ? r.language : undefined,
      payload_hex: typeof r.payload_hex === 'string' ? r.payload_hex : undefined,
    }));
  }
  function display(value: unknown) { return typeof value === 'string' ? value : JSON.stringify(value, null, 2); }
</script>

<div class="page-heading"><div><p class="eyebrow">INDEPENDENT DISCOVERY SERVICE</p><h1>{title} observations</h1><p class="subtitle">{protocol === 'bluetooth' ? 'Discover advertising BLE devices and inspect what they broadcast.' : 'Present a tag to your reader to inspect its identifier and readable NDEF records.'}</p></div><div class="heading-actions"><button class="secondary" onclick={refresh} disabled={loading || busy}><RefreshCw size={15}/> Refresh</button><button class="secondary" onclick={() => fileInput.click()} disabled={offline || busy}><Upload size={15}/> Import JSON</button><input type="file" accept=".json,application/json" bind:this={fileInput} onchange={importFile} hidden/></div></div>
{#if error}<div class="message error" role="alert"><AlertTriangle size={18}/><span>{error}</span><button aria-label="Dismiss error" onclick={() => error = ''}><X size={16}/></button></div>{/if}
{#if notice}<div class="message success" role="status"><Check size={18}/><span>{notice}</span></div>{/if}
<section class="panel service-status"><span class="service-icon">{#if protocol === 'bluetooth'}<Bluetooth size={25}/>{:else}<Nfc size={25}/>{/if}</span><div><strong>{offline ? `${title} service is offline` : status?.provider ?? 'Checking provider…'}</strong><p>{offline ? `Start bin/${protocol}-scan serve, or run make serve-all to launch all services.` : status?.message ?? 'Checking local dependencies and reader availability.'}</p></div><span class="badge" class:good={status?.available && !offline} class:neutral={!status?.available || offline}>{offline ? 'Offline' : loading ? 'Checking' : status?.available ? 'Provider available' : 'Setup needed'}</span></section>
<div class="sensor-controls panel"><label>Duration<select bind:value={duration}><option value={5}>5 seconds</option><option value={10}>10 seconds</option><option value={30}>30 seconds</option><option value={60}>60 seconds</option></select></label>{#if protocol === 'nfc'}<label class="reader-control">NFC reader<select bind:value={reader}><option value="">Automatic (one reader only)</option>{#each status?.readers ?? [] as r}<option value={r}>{r}</option>{/each}</select></label>{:else}<label>Linux adapter<input bind:value={adapter} placeholder="System default" aria-label="Linux Bluetooth adapter"/></label>{/if}<div class="sensor-buttons"><button class="secondary" disabled={offline || busy || !!running} onclick={() => start('demo')}>Load demo scan</button>{#if running}<button class="secondary stop" disabled={busy} onclick={cancel}><Square size={15}/> Cancel scan</button>{:else}<button class="primary" disabled={offline || busy || !status?.available} onclick={() => start('live')}><Play size={15}/>{protocol === 'bluetooth' ? 'Start BLE discovery' : 'Poll NFC reader'}</button>{/if}</div><p class="mode-note">{protocol === 'bluetooth' ? 'OS-managed discovery may transmit scan requests. No pairing or peripheral connections.' : 'Polling energizes the reader and sends fixed read/select commands. No tag writing or authentication.'}</p></div>
<div class="sensor-metrics"><div class="panel"><span>{protocol === 'bluetooth' ? 'Observed devices' : 'Observed tags'}</span><strong>{scan?.devices.length ?? '—'}</strong></div><div class="panel"><span>Selected scan</span><strong class="state-word">{scan?.state ?? 'No scan yet'}</strong></div><div class="panel"><span>Data source</span><strong class="state-word">{scan?.mode === 'demo' ? 'Synthetic demo' : scan ? scan.source : '—'}</strong></div><div class="panel"><span>Last scan</span><strong class="date-word">{scan ? when(scan.created_at) : '—'}</strong></div></div>
<div class="snapshot-toolbar"><label><input type="checkbox" bind:checked={showHidden} disabled={busy || offline} onchange={() => work(async () => { changes = null; from = ''; to = ''; await history(); })}/> Show hidden snapshots</label>{#if scan}{#key scan.id}<SnapshotActions id={scan.id} label={`${when(scan.created_at)} · ${scan.mode} · ${title}`} hidden={scan.hidden} disabled={busy || offline || scan.state === 'running'} onaction={action => manageSnapshot(scan!, action)}/>{/key}{/if}</div>
<div class="sensor-nav"><div class="segmented">{#each ['Devices','History'] as t}<button class:chosen={tab === t} onclick={() => tab = t}>{t === 'Devices' && protocol === 'nfc' ? 'Tags' : t}</button>{/each}</div>{#if scans.length}<label class="scan-picker">Snapshot<select value={scan?.id} onchange={e => choose(e.currentTarget.value)}>{#each scans as s}<option value={s.id}>{when(s.created_at)} · {s.mode} · {s.state}{s.hidden ? ' · Hidden' : ''}</option>{/each}</select></label>{/if}{#if scan?.state === 'completed'}<details class="export-menu"><summary><Download size={15}/> Export</summary><div>{#each ['json','csv'] as format}<a href={`/api/${protocol}/export/${scan.id}?format=${format}`}>{format.toUpperCase()} snapshot</a>{/each}</div></details>{/if}</div>
{#if scan?.error}<div class="message error" role="alert"><AlertTriangle size={18}/><span>{scan.error}</span></div>{/if}
{#if tab === 'Devices'}
  {#if !scan}<section class="panel empty-state"><Radio size={36}/><h2>{protocol === 'bluetooth' ? 'No visible Bluetooth snapshots' : 'No visible NFC snapshots'}</h2><p>{protocol === 'bluetooth' ? 'Start a bounded scan, import a snapshot, or explore the demo without using your adapter. Enable Show hidden snapshots to restore hidden scans.' : 'Connect a compatible PC/SC NFC reader, import a snapshot, or explore a synthetic tag. Enable Show hidden snapshots to restore hidden scans.'}</p></section>
  {:else if scan.state === 'running'}<section class="panel empty-state" role="status"><Radio size={36}/><h2>{protocol === 'bluetooth' ? 'Listening for advertisements…' : 'Waiting for tags…'}</h2><p>Scan duration: {scan.duration_seconds} seconds. Results are saved when the scan finishes.</p></section>
  {:else}<div class="sensor-grid"><section class="panel"><div class="panel-heading"><h2>{protocol === 'bluetooth' ? 'Devices' : 'Tags'} <span class="count">{devices.length}</span></h2><div class="search"><Search size={15}/><input placeholder="Search name or identifier…" aria-label={`Search ${title} observations`} bind:value={query}/></div></div><div class="table-wrap"><table><thead><tr><th>NAME / IDENTIFIER</th><th>{protocol === 'bluetooth' ? 'SIGNAL' : 'TYPE'}</th><th>OBSERVATIONS</th></tr></thead><tbody>{#each devices as device}<tr class:selected={selected?.id === device.id}><td><button class="ap-select" onclick={() => selected = device}><span class="ap-icon owned">{#if protocol === 'bluetooth'}<Bluetooth size={17}/>{:else}<Nfc size={17}/>{/if}</span><span><strong>{device.name}</strong><small>{device.id}</small></span></button></td><td>{protocol === 'bluetooth' ? `${device.rssi ?? 'Unknown'}${device.rssi === null ? '' : ' dBm'}` : device.details.technology ?? 'Unknown'}</td><td>{device.observations}</td></tr>{:else}<tr><td colspan="3" class="no-results">{query ? 'No observations match your search.' : scan.state === 'completed' ? 'No devices or readable tags observed during this scan.' : 'This scan did not produce a completed snapshot.'}</td></tr>{/each}</tbody></table></div></section><section class="panel sensor-detail">{#if selected}<p class="eyebrow">OBSERVATION DETAILS</p><h2>{selected.name}</h2><code class="device-id">{selected.id}</code><dl><div><dt>First seen</dt><dd>{when(selected.first_seen)}</dd></div><div><dt>Last seen</dt><dd>{when(selected.last_seen)}</dd></div></dl>{#each entries(selected) as [key,value]}<div class="metadata"><h3>{key.replaceAll('_',' ')}</h3><pre>{display(value)}</pre></div>{/each}{#if records(selected).length}<h3 class="ndef-heading">NDEF records</h3>{#each records(selected) as record}<article class="ndef-record"><span class="badge neutral">{record.type === 'T' ? 'Text' : record.type === 'U' ? 'URI' : record.type}</span>{#if record.language}<small>{record.language}</small>{/if}<p>{record.text ?? record.uri ?? record.payload_hex}</p></article>{/each}<p class="mode-note">Tag content is displayed as data. URLs are not opened automatically.</p>{/if}{:else}<p class="muted">Select an observation to inspect its metadata.</p>{/if}</section></div>{/if}
{:else}
  <section class="panel"><div class="panel-heading"><h2>Compare completed snapshots</h2><History size={19}/></div>{#if completed.length > 1}<div class="compare-controls"><label>Earlier<select bind:value={from}>{#each completed as s}<option value={s.id}>{when(s.created_at)} · {s.mode} · {s.id.slice(-6)}{s.hidden ? ' · Hidden' : ''}</option>{/each}</select></label><ChevronRight size={18}/><label>Later<select bind:value={to}>{#each completed as s}<option value={s.id}>{when(s.created_at)} · {s.mode} · {s.id.slice(-6)}{s.hidden ? ' · Hidden' : ''}</option>{/each}</select></label><button class="primary" disabled={busy || !from || !to || from === to} onclick={compare}>Compare</button></div>{:else}<p class="history-hint">Complete or import a second scan to compare observations.</p>{/if}{#if changes !== null}<div class="diff-results"><h3>{changes.length} changes</h3>{#each changes as change}<div class="change-row"><span class="badge neutral">{change.kind}</span><div><strong>{change.name}</strong><p>{change.detail}</p><code>{change.id}</code></div></div>{:else}<p>No metadata or signal changes for the observed identifiers.</p>{/each}</div>{/if}</section>
  <section class="panel history-panel"><div class="panel-heading"><h2>Scan history</h2><span class="muted">Latest 100 · {title} database</span></div>{#each scans as s (s.id)}<div class="managed-history-row"><button class="history-row" onclick={() => { choose(s.id); tab = 'Devices'; }}><History size={18}/><span><strong>{when(s.created_at)}</strong><small>{s.mode} · {s.id}</small></span><span class="badge neutral">{s.state}</span><span>{s.devices.length} observations</span><ChevronRight size={16}/></button><SnapshotActions id={s.id} label={`${when(s.created_at)} · ${s.mode} · ${title}`} hidden={s.hidden} disabled={busy || offline || s.state === 'running'} onaction={action => manageSnapshot(s, action)}/></div>{/each}</section>
{/if}
{#if scan?.warnings.length}<section class="panel sensor-notes"><h3>Observation notes</h3><ul>{#each scan.warnings as warning}<li>{warning}</li>{/each}</ul></section>{/if}
<footer class="main-footer"><span><Server size={14}/> Independent {title} service · Local SQLite history</span><span>Discovery observations do not establish security or ownership.</span></footer>

<style>
.service-status{display:flex;align-items:center;gap:16px;padding:20px 23px;margin-bottom:18px}.service-status>div{flex:1}.service-status strong{font-size:14px}.service-status p{font-size:13px;color:#738698;margin:7px 0 0}.service-icon{display:grid;place-items:center;width:44px;height:44px;background:#e7f3ef;color:#178674;border-radius:12px;flex-shrink:0}.sensor-controls{padding:22px;display:flex;flex-wrap:wrap;align-items:end;gap:17px}.sensor-controls label{font-size:12px;color:#718395}.sensor-controls select,.sensor-controls input{display:block;margin-top:9px;padding:10px;border:1px solid #dce4eb;background:#fff;border-radius:6px;max-width:100%;color:#3b5667}.reader-control{flex:1;min-width:200px}.reader-control select{width:100%}.sensor-buttons{margin-left:auto;display:flex;gap:10px}.mode-note{font-size:12px;color:#7c8e9c;width:100%;margin:0;line-height:1.7}.sensor-metrics{display:grid;grid-template-columns:repeat(4,1fr);gap:17px;margin:21px 0}.sensor-metrics>div{padding:20px}.sensor-metrics span{display:block;color:#7a8b99;font-size:12px}.sensor-metrics strong{display:block;font-size:29px;margin-top:14px;color:#294d59}.sensor-metrics .state-word{font-size:18px;text-transform:capitalize}.sensor-metrics .date-word{font-size:15px;line-height:1.7}.sensor-nav{display:flex;gap:15px;align-items:center;margin:22px 0}.sensor-nav .scan-picker{flex:1;justify-content:flex-end}.sensor-grid{display:grid;grid-template-columns:minmax(0,1.25fr) minmax(300px,1fr);gap:20px;align-items:start}.sensor-grid td{font-size:12px;white-space:normal}.sensor-grid .ap-select small{max-width:230px;overflow-wrap:anywhere;white-space:normal;font-size:11px}.sensor-grid .ap-select strong{font-size:14px}.sensor-detail{padding:24px}.device-id{color:#8494a2;display:block;overflow-wrap:anywhere}.metadata{margin-top:19px}.metadata h3{text-transform:capitalize;font-size:12px;color:#69818f}.metadata pre{background:#f3f6f8;padding:12px;border-radius:6px;white-space:pre-wrap;overflow-wrap:anywhere;font-size:12px;color:#4e687a;max-height:210px;overflow:auto}.ndef-heading{margin-top:24px}.ndef-record{background:#edf7f1;border:1px solid #d5e8dc;padding:14px;border-radius:7px;margin-bottom:12px}.ndef-record p{font-size:14px;color:#385f55;overflow-wrap:anywhere;margin:10px 0 0}.ndef-record small{margin-left:8px;color:#799184}.sensor-notes{padding:21px;margin-top:22px}.sensor-notes h3{font-size:13px}.sensor-notes li{font-size:12px;line-height:1.8;color:#748999}.history-hint{padding:0 22px 20px;color:#7d91a1;font-size:13px}.stop{color:#9f4b58}.sensor-controls .primary,.sensor-controls .secondary{font-size:13px}.sensor-detail h2{overflow-wrap:anywhere}
@media(max-width:1250px){.sensor-grid{grid-template-columns:minmax(0,1fr)}.sensor-metrics{grid-template-columns:1fr 1fr}.sensor-buttons{margin-left:0}.sensor-nav{flex-wrap:wrap}.sensor-nav .scan-picker{justify-content:flex-start}.service-status{align-items:start}.service-status>.badge{margin-left:auto}}
@media(max-width:600px){.service-status{flex-wrap:wrap;padding:17px}.service-status>.badge{margin-left:60px}.sensor-metrics{gap:10px}.sensor-metrics>div{padding:16px}.sensor-controls{padding:18px}.sensor-controls label{width:100%}.sensor-controls select,.sensor-controls input{width:100%}.sensor-buttons{flex-wrap:wrap}.sensor-nav .scan-picker{min-width:100%;flex-wrap:wrap}.sensor-nav select{max-width:100%}.sensor-grid .panel-heading{flex-wrap:wrap}.sensor-detail{padding:20px}.sensor-metrics .date-word{font-size:13px}.sensor-grid td:first-child{max-width:220px}}
</style>
