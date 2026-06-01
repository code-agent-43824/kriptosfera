# Documentation index

This directory holds the concept material, the engineering plans, and the
architecture overview for Kriptosfera. Start with the root
[`README.md`](../README.md) for the product summary and current status.

## Architecture

- [`architecture.md`](architecture.md) — launcher flow, embedded vs. remote
  payload, and the runtime AppData layout (with a diagram).

## Engineering plans

- [`remote-payload-implementation-plan.md`](remote-payload-implementation-plan.md)
  — thin launcher + remote immutable payload mode.
- [`cryptopro-extension-v0.4-blueprint.md`](cryptopro-extension-v0.4-blueprint.md)
  — CryptoPro CAdES browser extension delivery.
- [`native-messaging-cryptopro-plugin-plan.md`](native-messaging-cryptopro-plugin-plan.md)
  — native messaging host + embedded CryptoPro binaries.
- [`cryptopro-csp-lite-plan.md`](cryptopro-csp-lite-plan.md) — the current focus:
  activating bundled CSP Lite / Mini CSP on a clean machine (the main MVP
  blocker).
- [`handoff-minicsp-test-pages.md`](handoff-minicsp-test-pages.md) — handoff for
  the server-managing agent: deploy the `internal-csp-early` test page and a
  summary of repo changes made out-of-band.
- [`handoff-windows-minicsp-snapshots.md`](handoff-windows-minicsp-snapshots.md)
  — handoff for a Windows session: capture registry + Program Files snapshots
  across three CryptoPro install states (clean / no-flags / `ADDMINICSP=1`).
- [`handoff-windows-reverse-enum.md`](handoff-windows-reverse-enum.md) —
  handoff for the Windows session: reverse enumeration test (CryptoPro config
  view vs Microsoft registry view) settling why `About.CSPName(80)` is empty.
- [`handoff-windows-config-trace.md`](handoff-windows-config-trace.md) —
  handoff (Windows): diagnose where the plugin reads `config.ini` from, with a
  careful legitimate-troubleshooting framing.
- [`minicsp-snapshots/`](minicsp-snapshots/README.md) — the captured snapshots
  and per-phase analysis produced by that handoff.

## Inventories and references

- [`cryptopro-plugin-inventory.md`](cryptopro-plugin-inventory.md) — files inside
  the CryptoPro Browser Plugin bundle.
- [`cryptopro-static-bundles.md`](cryptopro-static-bundles.md) — how static
  artifacts (payload, plugin) are hosted and pinned.
- [`mvp-risks.md`](mvp-risks.md) — open technical risks for the MVP.

## Summary

- [`project-summary.md`](project-summary.md) — condensed product idea, MVP
  target, and current implementation status distilled from the source documents.

## Source concept documents

The `*.txt` files are the original imported concept documents (their hashed
suffixes come from the import) and are kept verbatim for reference:

- `Kriptosfera_Concept_Description---*.txt` — product concept and vision.
- `Kriptosfera_Concept_Tech---*.txt` — technical concept.
- `Kriptosfera_Remote_Payload_Architecture_Note---*.txt` — remote payload
  architecture note.
