# Reporting a vulnerability

Report security issues privately to **github@michaelkozicki.com** with “Signal Atlas
security” in the subject. Include the affected version/commit, reproduction steps,
impact, and synthetic sample data where possible. Do not post real captures,
credentials, or personal identifiers in a public issue.

This is an early-stage local tool. There is no promised response SLA or long-term
support schedule. Fixes target the current main branch.

The HTTP services bind to loopback and enforce local-origin mutation checks. Docker
mode explicitly allows a wildcard dashboard bind inside the container; Compose
publishes that port only on host loopback. Keep that port mapping local. The services
are not designed for public internet exposure, untrusted multi-user hosting, or
protection from hostile software running as the same OS account. Hardware scans
must be explicitly started; tests and demo mode use synthetic data.
