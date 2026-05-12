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

## Explicit non-goals for first MVP

- GOST TLS
- Linux support
- full installer UX
- auto-update
- broad token compatibility
- system CryptoPro fallback
