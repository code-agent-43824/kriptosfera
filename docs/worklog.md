# Worklog — handoff log between agents

Several agents work on this repo, sometimes in parallel. Keep a short entry per
chunk of work: **Planned / Done / Next**, newest on top. Document first, then code.
For deeper context see `docs/cryptopro-csp-lite-plan.md` and `CHANGELOG.md`.

---

## 2026-06-04 — Shorten CryptoPro AppData layout while keeping vendor path

**Context:** owner asked to reduce the deep AppData nesting, but keep the visible
CryptoPro-style folder names because the MSI install path on Windows is normally
`C:\Program Files (x86)\Crypto Pro\CAdES Browser Plug-in\`.

**Planned:**
- keep the per-app root unchanged: `%LOCALAPPDATA%\Kriptosfera\apps\demo\0.5.0`;
- extract the plug-in archive's `Program Files\Crypto Pro\...` subtree directly
  into `<appDir>\Crypto Pro\...`;
- make the resulting Mini CSP path:
  `%LOCALAPPDATA%\Kriptosfera\apps\demo\0.5.0\Crypto Pro\CAdES Browser Plug-in\Mini CSP`;
- bump the CryptoPro plugin layout version so old deep extractions are not
  reused;
- update native messaging lookup, diagnostics expected files, tests, and docs.

**Done:**
- added zip-entry mapping for the CryptoPro bundle so only the archive's
  `Program Files\Crypto Pro\...` subtree is extracted;
- changed the plugin root from `<appDir>\cryptopro\plugin` to
  `<appDir>\Crypto Pro`;
- bumped the CryptoPro plugin layout version from `2` to `3`;
- updated native messaging lookup and required-file validation to the shorter
  `CAdES Browser Plug-in\...` suffixes;
- added cleanup for the legacy `<appDir>\cryptopro` extraction after a successful
  layout-v3 extraction;
- updated tests and docs for the new path.

**Verification:**
- `git diff --check` passed locally;
- local `go test ./internal/bootstrap` and `gofmt` could not run because this
  Linux host does not have `go`/`gofmt` installed.
- GitHub Actions `build-windows` run `26940693588` passed on commit `ac36a79`:
  payload packaging, embedded launcher build, and remote launcher build all
  completed successfully;
- CI `go test` passed for `github.com/code-agent-43824/kriptosfera/internal/bootstrap`
  (`0.692s`);
- CI logs confirmed both launcher variants fetched the pinned CryptoPro bundle
  `2.0.15000` with SHA-256
  `4590391e35c251cd4685d839ab62fad69e08716335931ac1c1b753b0cd346c6a`;
- downloaded artifacts contain `KriptosferaDemo.exe` (`203556352` bytes) and
  `KriptosferaDemo-remote.exe` (`30519296` bytes).

**Next:** validate the new AppData path on Windows. The expected Mini CSP path is
`%LOCALAPPDATA%\Kriptosfera\apps\demo\0.5.0\Crypto Pro\CAdES Browser Plug-in\Mini CSP`.

---

## 2026-06-04 — Static bundle recovery and payload slimming plan

**Context:** owner approved a repo cleanup pass while waiting for the CryptoPro
vendor fix. Before changing code or assets, Watson pulled the latest `origin/main`
and reviewed the 14 intervening doc-only commits. The important new repo state is
that the portable/no-MSI blocker is now consolidated in
`docs/cryptopro-portable-plugin-findings.md`: the current path forward is to wait
for a fixed CryptoPro plug-in build rather than keep byte-patching
`npcades.dll`/`cades.dll`.

**Planned:**
- restore the live static-storage invariant for the pinned legacy plug-in:
  `build/cryptopro-plugin-lock.json` points at immutable `2.0.15000` URL
  `4590391e.../cryptopro-plugin.zip`, but the live server currently returns 404;
- inventory the restored CryptoPro `2.0.15000` bundle before any pruning and mark
  what is runtime-required versus cleanup-risky;
- document a small, reversible payload slimming approach. The current remote
  `payload.zip` is 173,037,165 bytes and its contents are almost entirely Chromium
  (`~389.7 MB` raw / `~172.9 MB` compressed); the CryptoPro plug-in is not inside
  that payload and is embedded into launcher variants as a separate verified
  archive;
- defer the actual payload rebuild, payload-lock update, GitHub Actions log
  review, and Windows smoke/E2E checks to a later chunk per owner instruction.

**Initial findings:** the live remote payload URL still verifies against
`build/payload-lock.json` (size `173037165`, SHA-256
`9b2a00bb8ba09f59f973691c9a26cdb0bd757795f75533ff3bb971cb83501c48`). The pinned
legacy CryptoPro plug-in URL and metadata URL currently return 404 on
`mescheryakov.pro`, while the server still has the older `2.0.15700` bundle that
the docs now mark as broken for Mini CSP.

**Done:** restored the immutable `2.0.15000` static bundle from the local
workspace recovery directory to:

```text
https://mescheryakov.pro/kriptosfera/cryptopro/plugin/2.0.15000/4590391e35c251cd4685d839ab62fad69e08716335931ac1c1b753b0cd346c6a/
```

Also restored the legacy installer and MV2 CRX source mirrors under:

```text
https://mescheryakov.pro/kriptosfera/cryptopro/special/legacy-cades-2.0.1500-mv2/
```

Public re-download verification:

```text
cryptopro-plugin.zip        size 24052329  sha256 4590391e35c251cd4685d839ab62fad69e08716335931ac1c1b753b0cd346c6a
cryptopro-plugin.json       HTTP 200       metadata sha256/size match lock
cadesplugin_2_0_1500.exe    size 11781256  sha256 7c43d41482684ff3d98fe45c741c6a14b63055c88721f0207ab2b605dbc28cb2
extension_1.2.13.crx        size 70909     sha256 cf9bd5ce31d8ae6e50038dc742b4fd900a87c854cccb5db69a39976cccbf07c9
```

Updated `docs/cryptopro-plugin-inventory.md` with the restored `2.0.15000`
inventory and pruning notes. Added `docs/payload-slimming-plan.md`: first safe
payload pass should target locales, hyphen-data, `setup.exe`, and helper EXEs
only after smoke testing; GPU/Vulkan/SwiftShader and core Chrome files stay out
of the first pass.

**Verification:** `git diff --check` passed. Commit `5ecdd55` was pushed and
GitHub Actions `build-windows` run `26939377762` passed. The run fetched the
restored `2.0.15000` CryptoPro bundle twice (embedded and remote launcher builds)
with SHA-256 `4590391e35c251cd4685d839ab62fad69e08716335931ac1c1b753b0cd346c6a`.
Artifacts uploaded successfully: embedded artifact `7406303451` (199,552,437
bytes) and remote artifact `7406303998` (26,666,817 bytes). The workflow also
rebuilt a same-size payload from the current commit (`173037165` bytes) for the
embedded launcher, but no new payload was published and `build/payload-lock.json`
was not changed.

**Next:** implement the actual Chromium slimming build step in a later chunk,
then rebuild/publish a new immutable payload, update `build/payload-lock.json`,
and inspect GitHub Actions logs. Do not prune the CryptoPro plug-in bundle until
a fixed vendor build exists and can be checked on Windows with clean-machine and
MSI-installed controls.

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

**Update 2:** header-only ASLR patch failed even with system ASLR fully disabled
(owner set Exploit Protection: Mandatory off, Bottom-up off; `Get-Process nmcades`
returned nothing — host too short-lived to catch). Conclusion: `0x10000000` is
occupied in the `nmcades.exe` process (or the address simply isn't npcades' base),
so `GetModuleFileName(0x10000000)` still misses. **Better patch (ASLR-independent):**
at the 3 sites (`0x10004069`, `0x10054cf2` builds `Mini CSP\capi20.dll`,
`0x10056637`) change `push 0x10000000` → `push 0` (one byte each: imm high byte
`0x10→0x00`). `GetModuleFileName(NULL)` returns the **main exe path = `nmcades.exe`**,
which sits in OUR dir next to `Mini CSP`/`mydss.dll`, so the module-relative build
resolves into our bundle regardless of load address/ASLR. Header restored to
original `0x140`; only 3 code bytes + checksum changed. Patched sha256 `9816aef0…`
(orig `0f7ffc9a…`). Owner can revert the system ASLR changes — not needed anymore.

**Vendor report (final):** "npcades.dll passes a hardcoded `0x10000000` as `hModule`
to `GetModuleFileNameA/W` in 3 places; under ASLR (or whenever the DLL isn't at its
preferred base) this returns the wrong/empty path and the plug-in falls back to
`%ProgramFiles%\Crypto Pro\CAdES Browser Plug-in`. Pass the module's real `HINSTANCE`
(from `DllMain`) or `NULL` (to use the host exe's dir) so a side-by-side/portable
deployment next to `nmcades.exe` works without a system install."

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
  `…/Crypto Pro/CAdES Browser Plug-in`, gated by
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

---

## 2026-06-03 — Dump: nmcades is NOT hung (idle on ReadFile); minimal single-site patch

**Minidump analysis (nmcades.exe, clean machine, patched npcades):** 1 thread,
parked in `ntdll/KERNELBASE` Wait inside its own frames. nmcades imports for I/O are
ONLY `ReadFile/WriteFile/GetStdHandle` — no `WaitForSingleObject/Event/CreateThread/
GetMessage`. So the "Wait" is `ReadFile(stdin)`, handle `0x74` = the Chrome native-
messaging pipe. **nmcades isn't hung — it's idly waiting for the next message.** The
"plugin load timeout" is therefore a handshake/response problem, not a hang or a
window.

**The 3 `push 0x10000000` sites in npcades.dll decoded:**
- `0x4069` — generic helper "module dir + suffix" (used for paths incl. mydss).
- `0x54cf2` — builds `Mini CSP\capi20.dll` (the provider path). THE one that matters.
- `0x56637` — `GetModuleFileNameW` + opens **HKLM** registry (`0x80000002`,
  `RegOpenKeyEx`) — trace/settings, unrelated to provider resolution.

Patching all 3 (NULL) cleared the mydss error but regressed the handshake ("Плагин
загружен" → "истекло время загрузки плагина"), likely because the trace/registry
site (or the helper) is on the init path. **New minimal patch: only `0x54cf2`**
(file off `0x540f6`: `0x10→0x00`), header untouched (`0x140`), checksum recomputed.
orig sha `0f7ffc9a…`, minimal-patch sha `db30e180…`.

**Most useful experiment (owner, CLEAN machine, no plugin):** drop the minimal
npcades into our extracted dir, run the launcher, read the page. Hypothesis: handshake
restored ("Плагин загружен"), and the provider now resolves from our `Mini CSP`. A
residual mydss popup may reappear (it did in the very first screenshot) but did not
block "plugin loaded". Clean machine is the only meaningful target — on a machine with
the plugin installed the provider comes from Program Files and tells us nothing.

---

## 2026-06-03 — Dump #2 memory: native-messaging dialog recovered; cades.dll has the SAME hardcoded-base bug

**New data mined from the 2nd minidump's memory (huge):** the native-messaging
conversation is in process memory:
- `cadesplugin.EnableInternalCSP` → **result `true`** (internal-CSP mode engages!).
- next request: `{"method":"CreateObject","params":[{"value":"CAdESCOM.About"}],"requestid":3}`.
So the extension↔plugin protocol works far in: internal CSP ON, plugin alive and
responding, parked in `ReadFile(stdin)` waiting for the next message (identical stack
to dump #1). `capi20/cpcspi` (the CSP) still never load — we stall around About/version
and provider load, not earlier.

**"Версия плагина: 0.0.0000" explained:** `cades.dll` ALSO has the hardcoded
`push 0x10000000` → `GetModuleFileName` bug — 2 sites (rva `0x252a` GetModuleFileNameA
helper, `0x4f09a` GetModuleFileNameW+registry). ASLR on (`DllChar 0x140`). So
`cades.dll` can't locate itself → `About.PluginVersion` reads `0.0.0000`. We only
patched `npcades` before. No self-integrity strings in `cades.dll` (safe to patch).

**Patch built:** `cades.dll` rva `0x252a` only (the path helper; left the registry
site `0x4f09a` alone, per the npcades lesson). file off `0x192e`: `0x10→0x00`,
checksum recomputed. orig sha `c108c5d5…`, patched `653efb0f…`. Use together with the
minimal `npcades.dll` (`db30e180…`).

**Takeaway:** this is the same single bug (hardcoded preferred ImageBase passed to
GetModuleFileName) replicated across modules; we're peeling it module by module
(npcades → cades → likely the CSP-load path next). Confirms the vendor fix (resolve
relative to the real HINSTANCE everywhere) is the clean global solution.

---

## 2026-06-03 — Dump #3 + page: NULL byte-patch is a dead end (breaks handshake); revert to originals

**Page with cades.dll patch (rva 0x252a → NULL):** NO new regression (owner
correction) — the version stopped showing already when the npcades patch turned the
mydss error into a hang; cades patch left that unchanged. "Истекло время ожидания
загрузки плагина", no version. Dump #3: 4 threads
(cades COM/thread-pool init kicked in — real change), main thread still in
`ReadFile(stdin)`, `capi20`/CSP still not loaded; protocol memory unchanged
(EnableInternalCSP=true → CreateObject CAdESCOM.About, no further).

**Why every `push 0x10000000 → push 0` (NULL) patch hurts:** `GetModuleFileName(NULL)`
returns the **nmcades.exe** path. But the plugin version is `cades.dll`'s version,
which it reads via `GetFileVersionInfo` on the path **to cades.dll**; and the provider
path needs `…\Mini CSP\…`. Different sites want paths to DIFFERENT modules; a single
NULL gives them all the exe path, so something always breaks. The version/handshake
path breaks → "истекло время загрузки плагина". `cades.dll` loads the CSP via
`LoadLibrary` + `GetModuleHandleA("capi20.dll")` (string VA 0x10206e9c, refs=2),
choosing capi10/capi20 — internal-CSP, no registry. No `CryptAcquireContext` import.

**Conclusion:** byte-patching `→NULL` cannot build a clean PoC — each patch fixes one
path and breaks another. The correct fix is `GetModuleHandle("<module>.dll")` per site
(real HINSTANCE), which is not a one-byte edit. Best reachable state is the ORIGINAL
DLLs: plugin "loaded", version 0.0.0000, stuck at "ожидание провайдера". Reverted
owner to original `cades.dll` (sha c108c5d5) + `npcades_orig.dll`. Remaining path =
the vendor fix (per-module HINSTANCE everywhere), which owner is already awaiting.

**Sites catalogued for the record:** npcades.dll push 0x10000000 → GetModuleFileName
at rva 0x4069 (path helper), 0x54cf2 (Mini CSP\capi20.dll), 0x56637 (GMFW+registry);
cades.dll at rva 0x252a (path helper), 0x4f09a (GMFW+registry). All ASLR-on (0x140),
no self-integrity strings in npcades/cades. capi20/cpcspi (certified CSP) NOT patched.

---

## 2026-06-03 — BREAKTHROUGH: nmcades is NOT hung — it responds (Go probe via stdin/stdout)

Couldn't use Frida; instead built a tiny Go probe (`nmprobe.exe`, GOOS=windows) that
launches `nmcades.exe`, writes a length-prefixed native-messaging message and actively
reads stdout (a plain `> file` redirect just buffered and looked empty/hung). Result —
host **answered**:
`{"data":{"message":"Can't find object by id","requestid":1,"type":"error"},"tabid":"…"}`.

**This disproves the "host hangs" theory.** nmcades reads, processes, replies with a
proper protocol error, and flushes — the ReadFile idle in the dumps is just normal
between-message waiting. The `Can't find object by id` (objid=0) is because a single
CreateObject without the prior handshake has no registered root object. Built
`nmprobe2.exe` to replay the real dump sequence (EnableInternalCSP → CreateObject
CAdESCOM.About → GetProperty PluginVersion) and reveal the response side of the full
dialog. Awaiting owner run. Probes live only on owner's box / /tmp; not committed.

---

## 2026-06-03 — Pause portable investigation; consolidate findings in repo

Owner: stop experiments, await CryptoPro fix. Probe v2 on the **working-install**
machine returned the **same** `Can't find object by id` as on the clean machine —
so that error is a limitation of our replayed handshake (objid=0 root object needs
the full browser sequence), not a machine difference. Fully reproducing the
browser handshake needs the real `cadesplugin_api.js` protocol (demo page loads it
as `../cadesplugin_api.js`).

**Consolidated everything important into the repo:**
- New `docs/cryptopro-portable-plugin-findings.md` — standalone summary: root cause
  (`GetModuleFileName(0x10000000)` hardcoded base in npcades/cades under ASLR →
  Program Files fallback), reproduction, all dead ends (AppPath/junction/ASLR/byte
  patch), the Go probe proof that the host responds (not hung), and the vendor
  report with affected RVAs.
- `docs/cryptopro-csp-lite-plan.md` — added the "Clean-machine (portable, no-MSI)
  blocker" section pointing at the findings doc.
- `docs/README.md` — linked the findings doc.

**Not committed (by design):** the diagnostic `npcades`/`cades` byte-patches and the
Go probes lived on the owner's box / in `/tmp` only — no CryptoPro binaries in Git.

**Next (when vendor build lands):** re-pin plug-in in `build/cryptopro-plugin-lock.json`,
re-run launcher on a clean machine, expect provider load + cert enumeration +
`SignCades` with a Rutoken; then track the MV3 / latest-Chromium migration.
