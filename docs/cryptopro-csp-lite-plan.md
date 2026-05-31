# CryptoPro CSP Lite integration plan

## Goal

Move Kriptosfera from "works when system CryptoPro CSP is already installed" to "works on a clean Windows machine using Kriptosfera-managed CryptoPro CSP Lite / Mini CSP components".

The immediate target is still the CryptoPro CAdES-BES demo page:

- extension is detected;
- native Browser Plugin is detected;
- plugin version is non-zero;
- CryptoPro provider is detected;
- the standard CryptoPro access confirmation dialog appears when the page tries to access certificates/keys;
- certificate enumeration works when a supported token/certificate source is available.

Signing with Rutoken is the next validation after provider activation and certificate enumeration are stable.

## Current evidence

Manual Windows validation on 2026-05-20 established three important states.

### Machine with system CryptoPro CSP installed

Regular Chrome:

- extension loads;
- plugin loads;
- plugin version is reported as `2.0.15700`;
- CSP version is reported as `5.0.13455`;
- provider is `Crypto-Pro GOST R 34.10-2012 Cryptographic Service Provider`;
- access confirmation dialog appears for the CryptoPro demo page;
- certificates are enumerated after user approval.

Kriptosfera launcher on the same machine behaves the same way:

- bundled extension and bundled native plugin layer load;
- plugin version is reported as `2.0.15700`;
- the system CryptoPro CSP is detected and used;
- the same access confirmation dialog appears;
- approving the dialog loads CSP and enumerates certificates;
- denying the dialog produces the expected user-cancelled error `0x000004C7`.

Conclusion: extension delivery, native messaging, embedded Browser Plugin extraction, HKCU native host registration, and the basic security prompt flow are working.

### Machine without system CryptoPro CSP installed

Kriptosfera launcher:

- extension loads;
- plugin loads;
- CSP is not loaded;
- plugin version is reported as `0.0.0000`;
- no access confirmation dialog appears;
- certificates are not enumerated.

Conclusion: the bundled Browser Plugin layer is not self-contained. It starts, but without an active CSP/provider layer its `CAdESCOM.About` state is incomplete and the page cannot reach certificate/key operations.

## Ground truth from binary analysis (2026-05-31)

Static analysis of the actual bundled binaries (and a reference CryptoPro CSP
Lite install on Linux) ruled out several earlier hypotheses by fact:

- **Files/layout are complete.** Our bundle's plugin root and `Mini CSP` folder
  match a real `addminicsp` install file-for-file. Nothing is missing.
- **No registry needed.** `cpsuprt.dll` reads config through CryptoPro's own
  `support_registry_*` abstraction; the `config.ini` sections ARE the
  "registry". On a real `addminicsp` machine there is no `Crypto Pro` key under
  `HKCU`/`HKLM\...\WOW6432Node`. So missing registry is NOT the blocker.
- **License is bundled.** `Mini CSP\license.ini` carries a `ProductID`, and
  `npcades.dll` reads it via `\local\license\ProductID\{50F91F80-...}`. License
  is not a blocker for enumeration.
- **No extra runtime.** Mini CSP DLLs import only `KERNEL32/ADVAPI32/msvcrt/
  ntdll` — no MFC/VC++ redist required.
- **Bitness is settled.** `nmcades.exe`/`npcades.dll` are 32-bit, so they load
  the 32-bit `Mini CSP\capi20.dll` and read `config.ini` (not `config64.ini`).
  `config.ini` already defines provider types 75/80/81 with `Image Path =
  cpcspi.dll`.
- **Activation mechanism (confirmed in `npcades.dll`):** the string
  `result = cadesplugin.EnableInternalCSP` sits next to `Mini CSP\capi20.dll`,
  with `GetModuleFileNameW`, `LoadLibraryExA(capi20.dll) failed.`,
  `AddAvailableCsps`, and `Check containers for provider: %s, type: %d`. So
  npcades asks the page for `EnableInternalCSP`; if true it builds a
  module-relative `Mini CSP\capi20.dll` path and `LoadLibraryEx`-loads it, then
  enumerates providers from `config.ini`. No registry, wrapper, or flattening
  required.

Remaining hypotheses for "providers not enumerated on a clean machine":

- **A — flag timing.** npcades reads `EnableInternalCSP` early via a page
  callback. If the page sets the flag only after the API loads (or too late),
  npcades sees `false`/`undefined` and never loads Mini CSP. Symptom: no
  `Load Image` for `Mini CSP\capi20.dll` at all.
- **B — DLL search path.** `LoadLibraryEx(Mini CSP\capi20.dll)` can fail to
  resolve capi20's own dependencies (`asn1*.dll`) if the process working
  directory / search path is wrong. Symptom: `Load Image` for capi20 followed
  by `NAME NOT FOUND` on `asn1*.dll` or `config.ini`.
- **C — integrity self-test.** If repackaging changed a file, an integrity
  check could refuse to initialize silently. Test by comparing SHA-256 of our
  Mini CSP files against the official MSI.

The diagnostics page now sets `EnableInternalCSP` before the API loads, keeps
re-asserting it, records a flag timeline, and prints an explicit A/B/C verdict,
so a single clean-machine run distinguishes these without ProcMon.

### Diagnostics run result (2026-05-31): hypotheses A and B refuted

First run of the public `diagnostics.html` in regular Chrome against a fresh
**system `ADDMINICSP=1` install** (the same install captured in
`docs/minicsp-snapshots/installed-addminicsp/`):

- Plugin works: `cadesplugin ready`, `CAdESCOM.About` → `PluginVersion` /
  `Version` = `2.0.15700`; `CAdESCOM.Store` opens the `My` store
  (`Certificates.Count = 0`, no token). Extension → native host → plugin is fine.
- **Flag delivered early:** `EnableInternalCSP` is `true` *before*
  `cadesplugin_api.js` (`after-set (pre-api inline)` → `true` at +0 ms) and stays
  `true` (+82 ms after the API parses, and after `cadesplugin ready`).
- **Providers still absent:** `About.CSPName` / `CSPVersion` for types **75 / 80 /
  81** all return **`0x80090017` (`NTE_PROV_TYPE_NOT_DEF`)**.

**Hypothesis A (flag timing) is refuted** — the flag is delivered correctly and
early, yet Mini CSP providers never enumerate. The page's verdict is **B/C**:
`npcades` did not load `Mini CSP\capi20.dll` (or its `asn1*.dll` / `config.ini`
deps), or an integrity self-test failed. Note this is the **official** system
`ADDMINICSP=1` install, so the gap is in the internal-CSP activation mechanism
itself, not in our repackaging.

Secondary signal: the `extension version response` query **timed out after
3000 ms** while `CreateObjectAsync` CAdES calls still worked — a possible sign the
native bridge did not actually deliver the page-side flag to `npcades`.

A read-only module probe of the live `nmcades.exe`
(`tools/windows/inspect-cryptopro-modules.ps1`, no elevation) settled the next
layer: **`Mini CSP\capi20.dll` is loaded** (3 of 4 hosts), with its
`asn1*`/`cpsuprt` deps, and in one host `cpcspi.dll` (the `config.ini`
`Image Path`) loaded too. So the flag reached native, the DLL search path is fine,
and `config.ini` was read far enough to resolve the provider DLL — **hypothesis B
is also refuted.** Evidence:
`docs/minicsp-snapshots/installed-addminicsp/files/nmcades-loaded-modules.txt`.

That leaves two candidates for `0x80090017`: (C) provider-type registration /
self-test inside the loaded `capi20`/`cpcspi` not completing, or `About.CSPName`
simply not enumerating an **in-process** internal CSP that is absent from the
(empty) system provider table — a probe false-negative. The decisive test is an
actual `CryptAcquireContext` / sign with a GOST token (`Certificates.Count` was 0
here, so it could not run). A ProcMon trace of `config.ini` `ReadFile` + any
self-test on `nmcades.exe` would separate (C) from the probe-semantics case, but
needs elevation + GUI access. Run evidence: `docs/minicsp-snapshots/CONCLUSIONS.md`
§6.

## Working hypothesis

The current embedded `cadescom-x64.msi` extraction gives us the Browser Plugin/native bridge layer and includes a `Mini CSP` directory, but the extracted AppData-only layout does not yet activate that provider layer.

The plugin can see and use a system-installed CryptoPro CSP. That is valuable fallback behavior for MVP, but it is not the final product goal.

The `0.0.0000` plugin version on a clean machine should be treated as a symptom of missing/inactive CSP/provider activation, not as a string-formatting bug.

Follow-up diagnostics on 2026-05-24 narrowed the clean-machine path further. `CAdESCOM.Store` can open the Windows `MY` store and enumerate certificates even when CryptoPro provider types 75/80/81 are absent. A user certificate with `HasPrivateKey=true` is visible, but its private key is exposed as `Microsoft Smart Card Key Storage Provider` with `ProviderType=0`. `CAdESCOM.CadesSignedData.SignCades` fails on that key with `0x80090014` once certificate-chain inclusion is disabled. Therefore, plain CNG/KSP visibility is not enough for standard CAdESCOM site compatibility.

The main implementation direction is now **CSP-compatible activation**: the bundled Mini CSP/CSP Lite layer must appear to CAdESCOM as a legacy CryptoAPI CSP provider with the expected provider type/name, and usable certificates must carry CSP-style `CERT_KEY_PROV_INFO`. Direct CNG/KSP signing remains a fallback/native-product option, but it is not sufficient for ordinary third-party pages that call CryptoPro CAdESCOM APIs.

## Product decision for MVP

Support two modes explicitly:

1. **System CSP mode**
   - If a normal CryptoPro CSP is installed, Kriptosfera may use it.
   - This is acceptable for MVP and useful in enterprise environments where CSP is already managed.
   - Diagnostics should say that system CSP was detected.

2. **Bundled CSP Lite mode**
   - If no system CSP is installed, Kriptosfera should activate its bundled CSP Lite / Mini CSP components from the app directory.
   - This is the desired "clean machine" path.
   - Until implemented, diagnostics should say bundled CSP Lite is not active yet.

Do not hide this behind vague "plugin failed" wording.

## Safety constraints

- Do not install files into `Program Files`.
- Do not require administrator rights unless we hit a hard CryptoPro requirement and document it.
- Prefer HKCU/per-user configuration over HKLM.
- Keep all CryptoPro binary artifacts out of Git.
- Continue using VDSina/`mescheryakov.pro` as the artifact CDN.
- Build scripts must verify size and SHA-256 before embedding.
- Runtime extraction must use staging + atomic rename + ready/state files.
- Registry writes must be idempotent and limited to the minimum keys actually needed.
- If a registry/config experiment is added, add diagnostics and rollback/overwrite behavior before relying on it.
- Do not disable or bypass the CryptoPro user confirmation dialog. The prompt is part of the expected security model.

## Phase 0 - Diagnostics before activation

Objective: make the current states observable before changing CSP behavior.

Implemented diagnostic slice:

- hosted `diagnostics.html` loads CryptoPro's official `cadesplugin_api.js`;
- it checks extension API readiness through the same public API path used by the CryptoPro demo page;
- it creates `CAdESCOM.About` through `cadesplugin.CreateObjectAsync`;
- it probes `PluginVersion`, `Version`, `CSPVersion("", 80)`, `CSPName(80)`, and `EnableInternalCSP`;
- every call is shown with ok/error status and exact error text.

Still useful to add later:

- native host manifest path and host exe path;
- extracted Browser Plugin path;
- certificate store open result and certificate count;
- normalized HRESULT field where the browser object exposes it separately from the message.

The current slice intentionally does not open the certificate store. That keeps this step diagnostic-only and avoids triggering the certificate access prompt.

Exit criteria:

- Done: on system-CSP machine, diagnostics show plugin `2.0.15700`, CSP `5.0.13455`, and provider name.
- Done: on clean machine, diagnostics show `0.0.0` plugin/provider state and exact `0x80090017` provider errors.

## Phase 1 - Inventory Mini CSP / CSP Lite files

Objective: understand exactly what is already inside the current bundle and what is missing.

First implementation slice:

- expand launcher-side bundle validation from native-host-only files to the CAdES runtime and core Mini CSP files;
- keep token-specific DLLs diagnostic-only until the clean-machine matrix proves which token path is needed;
- do not add activation logic, registry writes, wrapper processes, or DLL search-path changes in this slice.

Tasks:

1. Inventory files under:
   `Program Files/Crypto Pro/CAdES Browser Plug-in/Mini CSP/`
2. Record PE metadata, signatures, hashes, and dependencies for key DLLs/exes:
   - `capi20.dll`
   - `cpcspi.dll`
   - `cpconfig.exe`
   - `cplib.dll`
   - token-related DLLs such as `rutoken.dll`, `pcsc.dll`, `jacarta.dll`
3. Validate that the extracted Browser Plugin layout contains at least:
   - `nmcades.exe`
   - `nmcades.json`
   - `npcades.dll`
   - `cades.dll`
   - `xades.dll`
   - `cplib.dll`
   - `Mini CSP/capi10.dll`
   - `Mini CSP/capi20.dll`
   - `Mini CSP/cpcspi.dll`
   - `Mini CSP/cpsuprt.dll`
   - `Mini CSP/cpui.dll`
4. Compare this with a normal installed CryptoPro CSP machine:
   - relevant registry keys under `Software\\Crypto Pro\\...`;
   - AppPath-like values;
   - provider names;
   - installed file paths;
   - user-level settings.
5. Record only facts, not guessed registry emulation.

Exit criteria:

- Launcher rejects a bundle that lacks the core CAdES runtime or Mini CSP files needed for the next diagnostics step.
- New inventory section documents which Mini CSP files exist and which system/registry locations `npcades.dll` references.
- We know whether the currently bundled plugin archive already contains enough CSP Lite bits to attempt activation.

## Phase 1.5 - Read-only runtime diagnostics

Objective: make each launcher run leave a machine-readable snapshot of the CryptoPro runtime layout before any activation experiment.

Implemented diagnostic file:

```text
<appDir>/diagnostics/cryptopro-runtime.json
```

It records:

- `appDir`;
- extracted CryptoPro Browser Plugin root;
- selected CryptoPro extension id;
- embedded plugin bundle component/version/SHA-256/layout version;
- native messaging host name, generated manifest path, native host executable path, registration status, and expected HKCU registry key;
- expected CAdES/Mini CSP files with suffix, resolved path, exists flag, SHA-256, and error if missing/unreadable.

This is intentionally read-only. It does not activate Mini CSP, write CSP registry/config values, change DLL search paths, or introduce a native-host wrapper.

Exit criteria:

- A manual launcher run on each test machine produces `cryptopro-runtime.json`.
- The file is enough to compare clean/system-CSP machines for extracted file presence and native host path correctness.

## Phase 1.6 - Read-only loaded module diagnostics

Objective: prove which CryptoPro/CAdES/Mini CSP DLLs are actually loaded by `nmcades.exe` before any activation experiment.

Implemented diagnostic tool:

```text
tools/windows/inspect-cryptopro-modules.ps1
```

It records:

- every running `nmcades.exe` process id/path;
- full loaded module list for each `nmcades.exe`, plus a filtered CryptoPro-focused subset;
- filtered loaded modules whose name/path mentions CryptoPro, CAdES, Mini CSP, token support, or PC/SC terms;
- module path origin classified as `app`, `system`, `windows`, `other`, or `unknown`;
- file/product versions where Windows exposes them;
- related running processes whose name/path matches the same CryptoPro/CAdES/token filters;
- module access errors, if Windows refuses module enumeration.

Default output:

```text
<appDir>/diagnostics/cryptopro-modules.json
```

The script is intentionally read-only. It does not activate Mini CSP, write CSP registry/config values, change DLL search paths, or introduce a native-host wrapper.

Exit criteria:

- On the clean machine, the report shows whether `nmcades.exe` loads bundled CAdES DLLs and Mini CSP DLLs from AppData.
- On the system-CSP machine, the report shows whether extra modules are loaded from `C:\Program Files\Crypto Pro\...`.
- If `nmcades.exe` is not running, the report says `process_not_found`; in that case the diagnostics page must be opened first so the extension starts the native host.

## Phase 2 - Read official behavior where possible

Objective: avoid blind registry poking.

Tasks:

1. Check CryptoPro documentation / installer metadata for:
   - Mini CSP;
   - internal CSP / `EnableInternalCSP`;
   - portable/no-admin use;
   - CAdES Browser Plugin + CSP Lite deployment.
2. Inspect extension scripts around `EnableInternalCSP_request`.
3. Inspect `npcades.dll` references only to guide experiments, not to reverse undocumented behavior beyond what is necessary.

Exit criteria:

- We have a written activation hypothesis with sources or direct observed evidence.
- If official documentation says portable CSP Lite is unsupported, record that before continuing.

## Phase 3 - Minimal per-app activation experiment

Objective: make the clean machine report non-zero plugin/provider state without installing system-wide CSP.

Possible experiments, in order:

1. Set process environment / working directory so `npcades.dll` finds bundled Mini CSP files beside the plugin.
2. Generate a local config file if `Mini CSP/config.ini` requires path adjustments.
3. Add minimal HKCU registry keys that point CryptoPro AppPath/settings to the extracted AppData directory.
4. Prefer user-level keys and Kriptosfera-owned paths.
5. Avoid HKLM and COM registration unless evidence shows there is no user-level option.

Exit criteria:

- Clean machine changes from:
  `plugin loaded, CSP not loaded, plugin version 0.0.0000`
  to:
  non-zero plugin version and a named provider, or a precise hard blocker.

## Phase 4 - Bundle as launcher-managed component

Objective: turn the successful experiment into deterministic launcher behavior.

Add a `CryptoProCspLiteManager` separate from `CryptoProPluginManager`.

Runtime layout target:

```text
<appDir>/
  cryptopro/
    plugin/
      ...
    csp-lite/
      ...
  .cryptopro-csp-lite-state.json
  .cryptopro-csp-lite-ready
```

State fields:

```json
{
  "component": "cryptopro-csp-lite",
  "version": "<version>",
  "sha256": "<archive sha256>",
  "layoutVersion": 1
}
```

If the CSP Lite files remain part of the current plugin bundle, record that explicitly and make the manager prepare the `Mini CSP` sublayout from the already extracted plugin. If CryptoPro supplies a separate CSP Lite package, use a separate lock file and archive.

Exit criteria:

- First run prepares CSP Lite deterministically.
- Repeat run reuses it.
- Broken/missing files trigger recovery.
- No CryptoPro binaries are committed to Git.

## Phase 5 - Runtime detection and diagnostics modes

Objective: make the launcher and diagnostics explain which crypto layer is active.

Detect and report:

- system CSP present;
- bundled CSP Lite prepared;
- bundled CSP Lite active;
- provider name/version;
- access prompt observed / user cancelled;
- certificate count.

Expected modes:

- `system-csp`;
- `bundled-csp-lite`;
- `plugin-only-no-csp`;
- `error`.

Exit criteria:

- Diagnostics page distinguishes system CSP success from bundled CSP Lite success.
- Clean machine failure no longer looks like a generic plugin problem.

## Phase 6 - Manual validation matrix

Use at least these Windows states:

1. Clean machine, no system CryptoPro CSP.
2. Machine with normal CryptoPro CSP installed.
3. Clean machine after first Kriptosfera run, then second run.
4. User approves access prompt.
5. User denies access prompt.
6. Token/certificate present.
7. No token/certificate present.

For each state, record:

- extension status;
- plugin status;
- plugin version;
- CSP status/version/provider;
- prompt behavior;
- certificate count;
- signing readiness;
- launcher logs.

Exit criteria:

- Clean machine reaches provider detection and certificate enumeration through bundled CSP Lite, or the exact unsupported requirement is documented.
- System-CSP machine remains working and unchanged.

## Rollback and cleanup

Before writing any new registry key or config path, define cleanup behavior:

- overwritten HKCU values are deterministic and can be rewritten by later runs;
- keys created only for Kriptosfera use a clear owner/path convention where possible;
- no broad deletion of existing CryptoPro user settings;
- if system CSP exists, do not disrupt it.

## Known risks

### R1 - CSP Lite may not be legally/technically redistributable in the same way

Kirill has permission for redistribution of relevant CryptoPro binaries, but CSP Lite packaging and license terms should still be tracked separately from Browser Plugin packaging.

### R2 - Mini CSP may depend on installer-created registry state

If so, we need the minimum user-level state, not a full system install.

### R3 - CSP provider registration may require HKLM/admin

If confirmed, the MVP path changes:

- either require preinstalled system CSP for MVP;
- or add an explicit installer/admin step;
- or ask CryptoPro for a supported portable deployment recipe.

### R4 - Token drivers may still require system components

Bundled CSP Lite does not automatically mean every token works. Rutoken/PCSC/driver state must be tested separately.

### R5 - Security prompt behavior must remain intact

Do not suppress the CryptoPro prompt. If a trusted-site whitelist is later required, treat it as separate product policy, not an MVP shortcut.

## Immediate next step

Start Phase 1: inventory the bundled Mini CSP / CSP Lite files and compare them with the working system-CSP machine.

Phase 0 is complete enough for this transition: the two-machine diagnostic matrix now distinguishes the working system-CSP state from the clean-machine missing-provider state without adding more launcher plumbing.
