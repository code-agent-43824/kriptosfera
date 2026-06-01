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
- app config validation now checks that `startUrl` belongs to `allowedOrigins` when origins are configured, requires `diagnosticsUrl` to be HTTPS, and rejects an unsafe `profileName` (must be a single path segment with no `..`, path separators, or `:`) so the per-app profile directory cannot escape the app root;
- the remote downloader now caps how many bytes it will write (pinned expected size, or a 1 GiB absolute limit) and aborts early instead of streaming a runaway response to disk before the SHA-256 check;
- the repository ships committed zero-byte placeholder `payload.zip`/`cryptopro-plugin.zip` (to satisfy `go:embed`) so the launcher compiles and `go test ./...` runs on a clean checkout; an empty embed is treated as "bundle not embedded", and Windows build scripts overwrite the placeholders with the real artifacts;
- the launcher starts Chromium as a standalone app window (`--app=<startUrl>` when `windowMode` is `app`); diagnostics is off in the demo config.

Current implementation boundaries:
- `allowedOrigins` is a startup/config guard, not a full Chromium navigation sandbox;
- full post-start navigation/domain policy is future product hardening, not an MVP blocker;
- the clean-machine Mini CSP blocker was root-caused to a broken plug-in build (`2.0.15700`); the working combination is plug-in `2.0.15000` + a Manifest V2 extension (`1.2.13`) + a Manifest V2-capable Chromium (Chrome 138). Wiring this into the launcher (re-pin plug-in/extension) and the Rutoken signing check are the remaining MVP layers — see `docs/cryptopro-csp-lite-plan.md`.

## Explicit non-goals for first MVP

- GOST TLS
- Linux support
- full installer UX
- auto-update
- broad token compatibility
- relying only on preinstalled system CryptoPro CSP
