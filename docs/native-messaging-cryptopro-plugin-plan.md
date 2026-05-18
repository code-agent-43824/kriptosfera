# Native messaging + embedded CryptoPro binaries plan

## Goal

Bring the MVP from "the CryptoPro extension is loaded" to "the CryptoPro test page detects both the extension and the native CryptoPro Browser Plugin".

This step stops before Rutoken/certificate/signing. The immediate success target is:

- extension is detected by the CryptoPro test page;
- native CryptoPro Browser Plugin is detected by the same page;
- native messaging is registered and observable;
- all CryptoPro plugin binaries are deployed into Kriptosfera's own AppData application directory, not into system locations.

## Binding product decisions

Kirill confirmed:

- CryptoPro granted permission to redistribute the relevant binaries.
- Do not commit CryptoPro plugin or future CSP Lite binaries to GitHub.
- Do not include CryptoPro binaries in the remote Chromium payload.
- Build scripts must download CryptoPro binary bundles from Watson's own static web folder/server.
- Both launcher variants must embed the CryptoPro binary bundle:
  - embedded/thick launcher;
  - remote/thin launcher.
- At runtime, launcher extracts CryptoPro binaries into AppData next to Chromium, under the extracted app version directory.
- Do not install into system locations.
- Do not write binaries to Program Files or global CryptoPro folders.
- If CryptoPro binaries contain hardcoded paths, solve those blockers in place after we observe the real failure.
- Diagnostics remains enabled for MVP.

## Current baseline

Already done:

- pinned Chromium runtime is delivered in payload;
- CryptoPro CAdES Browser Plug-in extension 1.3.17 is delivered as unpacked extension;
- extension id is stable: `pfhgbfnnjiafkhfdkmpiflachepdcjod`;
- diagnostics page can load `chrome-extension://pfhgbfnnjiafkhfdkmpiflachepdcjod/nmcades_plugin_api.js`;
- launcher writes extension diagnostics and starts Chromium with `--disable-extensions-except` / `--load-extension`.

Important extension constraint:

- the committed extension calls `chrome.runtime.connectNative("ru.cryptopro.nmcades")` from `background.js`;
- for MVP, do not patch this extension host name;
- register a user-space native messaging host with that exact name.

## Architecture change from the previous draft

Previous draft treated native-host files as payload contents. That is no longer the target.

New target:

```text
Build time:
  static HTTPS server
    /kriptosfera/cryptopro/plugin/<version>/<sha256>/cryptopro-plugin.zip
    /kriptosfera/cryptopro/plugin/<version>/<sha256>/cryptopro-plugin.json

  build script downloads + verifies this bundle

  launcher binary embeds:
    - payload.zip               (embedded launcher only)
    - runtime-config.json
    - cryptopro-plugin.zip      (both embedded and remote launchers)
    - cryptopro-plugin metadata/checksum

Runtime:
  %LOCALAPPDATA%/Kriptosfera/apps/demo/<version>/
    chromium/
    extensions/
    native-host/
      cryptopro/
        ru.cryptopro.nmcades.json
        bin/
          <native host exe>
          <plugin dlls/runtime files>
    cryptopro/
      plugin/
        <if the official package has a separate plugin layout>
      csp-lite/
        <future, same delivery model>
    diagnostics/
    config/
```

The remote payload remains responsible for browser/runtime/application payload. CryptoPro binaries are a launcher-owned embedded asset because they are security-sensitive, version-pinned, and must be present in both launcher modes without depending on remote payload composition.

## Static server layout

Use a boring immutable layout:

```text
https://mescheryakov.pro/kriptosfera/cryptopro/
  plugin/
    <plugin-version>/
      <sha256>/
        cryptopro-plugin.zip
        cryptopro-plugin.json
  csp-lite/
    <future-version>/
      <sha256>/
        cryptopro-csp-lite.zip
        cryptopro-csp-lite.json
```

Metadata example:

```json
{
  "component": "cryptopro-browser-plugin",
  "version": "2.x.x",
  "platform": "windows-amd64",
  "archive": "cryptopro-plugin.zip",
  "sha256": "<lowercase sha256>",
  "size": 12345678,
  "source": "CryptoPro official package, redistribution permitted by CryptoPro",
  "createdAt": "2026-05-18T00:00:00Z"
}
```

Rules:

- URLs are immutable: version + sha256 path.
- Build must trust the pinned checksum, not server mutable metadata.
- The static server may expose metadata for human audit, but the build script must verify the archive checksum independently.
- No CryptoPro binaries in GitHub commits.
- Keep only source notes, checksums, and build configuration in GitHub.

Current pinned plugin bundle:

```text
version: 2.0.15700
url: https://mescheryakov.pro/kriptosfera/cryptopro/plugin/2.0.15700/c35327b079022f123a8c31e5656891d61a7e493312010a0893b76a25f15feebe/cryptopro-plugin.zip
sha256: c35327b079022f123a8c31e5656891d61a7e493312010a0893b76a25f15feebe
size: 21699162
metadata: https://mescheryakov.pro/kriptosfera/cryptopro/plugin/2.0.15700/c35327b079022f123a8c31e5656891d61a7e493312010a0893b76a25f15feebe/cryptopro-plugin.json
```

Source/audit bundle:

```text
url: https://mescheryakov.pro/kriptosfera/cryptopro/sources/2.0.15700/8b4b1bfbe801c4569c3bf23107110263a344a9b1bce30c575b9b92ff77f2c2d4/cryptopro-cades-official-2.0.15700.zip
sha256: 8b4b1bfbe801c4569c3bf23107110263a344a9b1bce30c575b9b92ff77f2c2d4
size: 38472470
```

## Build pipeline changes

### New files/scripts

Added:

- `build/cryptopro-plugin-lock.json`
- `build/cryptopro-plugin-lock.example.json`
- `build/fetch-cryptopro-plugin.ps1`
- `internal/bootstrap/cryptopro_plugin_windows.go`
- `internal/bootstrap/cryptopro_plugin_other.go`
- `internal/bootstrap/cryptopro_plugin.go`

Add later, when implementing:

- diagnostics probe and manual Windows validation against the CryptoPro test page.

Lock file shape:

```json
{
  "component": "cryptopro-browser-plugin",
  "version": "2.x.x",
  "platform": "windows-amd64",
  "url": "https://mescheryakov.pro/kriptosfera/cryptopro/plugin/2.x.x/<sha256>/cryptopro-plugin.zip",
  "metadataUrl": "https://mescheryakov.pro/kriptosfera/cryptopro/plugin/2.x.x/<sha256>/cryptopro-plugin.json",
  "sha256": "<lowercase sha256>",
  "size": 12345678
}
```

### Build flow

For both embedded and remote launchers:

1. Read `build/cryptopro-plugin-lock.json`.
2. Download `cryptopro-plugin.zip` from the static server into `internal/bootstrap/cryptopro-plugin.zip`.
3. Download `cryptopro-plugin.json` into `dist/cryptopro-plugin.json`.
4. Verify SHA-256 and size against the lock file.
5. Verify downloaded metadata matches the pinned SHA-256 and size.
6. Embed the verified zip through Go `embed` on Windows builds.
7. Build:
   - `KriptosferaDemo.exe`
   - `KriptosferaDemo-remote.exe`
8. Do not put the CryptoPro bundle into `payload.zip`.
9. Do not upload CryptoPro source archives as GitHub workflow artifacts; launcher executables contain the embedded bundle by design.

Important: if the launcher artifact itself contains embedded CryptoPro binaries, publishing the launcher is already redistribution. That is acceptable per Kirill's permission note, but keep the source archives out of GitHub.

## Runtime extraction model

Add a launcher-owned "component manager" separate from `PayloadManager`.

Suggested state:

```text
<appDir>/
  .cryptopro-plugin-state.json
  .cryptopro-plugin-ready
```

State fields:

```json
{
  "component": "cryptopro-browser-plugin",
  "version": "2.x.x",
  "sha256": "<archive sha256>",
  "layoutVersion": 1
}
```

Runtime flow:

1. Prepare normal payload first, so `appDir` exists.
2. Prepare CryptoPro plugin bundle second:
   - check `.cryptopro-plugin-ready` and state;
   - verify extracted files if a component manifest exists;
   - if missing/stale/broken, extract embedded `cryptopro-plugin.zip` into a staging dir;
   - validate required files;
   - rename staging into:
     - `<appDir>/native-host/cryptopro/`
     - and/or `<appDir>/cryptopro/plugin/`, depending on the official package layout.
3. Generate native messaging manifest using paths inside `appDir`.
4. Register HKCU native messaging key.
5. Start Chromium.

This keeps the remote/thin launcher independent of remote payload for CryptoPro assets while still using the same extracted app version directory.

## Proposed AppData layout

Target layout after first run:

```text
%LOCALAPPDATA%/Kriptosfera/
  apps/
    demo/
      <version>/
        chromium/
        extensions/
          cryptopro-cades/
        native-host/
          cryptopro/
            ru.cryptopro.nmcades.json
            bin/
              <native host exe>
              <required plugin dlls/runtime files>
        cryptopro/
          plugin/
            <optional official plugin files if separate from native-host/bin>
          csp-lite/
            <future>
        diagnostics/
          diagnostics.html
          extension-status.js
          native-messaging-status.js
        config/
          app-config.json
        manifest.json
        .payload-state.json
        .payload-ready
        .cryptopro-plugin-state.json
        .cryptopro-plugin-ready
  profiles/
    demo/
  logs/
    launcher.log
    chromium.stdout.log
    chromium.stderr.log
```

Keep CryptoPro next to Chromium and extension in the same versioned app dir. Do not share it globally across app versions until we have a real need.

## Native messaging manifest

Chrome native messaging on Windows requires:

- a manifest JSON with:
  - `name`
  - `description`
  - `path`
  - `type: "stdio"`
  - `allowed_origins`
- a registry key under HKCU or HKLM:
  - `HKCU\Software\Google\Chrome\NativeMessagingHosts\<host-name>`
  - default value = full path to the manifest JSON
- no wildcard in `allowed_origins`;
- host communication over stdin/stdout with a 32-bit native-endian length prefix before each UTF-8 JSON message.

For Kriptosfera:

```json
{
  "name": "ru.cryptopro.nmcades",
  "description": "CryptoPro CAdES Browser Plugin native host",
  "path": "C:\\Users\\...\\AppData\\Local\\Kriptosfera\\apps\\demo\\<version>\\native-host\\cryptopro\\bin\\<actual-host-binary>.exe",
  "type": "stdio",
  "allowed_origins": [
    "chrome-extension://pfhgbfnnjiafkhfdkmpiflachepdcjod/"
  ]
}
```

Generate this file at runtime because `appDir` is versioned and user-specific.

## Implementation phases

### Phase 0 — Static server and binary inventory

Objective: establish a controlled binary source outside GitHub.

Tasks:

1. Use the prepared static web directory documented in `docs/cryptopro-static-bundles.md`.
2. Upload the official CryptoPro Browser Plugin Windows package or a normalized extracted bundle archive.
3. Generate metadata:
   - version;
   - platform;
   - sha256;
   - size;
   - source note;
   - redistribution permission note.
4. Inspect package contents in a temporary non-repo workspace.
5. Identify:
   - native messaging host executable;
   - native host manifest, if present;
   - required DLLs/runtime files;
   - registry keys the official installer normally writes;
   - whether host expects fixed paths, COM registration, services, CSP, VC runtime, PATH, or current working directory.
6. Decide normalized archive layout for `cryptopro-plugin.zip`.

Exit criteria:

- HTTPS URL is available: done;
- checksum/size are known: done;
- exact file list is known: done for `cadescom-x64.msi` extraction;
- no binaries are committed to GitHub.

Inventory results from CryptoPro CADESCOM 2.0.15700:

- signed official MSI: `cadescom-x64.msi`;
- product/version: `CryptoPro CADESCOM` / `2.0.15700`;
- native messaging host name: `ru.cryptopro.nmcades`;
- native host executable: `Program Files/Crypto Pro/CAdES Browser Plug-in/nmcades.exe`;
- Chrome native host manifest template: `Program Files/Crypto Pro/CAdES Browser Plug-in/nmcades.json`;
- Firefox manifest template: `Program Files/Crypto Pro/CAdES Browser Plug-in/nmcades_firefox.json`;
- browser plugin DLL: `Program Files/Crypto Pro/CAdES Browser Plug-in/npcades.dll`;
- official installer normally writes browser native messaging keys through MSI registry rows, including `SOFTWARE\\Google\\Chrome\\NativeMessagingHosts\\ru.cryptopro.nmcades`;
- `nmcades.json` contains `<HOST_PATH>`; our launcher must generate or patch this path at runtime;
- official `nmcades.json` already includes the stable extension id `pfhgbfnnjiafkhfdkmpiflachepdcjod`;
- `nmcades.exe` requests `asInvoker`, so starting the host itself should not require admin rights;
- the extracted archive also contains Mini CSP/runtime DLLs; first MVP attempt will deploy the extracted layout under AppData without system install.

### Phase 1 — Build-time download + launcher embedding

Objective: make both launcher variants carry the CryptoPro plugin bundle.

Tasks:

1. Add `build/cryptopro-plugin-lock.json`.
2. Add `build/fetch-cryptopro-plugin.ps1`.
3. Verify archive checksum and size in the fetch script.
4. Embed the verified archive into launcher build for both modes.
5. Ensure `build-windows.yml` builds both launchers with the same embedded CryptoPro plugin archive.
6. Keep `payload.zip` unchanged except for normal browser payload contents.

Exit criteria:

- both `KriptosferaDemo.exe` and `KriptosferaDemo-remote.exe` contain the plugin bundle: implemented by Windows `go:embed`;
- build fails closed if the static archive is missing or checksum mismatches: implemented in `build/fetch-cryptopro-plugin.ps1`;
- GitHub repo still contains no CryptoPro binaries.

Current limitation: this phase only embeds the archive and logs its size/SHA-256 at launcher startup. Runtime extraction is now handled in Phase 2; native messaging registration is intentionally left to the next phases.

### Phase 2 — Runtime extraction next to Chromium

Objective: deploy CryptoPro plugin files into `appDir` beside Chromium.

Current implementation:

- `CryptoProPluginManager` extracts the embedded `cryptopro-plugin.zip` into `<appDir>/cryptopro/plugin`;
- extraction uses staging + rename and the same bootstrap lock pattern as payload preparation;
- state is stored in `<appDir>/.cryptopro-plugin-state.json`;
- readiness is marked by `<appDir>/.cryptopro-plugin-ready`;
- required files are validated after extraction and before reuse:
  - `nmcades.exe`;
  - `nmcades.json`;
  - `npcades.dll`.

Tasks:

1. Add `CryptoProPluginManager` or similar.
2. Extract embedded `cryptopro-plugin.zip` after payload preparation.
3. Use staging + atomic rename like payload extraction.
4. Store component state and ready marker.
5. Validate required native host executable and key DLLs.
6. Log extracted version/path/checksum.

Exit criteria:

- clean first run extracts CryptoPro plugin files under `%LOCALAPPDATA%/Kriptosfera/apps/demo/<version>/...`: implemented;
- repeat run reuses existing extracted component: covered by tests;
- broken/missing extraction recovers cleanly: covered by tests.

Current limitation: the extracted files are not yet wired into a generated Chrome native messaging manifest or HKCU registry key. That remains Phase 3/4.

### Phase 3 — Manifest generation

Objective: generate the native messaging manifest from extracted paths.

Current implementation:

- `PrepareCryptoProNativeMessaging` resolves `nmcades.exe` from the extracted plugin layout;
- it writes `<appDir>/native-host/cryptopro/ru.cryptopro.nmcades.json`;
- `allowed_origins` is generated from the detected CryptoPro extension id;
- tests cover manifest generation and missing extension id.

Tasks:

1. Detect extension id from `manifest.key` as already implemented.
2. Resolve actual host binary path from extracted CryptoPro layout.
3. Write `<appDir>/native-host/cryptopro/ru.cryptopro.nmcades.json`.
4. Use:
   - `name: "ru.cryptopro.nmcades"`;
   - `type: "stdio"`;
   - `allowed_origins: ["chrome-extension://pfhgbfnnjiafkhfdkmpiflachepdcjod/"]`;
   - absolute path to the extracted host binary.
5. Add tests for manifest generation and missing binary errors.

Exit criteria:

- manifest is valid JSON: covered by tests;
- manifest path points inside current `appDir`: implemented;
- no hardcoded user path in repo or static metadata: implemented.

### Phase 4 — HKCU registration

Objective: register the native host for the current user without admin rights.

Current implementation:

- Windows builds call `reg.exe add HKCU\\Software\\Google\\Chrome\\NativeMessagingHosts\\ru.cryptopro.nmcades /ve /t REG_SZ /d <manifestPath> /f`;
- non-Windows builds use a no-op stub;
- tests stub registry writes to avoid mutating HKCU on CI.

Tasks:

1. Add Windows-only registry helper.
2. Write default value:
   - key: `HKCU\Software\Google\Chrome\NativeMessagingHosts\ru.cryptopro.nmcades`
   - value: full path to generated manifest.
3. Do not use HKLM for MVP.
4. Log created/updated/reused status.
5. Add non-Windows stub for clean builds/tests.
6. Verify whether Chrome for Testing reads the standard Chrome HKCU key. If not, test Chromium/Chrome-for-Testing specific fallback keys.

Exit criteria:

- launcher registers host before Chromium starts: implemented;
- repeated runs are idempotent: `reg.exe /f` overwrites the same value;
- diagnostics shows registered manifest path: pending Phase 5 diagnostics update.

### Phase 5 — Diagnostics and test-page plugin detection

Objective: prove the CryptoPro test page sees both extension and plugin.

Tasks:

1. Extend launcher diagnostics:
   - CryptoPro plugin bundle version/checksum;
   - extracted plugin path;
   - native host exe path;
   - generated manifest path;
   - HKCU registry key/value;
   - registration result.
2. Extend `diagnostics.html`:
   - keep current extension probe;
   - show native messaging registration status;
   - call `window.cpcsp_chrome_nmcades.check_chrome_plugin(...)`;
   - show plugin detection result separately from extension detection.
3. Run the embedded launcher on Windows.
4. Open the CryptoPro demo page.
5. Verify:
   - extension detected;
   - plugin detected.
6. If plugin detection fails, classify exact layer:
   - registry key missing/wrong;
   - manifest invalid;
   - allowed origin mismatch;
   - host executable missing;
   - host executable cannot start;
   - missing DLL/runtime dependency;
   - hardcoded path assumption;
   - COM/CSP/system registration dependency.

Exit criteria:

- CryptoPro test page reports extension + plugin present, or a precise blocker is documented.

## Validation sequence

Use small commits:

1. docs update (this document);
2. static server + lock metadata;
3. build-time download/embed;
4. runtime extraction manager;
5. manifest generation;
6. HKCU registration;
7. diagnostics probe;
8. manual Windows validation result.

Automated checks per code commit:

- `go test ./...`;
- `build-windows` GitHub Actions;
- verify both launcher variants build;
- verify build fails on checksum mismatch;
- verify `payload.zip` does not contain CryptoPro binaries;
- verify GitHub tree does not contain CryptoPro binaries.

Manual checks on Windows:

- clean profile first run;
- repeat run;
- inspect AppData layout;
- inspect HKCU registry key;
- verify manifest path points inside current extracted app version;
- verify extension popup moves from missing-host/error toward active;
- verify CryptoPro demo page sees plugin.

## Risk register

### R1 — Static server availability

Build now depends on Watson's static server for CryptoPro bundles.

Mitigation:

- immutable URLs;
- checksum lock file;
- cache downloaded archive in CI if useful;
- fail closed with clear error if unavailable.

### R2 — Launcher size

Both launcher variants will embed CryptoPro plugin binaries, increasing executable size.

Mitigation:

- measure artifact size after first integration;
- accept for MVP unless first-run UX or GitHub artifact limits become a real problem.

### R3 — Hardcoded paths inside CryptoPro binaries

The plugin may assume installer-created paths or registry entries.

MVP handling:

- deploy into our AppData folder first;
- observe exact failure;
- patch environment/current working directory/manifest/registry only as needed;
- do not pre-emptively install system-wide.

### R4 — Host requires installed components

The native host may depend on system-installed CryptoPro CSP, COM registration, services, drivers, or VC runtime.

MVP handling:

- first prove native messaging host visibility;
- then prove plugin detection;
- if plugin detection needs system registration, document exact blocker and decide the minimal local/user-space workaround.

### R5 — Registry path for Chrome for Testing

Official docs say Windows native messaging uses Chrome registry keys. Our runtime is Chrome for Testing/Chromium-shaped, so verify empirically that it reads the HKCU Chrome key.

Fallback options, in order:

1. standard HKCU Chrome key;
2. Chromium key if needed;
3. Chrome-for-Testing specific key if present/required;
4. profile-level native messaging host location only if supported by this runtime;
5. patch runtime/launch strategy only if all standard paths fail.

### R6 — stdout pollution

Native messaging protocol uses stdout for framed JSON. Any logs written to stdout can break communication.

Mitigation:

- for our debug tooling, log to stderr/file only;
- for CryptoPro host, capture Chromium stderr/stdout and launcher diagnostics.

### R7 — Future CSP Lite bundle

CSP Lite should use the same static-server -> build-download -> launcher-embed -> AppData-extract model later.

Do not design a second delivery path for CSP Lite unless CryptoPro's actual packaging forces it.

## What is not included in this step

Not part of this plan:

- full Rutoken signing flow;
- certificate selection UX;
- CSP Lite integration unless plugin detection cannot work without a minimal CSP component;
- broad token compatibility;
- full domain/navigation policy;
- installer/product polish;
- system-wide CryptoPro installation.

Those stay in later MVP/product phases.
