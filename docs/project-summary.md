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
- launcher now derives stable extension wiring from payload layout and computes the expected extension id from `manifest.json`;
- hosted diagnostics now uses CryptoPro's official `cadesplugin_api.js` path and targeted `CAdESCOM.About` calls, so extension delivery, Browser Plugin version, and CSP/provider state are observable before CSP Lite activation starts.
- CryptoPro Browser Plugin `2.0.15700` is pinned in `build/cryptopro-plugin-lock.json`, downloaded from project static storage, verified by SHA-256/size, and embedded into both launcher variants during Windows builds.
- launcher now extracts the embedded CryptoPro Browser Plugin bundle into the versioned AppData app directory and validates `nmcades.exe`, `nmcades.json`, and `npcades.dll` before reuse.
- launcher now generates the Chrome native messaging manifest for `ru.cryptopro.nmcades` and registers it under HKCU for the current user before Chromium starts.
- manual Windows validation showed that, when a normal system CryptoPro CSP is installed, Kriptosfera behaves like a configured Chrome: extension, Browser Plugin, plugin version, system CSP, standard access confirmation dialog, and certificate enumeration all work.
- app config validation now checks that `startUrl` belongs to `allowedOrigins` when origins are configured;
- `diagnosticsUrl` controls whether Chromium opens a public HTTPS diagnostics page alongside the target page.

Current implementation boundaries:
- `allowedOrigins` is a startup/config guard, not a full Chromium navigation sandbox;
- full post-start navigation/domain policy is future product hardening, not an MVP blocker;
- diagnostics remains enabled for the MVP because it is needed to verify launcher/runtime/extension wiring;
- while diagnostics is enabled and `diagnosticsUrl` is configured, launcher opens both the configured start URL and the public HTTPS diagnostics page in Chromium window mode so test machines can capture the CSP matrix without manual file navigation;
- bundled CSP Lite / Mini CSP activation on clean machines, Rutoken discovery, certificate selection, and signing remain the next MVP layers.
- on clean machines without system CryptoPro CSP, the current embedded Browser Plugin layer loads but reports plugin version `0.0.0000` and does not load CSP/provider state; treat this as missing provider activation, not an extension/native messaging failure.

## Explicit non-goals for first MVP

- GOST TLS
- Linux support
- full installer UX
- auto-update
- broad token compatibility
- relying only on preinstalled system CryptoPro CSP
