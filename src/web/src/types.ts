export interface Security { protocol: string; akm: string[]; ciphers: string[]; pmf: string }
export interface AP { ssid: string; bssid: string; vendor: string; band: string; channel: number; frequency: number; width?: number; rssi: number | null; generation: string; hidden: boolean; beacon_interval: number; security: Security; first_seen: string; last_seen: string; observations: number; authorized: boolean }
export interface Finding { id: string; rule_id: string; severity: string; title: string; bssid: string; ssid: string; evidence: string[]; why_it_matters: string; remediation: string; confidence: number }
export interface Scan { hidden?: boolean; id: string; created_at: string; source: string; rule_version: string; access_points: AP[]; findings: Finding[]; score: number | null; warnings: string[] }
export interface Change { bssid: string; ssid: string; kind: string; detail: string }
export interface Difference { from: string; to: string; changes: Change[] }
