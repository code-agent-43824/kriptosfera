# Documentation index

This directory holds the concept material, the engineering plans, and the
architecture overview for Kriptosfera. Start with the root
[`README.md`](../README.md) for the product summary and current status.

## Architecture

- [`architecture.md`](architecture.md) — launcher flow, embedded vs. remote
  payload, and the runtime AppData layout (with a diagram).

## Worklog

- [`worklog.md`](worklog.md) — per-chunk handoff log (Planned / Done / Next) for
  the multi-agent workflow; read this first to see what was last in progress.

## Engineering plans

- [`remote-payload-implementation-plan.md`](remote-payload-implementation-plan.md)
  — thin launcher + remote immutable payload mode.
- [`cryptopro-extension-v0.4-blueprint.md`](cryptopro-extension-v0.4-blueprint.md)
  — CryptoPro CAdES browser extension delivery.
- [`native-messaging-cryptopro-plugin-plan.md`](native-messaging-cryptopro-plugin-plan.md)
  — native messaging host + embedded CryptoPro binaries.
- [`cryptopro-csp-lite-plan.md`](cryptopro-csp-lite-plan.md) — Mini CSP / CSP Lite
  activation: root cause (a broken plugin build), the working combination, and
  the remaining integration + future-migration goals.
- [`cryptopro-portable-plugin-findings.md`](cryptopro-portable-plugin-findings.md)
  — consolidated record of the clean-machine (portable, no-MSI) blocker: the
  CryptoPro `GetModuleFileName(0x10000000)` bug, everything we tried, the host
  probe findings, and the vendor report. Investigation paused, awaiting a vendor fix.

- [`cryptopro-rutoken-fkc-pkcs11.md`](cryptopro-rutoken-fkc-pkcs11.md) — Rutoken ЭЦП
  FKC + PKCS#11 (active) carriers ported from the Linux provider, Windows-adapted,
  with the required-DLL caveat.

## Inventories and references

- [`cryptopro-plugin-inventory.md`](cryptopro-plugin-inventory.md) — files inside
  the CryptoPro Browser Plugin bundle.
- [`cryptopro-static-bundles.md`](cryptopro-static-bundles.md) — how static
  artifacts (payload, plugin) are hosted and pinned.
- [`payload-slimming-plan.md`](payload-slimming-plan.md) — current size inventory
  and conservative plan for trimming the Chromium payload.
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
