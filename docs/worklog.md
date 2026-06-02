# Worklog — handoff log between agents

Several agents work on this repo, sometimes in parallel. Keep a short entry per
chunk of work: **Planned / Done / Next**, newest on top. Document first, then code.
For deeper context see `docs/cryptopro-csp-lite-plan.md` and `CHANGELOG.md`.

---

## 2026-06-02 — REAL root cause: npcades.dll passes hardcoded ImageBase to GetModuleFileName + ASLR

**Found via capstone/pefile (not Ghidra):** `npcades.dll` actually *tries* to resolve
its provider/module paths **relative to its own module**, but in 3 places it calls
`GetModuleFileNameA/W(hModule = 0x10000000, …)` with the **hardcoded preferred
ImageBase** (`push 0x10000000`) instead of the real `HINSTANCE`. Sites:
`0x10004069`, `0x10054cf2` (→ builds `<dir>\Mini CSP\capi20.dll`), `0x10056637`.
The DLL has **ASLR enabled** (`DllCharacteristics=0x140`, `DYNAMIC_BASE`), so on
modern Windows it loads at a random base; `0x10000000` is not its base →
`GetModuleFileName` fails → code falls back to the `Program Files\Crypto Pro\CAdES
Browser Plug-in` path. THAT is why the provider loads from the system dir, not ours.
(So it is a CryptoPro bug, not a fundamental hardcoded-absolute-path design.)

**Patch produced (header-only, no code touched):** clear `DYNAMIC_BASE` (ASLR) in
`DllCharacteristics` `0x140 → 0x100` (file offset `0x19e`: `0x40→0x00`) + recomputed
PE checksum (offset `0x198`). With ASLR off, npcades loads at its preferred
`0x10000000`, the hardcoded `GetModuleFileName(0x10000000)` becomes correct, and the
module-relative resolution loads Mini CSP/mydss **from the dir next to our
`nmcades.exe`** — exactly our extracted bundle. No %LOCALAPPDATA% reshuffling, no
CSIDL edit. Orig sha256 `0f7ffc9a…`, patched `4c52c39b…`.

**Caveats:** only works if `0x10000000` is free in the `nmcades.exe` process (normal)
and system-wide *mandatory* ASLR (Exploit Protection "Force randomization for
images") is OFF (default). If on, fall back to a Frida hook of
`GetModuleFileNameA/W` (return our path when `hModule==0x10000000`) or junction.

**Vendor report (sharpened):** "npcades.dll calls `GetModuleFileNameA/W` with a
hardcoded `0x10000000` (preferred ImageBase) instead of the module's real
`HINSTANCE`; with `/DYNAMICBASE` this yields the wrong path under ASLR and falls
back to `Program Files`. Pass the actual `HINSTANCE` (from `DllMain`) or
`GetModuleHandleW(L"npcades.dll")`."

**Next:** owner drops the patched `npcades.dll` into the extracted bundle next to
our `nmcades.exe` and runs the launcher; if the provider loads with NO Program Files
install present, hypothesis confirmed and PoC achieved.

---

## 2026-06-02 — PoC plan: redirect npcades.dll's provider base to %LOCALAPPDATA% (owner machine, not for release)

**Goal:** owner wants to *prove* the path hypothesis on his own machine/licence
(not a release; awaiting a fixed vendor build that also moves to MV3). Legit
interop PoC.

**Static recon of `npcades.dll` (Linux, objdump):**
- **No self-integrity strings** (`integrity/tamper/crc/self-test` absent; the only
  "signature" strings are about *document* signing: `CryptVerifySignature`,
  `SignatureMethod`). So a patched `npcades.dll` will very likely load — `LoadLibrary`
  doesn't check Authenticode without WDAC. (Do NOT touch `capi20.dll`/`cpcspi.dll`
  — those are the certified CSP and almost certainly self-verify.)
- Provider base is built as `SHGetFolderPathW(CSIDL_PROGRAM_FILES* )` +
  `\Crypto Pro\CAdES Browser Plug-in\Mini CSP\capi20.dll` (and `mydss.dll` in the
  plug-in root). CSIDL immediates: `6a 2a` = `CSIDL_PROGRAM_FILESX86 (0x2a)`,
  `6a 26` = `CSIDL_PROGRAM_FILES (0x26)`.
- **Caution:** blind static localisation of the exact callsite is unreliable — a
  frequency heuristic flagged `0x1020a7f0`, but that is a C++ `__thiscall` method
  (args via `ecx`), NOT `SHGetFolderPathW`. Do the final localisation dynamically
  or in Ghidra (xref on the import), not by guessing offsets.

**Minimal-patch idea (one byte):** change the CSIDL constant feeding
`SHGetFolderPathW` from `0x2a`/`0x26` to `0x1c` (`CSIDL_LOCAL_APPDATA`) [or `0x1a`
`CSIDL_APPDATA`]. Base becomes `%LOCALAPPDATA%`, so the provider is sought in
`%LOCALAPPDATA%\Crypto Pro\CAdES Browser Plug-in\Mini CSP\…` — a no-admin,
user-writable dir (where our launcher already writes). Cleaner than rewriting the
`SHGetFolderPath→GetModuleFileName` logic.

**Recommended PoC routes (pick one):**
1. **Junction (no binary change, proves hypothesis 100%):**
   `mklink /J "C:\Program Files (x86)\Crypto Pro\CAdES Browser Plug-in" <our dir>`.
2. **Frida runtime hook (no file change, signature intact):** hook
   `shell32!SHGetFolderPathW`; if `nFolder ∈ {0x2a,0x26}` force `0x1c` (or rewrite
   the returned path to our dir). Attach to `nmcades.exe`.
3. **Ghidra file patch:** import `npcades.dll`, find import `SHGetFolderPathW`,
   follow each XREF, find the one whose decompile concatenates
   `Crypto Pro\CAdES Browser Plug-in` / `Mini CSP`, patch that `PUSH 0x2a/0x26`
   immediate → `0x1c`. Then drop our `Crypto Pro\CAdES Browser Plug-in\Mini CSP`
   (+ `mydss.dll`) under `%LOCALAPPDATA%`.

**Import-table detail (pefile):** `npcades.dll`'s static imports are 21 DLLs incl.
`MYDSS.dll`, `cades.dll`, `xades.dll`, `cplib.dll`, `cpasn1.dll` — **but NOT
`SHELL32.dll`**. So `SHGetFolderPathW` is resolved **dynamically**
(`LoadLibrary("SHELL32.dll")`+`GetProcAddress`), and a Ghidra "xref on import
SHGetFolderPathW" will find nothing. Localise instead by: (a) string xref on
`"SHGetFolderPathW"` → the `GetProcAddress` site → the global fn-ptr → its xrefs;
or (b) just a dynamic breakpoint on `shell32!SHGetFolderPathW` (works regardless).
The static imports `MYDSS/cades/xades/cplib/cpasn1` ARE pulled by the loader from
*our* dir (next to `nmcades.exe`); only the Mini CSP provider (`capi20.dll`) is
loaded via the built `<base>\…\Mini CSP` path — that's the one to redirect.
The Frida route (2) is unaffected by the dynamic resolution and stays the simplest.

**Next:** owner runs route 1 (or 2) to confirm; vendor change (resolve relative to
`GetModuleFileName(npcades)` or honor a CSIDL_LOCAL_APPDATA/env override) remains
the only release-grade fix.

---

## 2026-06-02 — Root cause: npcades.dll resolves Mini CSP via SHGetFolderPath(Program Files), not relative to itself

**Owner registry check (decisive):** on the working machine there is **no**
`Crypto Pro\Cryptography\CurrentVersion` / `AppPath` anywhere — not HKCU
(no `Crypto Pro` key at all), not `HKLM\SOFTWARE\Crypto Pro` (only `cpoids1`),
not `HKLM\SOFTWARE\WOW6432Node\Crypto Pro` (only `OCSPAPI`/`TSPAPI`,
`cpoids1`, `pkimgmt.ru`). So `AppPath` is **not** how the path is found. Registry
hypothesis rejected (owner was right).

**Static analysis of the bundle (this chunk) — the real chain:**
- `nmcades.exe` (our host, launched by Chrome via our HKCU manifest) loads
  **`npcades.dll`** (string `npcades.dll` is in `nmcades.exe`); the search order
  finds *our* `npcades.dll` sitting next to it. ✓ (matches: renaming the system
  `nmcades.exe` didn't break anything.)
- `npcades.dll` imports `SHGetFolderPathW` + `GetEnvironmentVariableW` and holds
  the **relative** suffix strings `Mini CSP\capi20.dll`, `CAdES Browser Plug-in`,
  `Crypto Pro`, plus the log line `LoadLibraryExA(capi20.dll) failed.`. It builds
  `<base>\Crypto Pro\CAdES Browser Plug-in\Mini CSP\capi20.dll` and `LoadLibraryEx`s
  it. `<base>` comes from `SHGetFolderPath` (Program Files), **not** from
  `GetModuleFileName(npcades)`. (matches: renaming the *system* `Mini CSP` broke
  it — the provider loads from `…\Program Files (x86)\Crypto Pro\CAdES Browser
  Plug-in\Mini CSP`, ignoring our co-located copy.)
- `mydss.dll` lives in the plug-in root (same `<base>`), which is why the clean
  machine first fails on `mydss.dll installation path` and reports `0.0.0000`:
  `<base>` (`…\Program Files (x86)\Crypto Pro\CAdES Browser Plug-in`) does not
  exist there at all.

**Conclusion:** the provider/module path is effectively hardcoded as
`SHGetFolderPath(Program Files[ x86]) + \Crypto Pro\CAdES Browser Plug-in\…`,
resolved inside `npcades.dll`, independent of where our files sit. `cades.dll`
and `npcades.dll` *do* read `…\CurrentVersion\AppPath`, but only (apparently) as an
optional override that nobody sets — the working machine has no such key yet works,
so the `SHGetFolderPath` path is the live mechanism.

**Options (no rebuild) to test, in order:**
1. **(still worth a 1-min test, despite "no registry")** create
   `HKLM\SOFTWARE\WOW6432Node\Crypto Pro\Cryptography\CurrentVersion` with
   `AppPath = <our extracted …\CAdES Browser Plug-in>` and see if the provider
   loads from our dir. "Key absent" ≠ "override won't work"; the binaries read it.
   If it works → relocatable via one HKLM value (admin once, but path is ours, not
   Program Files).
2. **Junction/copy into Program Files (x86)** — MVP unblock: create
   `C:\Program Files (x86)\Crypto Pro\CAdES Browser Plug-in` (or a junction to our
   dir). Needs admin once; collides with a real CryptoPro install; not portable.
3. **CryptoPro change (best for product, owner has a contact):** make `npcades.dll`
   resolve `Mini CSP\…` / `mydss.dll` relative to its own module
   (`GetModuleFileName(hNpcades)` — already imported) instead of
   `SHGetFolderPath(Program Files)+Crypto Pro\CAdES Browser Plug-in`, or honor an
   env var / HKCU override. This is the only path to true admin-free portability.

**Next:** owner to (a) try option 1 (set WOW6432Node AppPath → our dir) as a quick
check, and/or (b) raise option 3 with CryptoPro. ProcMon would confirm the exact
`SHGetFolderPath` CSIDL + whether AppPath is consulted first, if needed.

---

## 2026-06-02 — Owner bisect: Mini CSP loads from the ABSOLUTE system path

**Context:** owner ran our launcher on a machine where the CryptoPro plug-in was
already installed via MSI with `ADDMINICSP=1`. It worked (version now reads
`2.0.15000`, not `0.0.0000`). Then a two-step rename test:
1. Renamed `nmcades.exe` inside the *system* install
   (`C:\Program Files (x86)\Crypto Pro\CAdES Browser Plug-in\`) → **still worked**
   ⇒ Chrome launches *our* `nmcades.exe` (via our HKCU native-messaging manifest),
   not the system one. Good.
2. Renamed the *system* `Mini CSP` folder → **broke immediately** ⇒ the provider
   (Mini CSP / `cpcspi.dll`) is loaded from the absolute system path
   `…\Program Files (x86)\Crypto Pro\CAdES Browser Plug-in\Mini CSP`, **not** from
   our extracted bundle next to our `nmcades.exe`.

**Interpretation:** the earlier `mydss.dll` / `0.0.0000` failure on a clean machine
and this bisect agree: the plug-in resolves its provider/module path from an
ABSOLUTE location (registry `AppPath` and/or a hardcoded
`Program Files (x86)\Crypto Pro\CAdES Browser Plug-in`), independent of where our
`nmcades.exe` actually sits. Our portable extraction never points it at our dir.

**Still-open branch (decides the fix):** is that absolute path
(a) read from the registry key
`HKLM\SOFTWARE\WOW6432Node\Crypto Pro\Cryptography\CurrentVersion\AppPath`
(32-bit view) — in which case we override it (ideally HKCU, no rebuild, maybe no
admin) to point at our extracted dir; or
(b) hardcoded in `cpcspi.dll`/`cades.dll` as a literal
`C:\Program Files (x86)\…` — in which case we must place Mini CSP there (junction
or admin copy) or get a relocatable build from CryptoPro.

**Next (decisive test):** ProcMon on `nmcades.exe` (+children), filter
`RegQueryValue`/`CreateFile`, watch what precedes the `…\Mini CSP\cpcspi.dll`
load — a `RegQueryValue` on `…\CurrentVersion\AppPath` (or on
`HKLM\…\Microsoft\Cryptography\Defaults\Provider\…\Image Path`) means registry-
driven (good, override it); a bare `CreateFile` on the literal Program Files path
with no preceding reg read means hardcoded. Quick pre-check on the working machine:
`reg query "HKLM\SOFTWARE\WOW6432Node\Crypto Pro\Cryptography\CurrentVersion" /v AppPath`.

---

## 2026-06-02 — Diagnose "mydss.dll installation path" / provider-not-loaded on real run

**Context:** with the launcher now reaching Chrome (after `9f08cb5` made the MV2
policy write non-fatal), a real run on a clean machine shows: extension loaded ✓,
plugin loaded ✓, but **"Версия плагина: 0.0.0000"**, **"Объекты плагина: ожидание
загрузки провайдера"**, and an error dialog **"Error occured while trying to get
mydss.dll installation path. Maybe CryptoPro Browser plug-in was not correctly
installed."** Provider never loads.

**Investigation (this chunk):**
- Re-read `9f08cb5`: it only changes the MV2 policy (registry write → fall back to
  Chrome `--enable/--disable-features` flags) and the download timeout. It touches
  neither the plugin nor any CryptoPro registry key. The extension clearly loaded,
  so MV2 is active. **Conclusion: that fix is not the cause** — it merely let the
  launcher progress far enough to surface the next, pre-existing problem.
- Downloaded and unpacked the pinned `2.0.15000` bundle. `mydss.dll` (32-bit) IS
  present next to `nmcades.exe`/`cades.dll`; the 32-bit `Program Files` tree is
  fully self-contained (nmcades + cades + mydss + a complete `Mini CSP` with
  `rutoken.dll`, `cpcspi.dll`, `config.ini`). Native host is PE32 (x86); it loads
  the matching 32-bit DLLs — architecture is consistent.
- The error string lives in `cades.dll`/`npcades.dll`. Their registry strings
  include `SOFTWARE\Crypto Pro\Cryptography\CurrentVersion` + `…\AppPath`,
  `Software\Crypto Pro\CAdES`, `Software\Crypto Pro\CAdESplugin`. Our launcher
  writes only two HKCU keys: the native-messaging host and the MV2 policy. It never
  records where the plug-in is "installed".

**Hypothesis (needs verification — do NOT treat as fact):** on a clean machine the
plug-in cannot resolve its own install root, so `cades.dll` fails to locate
`mydss.dll` and the GOST provider → version `0.0.0000`, provider never loads. A
normal MSI install writes `…\Cryptography\CurrentVersion\AppPath`; our portable
extraction does not. NOTE: an earlier registry-centric conclusion was wrong once
before, so verify on the known-good machine first.

**Next:**
- Verify on the machine where `2.0.15000` worked manually:
  `reg query "HKCU\Software\Crypto Pro\Cryptography\CurrentVersion" /v AppPath` and
  the same under `HKLM`. If present and pointing at a plug-in dir → confirms the
  hypothesis.
- Candidate fix (no admin): after extracting the plug-in, write
  `HKCU\Software\Crypto Pro\Cryptography\CurrentVersion\AppPath` (and any sibling
  key the plug-in needs) pointing at our extracted
  `…/cryptopro/plugin/.../Program Files/Crypto Pro/CAdES Browser Plug-in`, gated by
  a state file like the other registrations. Implement only after the reg-query
  check (or after confirming the exact value via ProcMon on the local Windows box).

---

## 2026-06-02 — Fix first-run launcher failures after the legacy MV2 repin

**Context:** owner tested the latest remote and embedded launchers after the
legacy MV2 stack and internal-csp payload repin. Remote first-run repeatedly
failed after exactly five minutes while reading the 173 MB `payload.zip`:
`PAYLOAD_DOWNLOAD_FAILED: context deadline exceeded (Client.Timeout or context
cancellation while reading body)`. Embedded first-run prepared the payload and
CryptoPro bundle, detected the MV2 extension, then failed before Chromium launch:
`apply chrome manifest v2 policy: set chrome policy ExtensionManifestV2Availability:
ERROR: Access is denied.`

**Planned:**
- make remote payload downloads tolerant of slow first-run connections without
  weakening HTTPS/SHA-256/size verification;
- keep the MV2 compatibility setup reversible, but do not abort the whole
  launcher when the per-user Chrome policy registry write is denied;
- add focused tests and update the docs/changelog so the next agent can see
  exactly why this was changed.

**Done:** increased the default remote download client timeout from 5 minutes to
30 minutes; this addresses the exact `Client.Timeout ... while reading body`
failure without changing the HTTPS requirement, pinned expected size, early
oversize abort, or SHA-256 verification. Changed MV2 policy handling so a denied
`HKCU\Software\Policies\Google\Chrome\ExtensionManifestV2Availability` write is
logged as a compatibility warning instead of returning a fatal launcher error.
When that write fails, the launcher adds Chrome 138 MV2 fallback flags before
the app URL:
`--enable-features=AllowLegacyMV2Extensions` and
`--disable-features=ExtensionManifestV2Unsupported,ExtensionManifestV2Disabled`.
Added focused tests for the longer first-run download window and the policy
failure fallback path. Verified the pinned remote payload URL is live: full HTTPS
download returned `200`, size `173037165`, SHA-256
`9b2a00bb8ba09f59f973691c9a26cdb0bd757795f75533ff3bb971cb83501c48`.

**Verification:** local Go is unavailable on Watson's host, so local verification
was limited to `git diff --check` and the full HTTPS payload download/SHA check
above. Pushed commit `9f08cb5`; GitHub Actions `build-windows` run
`26832228871` passed. Windows `go test` passed for `internal/bootstrap`; embedded
launcher artifact `kriptosfera-windows-embedded` was uploaded as artifact
`7363256347` (199552391 bytes), and remote launcher artifact
`kriptosfera-windows-remote` was uploaded as artifact `7363257541` (26666775
bytes).

**Next:** owner should retry both launchers on Windows using run `26832228871`.
Expected behavior: remote first-run is no longer killed at 5 minutes; embedded
startup no longer aborts on denied `ExtensionManifestV2Availability` registry
write. If Chrome still does not load the MV2 extension when the registry write is
denied, the next investigation is Chrome's actual `chrome://policy` /
command-line state on that machine.

---

## 2026-06-02 — Re-pin the remote payload to the internal-csp build

**Context:** the previous chunk changed `app-config.json` `startUrl` to the
internal-csp page and pushed it (`23a3b8c`). CI went green: `build-windows`
(run 26815886534) and `build-payload` (run 26815886446) both succeeded. The
**embedded** launcher bakes the config from the commit, so it already opens the
internal-csp page. The **remote** launcher, however, downloads the payload pinned
by `build/payload-lock.json`, which still pointed at the *old* payload
(`9575882…`, official-demo startUrl). So a remote launcher would still open the
old page.

**Planned:** re-pin `build/payload-lock.json` to the payload that `build-payload`
just published from `23a3b8c`
(SHA `9b2a00bb8ba09f59f973691c9a26cdb0bd757795f75533ff3bb971cb83501c48`,
size `173037165`).

**Done:** verified the new payload is live on the server (HTTP 200, content-length
and `payload.json` SHA/size match the build log), then updated `payload-lock.json`
`sha256`, `size`, `url`, and `metadataUrl` to the new artifact. Updated CHANGELOG.
Embedded and remote launchers now both open the internal-csp page.

**Next:** same as below — E2E on a clean Windows machine with a Rutoken. The
remote launcher can now be used for that check too (not just embedded).

---

## 2026-06-02 — Point launcher startUrl at the internal-csp test page

**Context:** the legacy MV2 stack is integrated and CI-green (plug-in `2.0.15000`
+ MV2 extension `1.2.13` + Chrome 138 + `ExtensionManifestV2Availability=2` policy;
verified the 2.0.15000 bundle on the server matches the lock SHA/size and contains
all required files). The launcher's `startUrl` still pointed at the official
CryptoPro demo page, which does **not** set `EnableInternalCSP`, so a run there
would not exercise the bundled Mini CSP. The owner confirmed the normal
`internal-csp` page works.

**Planned:** set `payload-template/config/app-config.json` `startUrl` to
`https://mescheryakov.pro/kriptosfera/cryptopro-cades-test/internal-csp/demopage/cades_bes_sample.html`
and set `allowedOrigins` to `https://mescheryakov.pro` (startUrl must be inside
allowedOrigins per `validateAppConfig`).

**Done:** updated `app-config.json` (startUrl + allowedOrigins); kept
`windowMode: app` (standalone app window) and diagnostics off. Updated the plan
doc and CHANGELOG.

**Next:**
- E2E on a clean Windows machine: download the embedded launcher from the latest
  `build-windows` run, insert a Rutoken with a CryptoPro-format GOST cert, and
  confirm certificate enumeration + `SignCades` on the internal-csp page.
- If it works, this closes MVP stage 6 (clean-machine signing). Then consider
  branding/customer startUrl as a config, and track the future MV3/latest-Chromium
  migration (see `docs/cryptopro-csp-lite-plan.md` → Future goals).
- Blocked/needs owner: nothing for this step.
