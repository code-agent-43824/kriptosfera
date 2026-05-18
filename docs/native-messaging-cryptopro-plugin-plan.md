# Native messaging + CryptoPro plugin plan

## Goal

Bring the MVP from "the CryptoPro extension is loaded" to "the CryptoPro test page detects both the extension and the native CryptoPro Browser Plugin".

This plan intentionally stops before Rutoken/certificate/signing. The success target for this step is detection and a clean native messaging connection.

## Current baseline

Already done:

- pinned Chromium runtime is delivered in payload;
- CryptoPro CAdES Browser Plug-in extension 1.3.17 is delivered as unpacked extension;
- extension id is stable: `pfhgbfnnjiafkhfdkmpiflachepdcjod`;
- diagnostics page can load `chrome-extension://pfhgbfnnjiafkhfdkmpiflachepdcjod/nmcades_plugin_api.js`;
- launcher writes extension diagnostics and starts Chromium with `--disable-extensions-except` / `--load-extension`.

Important constraint:

- the committed extension calls `chrome.runtime.connectNative("ru.cryptopro.nmcades")` from `background.js`.
- For MVP, do not patch this extension host name. Register a user-space native messaging host with that exact name.

## Reference facts

Chrome native messaging on Windows requires:

- a host manifest JSON with:
  - `name`
  - `description`
  - `path`
  - `type: "stdio"`
  - `allowed_origins`
- a registry key under HKCU or HKLM:
  - `HKCU\Software\Google\Chrome\NativeMessagingHosts\<host-name>`
  - default value = full path to the manifest JSON
- no wildcard in `allowed_origins`;
- the native host talks over stdin/stdout with a 32-bit native-endian length prefix before each UTF-8 JSON message.

For this project the manifest should be:

```json
{
  "name": "ru.cryptopro.nmcades",
  "description": "CryptoPro CAdES Browser Plugin native host",
  "path": "C:\\...\\native-host\\cryptopro\\<actual-host-binary>.exe",
  "type": "stdio",
  "allowed_origins": [
    "chrome-extension://pfhgbfnnjiafkhfdkmpiflachepdcjod/"
  ]
}
```

## Proposed payload layout

Keep the layout explicit and boring:

```text
payload/
  extensions/
    cryptopro-cades/
      manifest.json
      ...
  native-host/
    cryptopro/
      README.md
      host-manifest.template.json
      bin/
        <cryptopro-native-host.exe>
        <required-plugin-dlls>
        <required-runtime-files>
  diagnostics/
    diagnostics.html
    extension-status.js
    native-messaging-status.js
  cryptopro/
    plugin/
      <if source package separates plugin files>
    csp-lite/
      <future step, not required for detection-only milestone unless plugin needs it>
```

Do not mix our future debug host with the real CryptoPro host in the same directory. If a debug host is needed, put it under:

```text
payload/native-host/debug/
```

and use a separate host name for it.

## Implementation phases

### Phase 0 — Source and inventory the CryptoPro native binaries

Objective: know exactly what we are shipping before wiring it.

Tasks:

1. Obtain the official CryptoPro Browser Plugin Windows package/archive from a controlled source.
2. Extract it into a temporary, non-committed workspace.
3. Identify:
   - native messaging host executable;
   - native host manifest, if the package contains one;
   - required DLLs/runtime files;
   - registry keys normally created by the installer;
   - whether the host depends on installed services, COM registration, CSP, VC runtime, or PATH entries.
4. Record the source URL/package version/checksum in `payload-template/native-host/cryptopro/README.md`.
5. Decide whether binaries can be committed to this public repo. If not, use a pinned private artifact/download step.

Exit criteria:

- exact file list is known;
- host name is confirmed as `ru.cryptopro.nmcades`;
- host binary can be run manually enough to show it starts or fails with a specific missing dependency.

### Phase 1 — Add payload scaffold and required-file checks

Objective: make payload shape stable without changing launcher behavior yet.

Tasks:

1. Add `payload-template/native-host/cryptopro/README.md`.
2. Add `host-manifest.template.json` with placeholders for:
   - host path;
   - extension origin.
3. If binaries are allowed in repo, add the minimal real files under `bin/`.
4. Update `build/prepare-payload.ps1` required checks after real host files exist.
5. Ensure payload manifest includes native-host files.

Exit criteria:

- payload package contains the native-host layout;
- CI builds payload and launchers;
- no registry writes yet.

### Phase 2 — Launcher-side manifest generation

Objective: generate the real native messaging manifest inside the extracted app dir.

Tasks:

1. Add a small Go helper, e.g. `internal/bootstrap/native_messaging.go`.
2. Detect the stable extension id from the already detected CryptoPro extension.
3. Resolve the native host binary path under:
   - `<appDir>/native-host/cryptopro/bin/<actual-host-binary>.exe`
4. Write:
   - `<appDir>/native-host/cryptopro/ru.cryptopro.nmcades.json`
5. Manifest content must use:
   - `name: "ru.cryptopro.nmcades"`
   - `type: "stdio"`
   - `allowed_origins: ["chrome-extension://pfhgbfnnjiafkhfdkmpiflachepdcjod/"]`
   - absolute Windows path to the host binary
6. Add unit tests for generated JSON and missing-host behavior.

Exit criteria:

- launcher can generate a valid manifest deterministically;
- missing binary produces a clear diagnostic/error, not a silent failure.

### Phase 3 — HKCU registration

Objective: register the native host for the current user with no admin rights.

Tasks:

1. Add Windows-only registry helper.
2. Write default value:
   - key: `HKCU\Software\Google\Chrome\NativeMessagingHosts\ru.cryptopro.nmcades`
   - value: full manifest path
3. Do not use HKLM for MVP.
4. Log:
   - key path;
   - manifest path;
   - whether the value was created/updated/reused.
5. Add non-Windows stub so tests/builds stay simple.
6. Add tests around path/key construction where possible; registry write itself can be covered by Windows CI smoke later.

Exit criteria:

- launcher registers the host before Chromium starts;
- repeated runs are idempotent;
- diagnostics can show the current registered manifest path.

### Phase 4 — Native messaging diagnostics

Objective: know whether Chrome can see and start the host.

Tasks:

1. Extend launcher diagnostics:
   - manifest path;
   - binary path;
   - host name;
   - HKCU registry path;
   - registration result.
2. Extend `diagnostics.html`:
   - show native host registration status from launcher-side diagnostics;
   - trigger extension/plugin probe using `window.cpcsp_chrome_nmcades.check_chrome_plugin(...)`.
3. Keep the existing extension probe.
4. Make result states explicit:
   - extension script loaded;
   - native host registered;
   - native host connected;
   - plugin object created;
   - plugin missing/dependency error/native host not found.

Exit criteria:

- diagnostics distinguishes "extension loaded but native host missing" from "native host present but plugin/dependency failed".

### Phase 5 — First real plugin detection

Objective: the CryptoPro test page detects the extension and plugin.

Tasks:

1. Run the embedded launcher on Windows.
2. Open the configured CryptoPro demo page.
3. Verify page-level detection:
   - extension detected;
   - plugin detected.
4. Check logs:
   - `logs/launcher.log`
   - Chromium stderr/stdout logs;
   - extension popup state, if useful.
5. If detection fails, classify the failure:
   - host registry not found;
   - manifest invalid;
   - allowed origin mismatch;
   - host executable cannot start;
   - host starts but misses DLL/dependency;
   - host starts but cannot create CAdESCOM object;
   - plugin requires system CSP/COM registration and cannot run user-space yet.

Exit criteria:

- CryptoPro test page reports extension + plugin present, or a concrete blocker is documented with the exact failing layer.

## Validation sequence

Use small commits:

1. docs/scaffold only;
2. payload native-host files;
3. manifest generation;
4. HKCU registration;
5. diagnostics probe;
6. manual Windows validation result.

Automated checks per code commit:

- `go test ./...`;
- `build-windows` GitHub Actions;
- payload artifact contains native-host files;
- launcher artifact still builds both embedded and remote variants.

Manual checks on Windows:

- clean profile first run;
- repeat run;
- inspect HKCU registry key;
- verify manifest path points inside current extracted app version;
- verify extension popup moves from missing-host/error toward active;
- verify CryptoPro demo page sees plugin.

## Risk register

### R1 — Binary licensing / redistribution

The real CryptoPro Browser Plugin binaries may not be suitable for a public GitHub repo.

Decision path:

- if redistribution is allowed: commit or release-asset them with checksum;
- if not: use a private artifact/pinned download and keep only metadata/checksum in public repo.

### R2 — Host requires installed components

The native host may depend on system-installed CryptoPro CSP, COM registration, services, or drivers.

MVP handling:

- first prove native messaging host detection;
- if plugin detection needs system registration, record blocker and separate "user-space plugin packaging" from "CSP Lite/signing".

### R3 — Registry path for Chrome for Testing

Official docs say Windows native messaging uses Chrome registry keys. Our runtime is Chrome for Testing/Chromium-shaped, so verify empirically that it reads the HKCU Chrome key.

Fallback options, in order:

1. standard HKCU Chrome key;
2. Chromium key if needed;
3. profile-level native messaging host location only if Chromium supports it in this runtime;
4. patch runtime/launch strategy only if all standard paths fail.

### R4 — Extension id mismatch

This is mostly controlled by `manifest.key`. Keep generating `allowed_origins` from detected extension id rather than hardcoding it in multiple places.

### R5 — stdout pollution

Native messaging protocol uses stdout for framed JSON. Any logs written to stdout can break communication.

For our own debug host, log to stderr/file only. For CryptoPro host, capture Chrome stderr and launcher diagnostics.

## What is not included in this step

Not part of this plan:

- full Rutoken signing flow;
- certificate selection UX;
- CSP Lite packaging unless the plugin cannot even be detected without it;
- broad token compatibility;
- full domain/navigation policy;
- installer/product polish.

Those stay in later MVP/product phases.

