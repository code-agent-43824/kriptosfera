# Project summary from source documents

## Product idea

Kriptosfera is a technology/service for producing branded desktop business apps for Russian companies whose clients work with qualified electronic signatures.

## Core promise

One downloadable executable, no manual browser/plugin/CSP setup, isolated runtime, customer-branded shell.

## Primary MVP target

Windows executable with:
- Go launcher
- embedded or remote payload
- bundled Chromium runtime
- CryptoPro extension
- native messaging host
- CryptoPro crypto libraries / CSP Lite hypothesis
- Rutoken-based signing test against CryptoPro demo page

Current delivery direction after early MVP validation:
- keep embedded mode for offline/demo/support cases;
- make thin launcher + remote immutable payload the main product path.

Current implementation status:
- remote runtime core is already in code;
- CI now builds both embedded and remote launcher variants;
- payload artifacts are produced in immutable version/sha-based layout for delivery.
- remote first-run now has minimal visible progress UX on Windows.
- canonical unpacked CryptoPro extension `1.3.17` is now committed into `payload-template/extensions/cryptopro-cades/`;
- launcher now derives stable extension wiring from payload layout, computes the expected extension id from `manifest.json`, and writes diagnostics status into `diagnostics/extension-status.js`;
- diagnostics page now probes `chrome-extension://.../nmcades_plugin_api.js`, so extension delivery and runtime script availability are observable before native messaging/CSP work starts.
- app config validation now checks that `startUrl` belongs to `allowedOrigins` when origins are configured;
- `diagnosticsEnabled` now gates launcher-side diagnostic file generation instead of being a purely decorative field.

Current implementation boundaries:
- `allowedOrigins` is a startup/config guard, not a full Chromium navigation sandbox yet;
- native messaging, CryptoPro plugin/CSP, Rutoken discovery, certificate selection, and signing remain the next MVP layers.

## Explicit non-goals for first MVP

- GOST TLS
- Linux support
- full installer UX
- auto-update
- broad token compatibility
- system CryptoPro fallback
