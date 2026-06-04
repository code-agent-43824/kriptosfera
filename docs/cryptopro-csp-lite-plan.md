# CryptoPro CSP Lite / Mini CSP — status and plan

## Goal

Kriptosfera must work on a clean Windows machine using Kriptosfera-managed
CryptoPro components (bundled CAdES Browser Plug-in with Mini CSP / CSP Lite),
**without** a system-installed CryptoPro CSP. The reference scenario is the
CryptoPro CAdES-BES demo page: extension detected, plugin detected, crypto
provider loaded, certificates enumerated, and a test signature with a Rutoken.

## Outcome (resolved)

The long "provider not loaded / `0x80090017`" blocker was **not** a packaging,
registry, config-path, licensing, or architecture problem on our side. Per
CryptoPro, the plugin build we had pinned (**2.0.15700**) was **broken**.

- **Root cause:** the broken `2.0.15700` CAdES plug-in build does not activate
  its internal Mini CSP. Rolling back to plug-in **2.0.15000** makes the bundled
  Mini CSP load: the demo/diagnostics page reports "Криптопровайдер загружен",
  provider `Crypto-Pro GOST R 34.10-2012 Cryptographic Service Provider`,
  CSP version `5.0.13001`, with **no system CSP installed**. Our approach was
  correct all along.
- **Extension constraint:** the working `2.0.15000` plug-in pairs with the
  **Manifest V2** CryptoPro extension (`1.2.13`). The Manifest V3 extension
  (`1.3.17`) does not work with this plug-in build.
- **Chromium constraint:** Manifest V2 only runs up to **Chrome 138** (via the
  `ExtensionManifestV2Availability` enterprise policy); Chrome 139+ removed it.
  The bundled Chromium is therefore pinned to the last MV2-capable Chrome for
  Testing milestone (`138.0.7204.183`, see `build/chromium-runtime.json`).

## Working combination

| Component | Use |
| --- | --- |
| CAdES Browser Plug-in | **2.0.15000** (not the broken 2.0.15700) |
| CryptoPro extension | **Manifest V2 `1.2.13`** (not MV3 1.3.17) |
| Bundled Chromium | **Chrome for Testing 138.x** + `ExtensionManifestV2Availability=2` policy |
| Provider | bundled Mini CSP, no system CSP needed |

## Remaining integration work (to ship the clean-machine path)

These are the concrete steps to wire the working combination into the launcher
as a **temporary legacy compatibility profile**:

1. **Re-pin the plug-in to 2.0.15000.** `build/cryptopro-plugin-lock.json`
   now points at the immutable 2.0.15000 bundle on project static storage,
   `cryptoProPluginVersion` is `2.0.15000`, and `cryptoProPluginLayout` is
   bumped to `2` so stale 2.0.15700 extractions are not reused. This is a
   deliberate rollback pin, not the long-term target stack.
2. **Switch the bundled extension to MV2 `1.2.13`.**
   `payload-template/extensions/cryptopro-cades/` now contains the legacy MV2
   extension, with extension id `iifchhfnnmpdbibifmljnfjhpififfog` derived from
   `manifest.key`. The native-messaging `allowed_origins` continue to be
   generated from the detected extension id.
3. **Chromium 138 pin and MV2 policy.** `build/chromium-runtime.json` pins
   Chrome for Testing `138.0.7204.183`. The launcher now applies the per-user
   Chrome policy `ExtensionManifestV2Availability=2` only when the payload
   contains a loadable Manifest V2 extension, keeping the future MV3/latest
   Chromium path reversible.
4. **Launcher startUrl** points at the internal-csp test page
   (`…/cryptopro-cades-test/internal-csp/demopage/cades_bes_sample.html`) so a run
   actually sets `EnableInternalCSP` and exercises the bundled Mini CSP;
   `allowedOrigins` is `https://mescheryakov.pro` and `windowMode` is `app`.
5. **End-to-end check** with a Rutoken: certificate enumeration + `SignCades`
   through the bundled stack (still pending; see `docs/worklog.md`).

## Clean-machine (portable, no-MSI) blocker — paused, awaiting vendor fix

The MV2 stack works end-to-end **when the plug-in is installed via MSI
(`ADDMINICSP=1`)**. Running it **portably from our extracted directory on a clean
machine** is currently blocked by a CryptoPro bug: `npcades.dll` and `cades.dll`
resolve their module/provider paths from a **hardcoded preferred image base**
(`GetModuleFileName(0x10000000)`) instead of the real `HINSTANCE`; under ASLR this
misses and the plug-in falls back to `%ProgramFiles%\…`, which is absent on a clean
machine (hence the `mydss.dll installation path` error and `0.0.0000` version).

Full root-cause analysis, everything we tried (registry `AppPath`, junction, ASLR
disable, byte-patching — all dead ends), the Go stdin/stdout probe findings, and the
exact vendor report (affected RVAs) are in
**`docs/cryptopro-portable-plugin-findings.md`**. We are paused here pending a fixed
plug-in build from CryptoPro that resolves paths relative to each module.

## Future goals

- **Return to a fresh stack once CryptoPro ships a fixed MV3-compatible plug-in
  build.** MV2 + Chrome 138 is a deliberate, temporary measure: the pinned
  Chromium stops receiving security updates and MV2 is gone from current Chrome.
  Track this with CryptoPro and migrate back to the latest Chromium + a
  Manifest V3 extension when a working plug-in build is available.
- Keep the two-mode behavior: use a system CryptoPro CSP when present; otherwise
  activate the bundled Mini CSP.

## Safety / engineering constraints (unchanged)

- Do not install files into `Program Files`; prefer per-user (HKCU) configuration
  over HKLM; do not require administrator rights unless a hard CryptoPro
  requirement is documented.
- Keep all CryptoPro/Chromium binaries out of Git; fetch/generate via build
  scripts, pinned by SHA-256/size lock files.
- Runtime extraction uses staging + atomic rename + ready/state files.
- Do not disable or bypass the CryptoPro user confirmation dialog — it is part of
  the expected security model.
