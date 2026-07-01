# Worklog — handoff log between agents

Several agents work on this repo, sometimes in parallel. Keep a short entry per
chunk of work: **Planned / Done / Next**, newest on top. Document first, then code.
For deeper context see `docs/cryptopro-csp-lite-plan.md` and `CHANGELOG.md`.

---

## 2026-07-01 - Native x64 Rutoken FKC/PKCS#11 experiment started

**Planned:**
- run the native x64 retest from `docs/handoff-rutoken-fkc-diagnostic-runbook.md`
  and `docs/pkcs11-active-investigation-2026-06.md`;
- verify the current Program Files Mini CSP state before changing anything;
- apply only the minimal elevated Program Files changes needed for the experiment
  (backup first, no vendor binaries committed);
- launch the Kriptosfera test app, enumerate Rutoken containers, and capture
  module evidence for FKC and PKCS#11;
- record the result here for the next agent.

**Initial context:**
- host is native AMD64/x64 (`AMD Ryzen 7 5800X3D`), not the earlier ARM/Parallels
  environment;
- current shell is non-elevated, so Program Files writes need an elevated step;
- `C:\Tools\listdlls.exe` is not present; use PowerShell process module snapshots
  unless ListDLLs is installed later;
- staged DLL sources are expected under the owner's Desktop `cryptopro csp`
  extraction.

**Done:**
- confirmed installed Mini CSP core is now `cpcspi.dll` ProductVersion
  `5.0.13800.0` on native AMD64. This removes the earlier ARM session's
  `5.0.13000` core-version mismatch as an explanation;
- before setup, Program Files Mini CSP already had `cpfkc.dll` and FKC carrier
  config, but lacked `cryptoki.dll`, `rtPKCS11ECP.dll`, and the
  `cryptoki_rutoken` KeyDevice section;
- ran an elevated setup:
  - backed up Program Files `config.ini` to
    `config.ini.native-x64-20260701-091021.bak`;
  - copied x86 `cryptoki.dll` ProductVersion `5.0.13800.0` and x86
    `rtPKCS11ECP.dll` ProductVersion `2.15.1.0` into both Program Files Mini CSP
    and Program Files `CAdES Browser Plug-in`;
  - added `[apppath]` mappings and `[KeyDevices\cryptoki_rutoken]` to Program
    Files `config.ini` as CP1251;
- launched the local Desktop `kriptosfera-windows-embedded\KriptosferaDemo.exe`.
  This is an older layout-v2 embedded launcher, so it extracts the plugin under
  `%LOCALAPPDATA%\Kriptosfera\apps\demo\0.5.0\cryptopro\plugin\...`;
- duplicated the same `cryptoki.dll`/`rtPKCS11ECP.dll` placement and config
  fragment into that actual AppData runtime plugin folder;
- attempted the extra canonical reader-instance step:
  `cpconfig -hardware reader -add cryptoki_rutoken -connect "PNP cryptoki" -name "Rutoken PKCS11"`.
  `cpconfig` printed `Adding new reader`, but `cpconfig -hardware reader -view`
  still listed only `Aktiv Rutoken ECP 0` / `All PC/SC readers`, and no
  cryptoki reader appeared.

**Evidence:**
- The demo page did enumerate certificates through the native host. Chromium's
  `chrome_debug.log` shows `nmcades_plugin_api.js` requests and successful native
  responses for certificate objects/thumbprints, including subjects
  `CN=Test Certificate` and `CN=mytest_csp`.
- A normal 64-bit PowerShell module snapshot showed only `nmcades.exe`; this was
  a false negative for the 32-bit host. Re-taking the snapshot from 32-bit
  PowerShell (`SysWOW64\WindowsPowerShell`) showed the real loaded modules.
- The 32-bit module snapshot after enumeration loaded:
  - Program Files Mini CSP: `cpcspi.dll`, `capi10.dll`, `cpfkc.dll`, `pcsc.dll`,
    `rutoken.dll`, `jacarta.dll`, `safenet.dll`, `cpsuprt.dll`;
  - AppData plugin/runtime DLLs: `nmcades.exe`, `npcades.dll`, `cades.dll`,
    `cplib.dll`, `asn1*`, `capi20.dll`, `cpcspi.dll`, and support DLLs.
- **Not loaded:** `cryptoki.dll` and `rtPKCS11ECP.dll`, despite being present in
  both the authoritative Program Files plugin dirs and the AppData runtime plugin
  dirs, with `[apppath]`, `KeyDevices\cryptoki_rutoken`, and the attempted
  `cpconfig reader -add`.

**Result / verdict:**
- FKC path is active: `cpfkc.dll` loads on native x64, same as the prior run.
- PKCS#11-active still does not activate on native x64, even with Mini CSP core
  `5.0.13800.0` and matching `cryptoki.dll` `5.0.13800.0`. The earlier ARM-only
  caveat is now mostly retired for this failure mode: `cryptoki.dll` is still not
  loaded on native AMD64.
- Current best verdict: bundled/MSI Mini CSP does not instantiate the
  `cryptoki_rutoken` reader from this config/package set. FKC remains the working
  active-mode Rutoken path for the MVP.

**Next:**
- if PKCS#11-active is still required, the remaining path is a vendor answer or a
  known-good full-CSP Windows registry/export comparison from a machine where
  `cpconfig -hardware reader -view` actually shows a cryptoki reader.

---

## 2026-07-01 — Prep for Rutoken FKC/PKCS#11 runbook: DLL source verified (Phase 4 not started)

**Context:** picking up `docs/handoff-rutoken-fkc-diagnostic-runbook.md`. This box
is MSI-installed, so per the runbook's Phase 1 the authoritative Mini CSP is
`C:\Program Files (x86)\Crypto Pro\CAdES Browser Plug-in\Mini CSP\` (our
`%LOCALAPPDATA%` overlay is bypassed while the MSI CSP is present). Phases 1–2 were
already DONE (2026-06-24): the three reader DLLs are absent from disk; FKC carrier
config already correct; PKCS#11 device config absent → hypothesis A (missing DLL).

**Done (prep / passive checks only, no task edits yet):**
- Located the Phase-3 DLL source the owner staged: `Desktop\cryptopro csp\`
  (OneDrive Desktop, `C:\Users\mesch\OneDrive\Рабочий стол\cryptopro csp`). It is a
  full CryptoPro CSP extraction (`ProgramFiles` = x86, `ProgramFiles64` = x64,
  `system32` = x64, `syswow64` = x86).
- Verified the three **x86** reader DLLs the 32-bit `nmcades.exe` needs are present
  and correct bitness: `cpfkc.dll` (x86, `ProgramFiles\Crypto Pro\CSP\`),
  `cryptoki.dll` (x86, same), `rtpkcs11ecp.dll` (x86, `syswow64\`). 64-bit twins
  also present but unused. The x86 CSP dir has 67 files, so `cpfkc`'s sibling deps
  are on hand if the load needs them.
- Registry references also on the Desktop for Phase 3(c): `hklm.reg` (~281 MB),
  `registry.reg` (~296 MB) full exports, plus small `capi10.reg`/`capi20.reg`. The
  working-machine `Crypto Pro\Cryptography\CurrentVersion` (KeyDevices/KeyCarriers)
  subtree must be extracted from these during Phase 4b (too large to grep whole).
- Independent corroboration of the resolved `2.0.15700`-broken root cause: a live
  module probe of the system MSI `nmcades.exe` (build 2.0.15700) showed
  `Mini CSP\capi20.dll` + `asn1*`/`cpsuprt` + (in one host) `cpcspi.dll` DO load,
  yet `About.CSPName(75/80/81)` = `0x80090017`. So the flag reached native, the DLL
  search path resolved, and config.ini was read — the failure was the broken plug-in
  build, not our packaging/path. (Matches `cryptopro-csp-lite-plan.md`.)

**Environment / blockers to clear before Phase 4:**
- Shell is **non-elevated** (`admin: False`) even after restarting Claude as admin;
  the Bash/PowerShell tool runs unelevated. Writing `cpfkc.dll` into the Program
  Files Mini CSP needs an elevated shell — must be dictated to the owner or run in
  an elevated session.
- `C:\Tools\listdlls.exe` is **absent**. Substitute: `Get-Process nmcades | %{ $_.Modules }`
  under **32-bit** PowerShell (`SysWOW64\WindowsPowerShell`), which already worked
  this session, or install Sysinternals ListDLLs.
- Phase 4a needs the **Rutoken ЭЦП inserted in FKC mode** and enumeration triggered
  on the internal-csp demo page (Claude-in-Chrome connector is available to drive
  the page).

**Next — Phase 4a (pivotal, single variable), on owner's go:**
1. Back up `<PF>\Mini CSP\config.ini`.
2. Copy **only** x86 `cpfkc.dll` → `C:\Program Files (x86)\Crypto Pro\CAdES Browser
   Plug-in\Mini CSP\` (no config edit — FKC carrier already present).
3. Fully restart launcher/Chrome + `nmcades.exe`, enumerate with token in FKC mode,
   snapshot nmcades modules → does `cpfkc.dll` load and FKC enumerate?
4. Then Phase 4b: `cryptoki.dll` + `rtPKCS11ECP.dll` (beside `nmcades.exe`) + append
   `cryptoki_rutoken` device to `config.ini` (CP1251), derived from the Desktop
   registry export.

**Open questions:**
- Does the authoritative MSI Mini CSP honor `rutokenfkc` once `cpfkc.dll` exists
  (A → ship the DLL) or ignore the carrier section entirely (B → vendor bug report)?
- Confirm the `cryptoki_rutoken` PKCS#11 device config against the working machine's
  real registry export (Desktop `hklm.reg`/`registry.reg`), not just the
  Linux-adapted fragment.
- Whether `cpfkc.dll` binds but device-init fails (middle branch) — would need a
  ProcMon gold-trace compare; ProcMon still needs elevation on this box.
- Portable/no-MSI path stays blocked by the CryptoPro `GetModuleFileName(0x10000000)`
  bug (orthogonal; re-confirm FKC on the overlay path once the vendor fix lands).

## 2026-06-13 — Fix Rutoken PKCS#11 runtime DLL placement

**Context:** hardware review found that the first Rutoken overlay put
`rtPKCS11ECP.dll` only in `CAdES Browser Plug-in\Mini CSP\`. That is fine for
the harmless spare copy, but the PKCS#11 path loads it by bare name from
`cryptoki.dll`, so Windows searches the process directory first
(`nmcades.exe` in `CAdES Browser Plug-in\`).

**Done:**
- changed `build/fetch-cryptopro-plugin.ps1` to keep the existing Mini CSP copy
  of `rtPKCS11ECP.dll` and also write `CAdES Browser Plug-in\rtPKCS11ECP.dll`
  into the slim embedded archive;
- changed the runtime/CI required-file guard to require
  `CAdES Browser Plug-in\Mini CSP\cpfkc.dll`,
  `CAdES Browser Plug-in\Mini CSP\cryptoki.dll`, and
  `CAdES Browser Plug-in\rtPKCS11ECP.dll`;
- verified the pinned `2.0.15000` Mini CSP core and the current
  `cpfkc.dll`/`cryptoki.dll` reader DLLs from the same Windows+CAdES
  distribution with PE version resources: `cpcspi.dll`, `cpfkc.dll`, and
  `cryptoki.dll` all report ProductVersion `5.0.13000.0`. No separate
  `5.0.13001` reader payload was found in the pinned `2.0.15000` distribution,
  so `build/rutoken-fkc-lock.json` remains pinned to the verified same-source
  `5.0.13000` files instead of relabeling identical binaries.

**Verification:** local portable PowerShell rebuilt
`internal/bootstrap/cryptopro-plugin.zip`: 65 entries, size `13324371`,
SHA-256 `983efe16e23c169ff276945fd4d5bbe3d29f933d9592b013aecb90e247f0b544`.
The archive contains all expected overlay entries:
`Mini CSP\cpfkc.dll`, `Mini CSP\cryptoki.dll`, spare
`Mini CSP\rtPKCS11ECP.dll`, and process-dir
`CAdES Browser Plug-in\rtPKCS11ECP.dll`. The slim guard still finds no
`Program Files`, `Program Files 64`, `Common*`, or MSI entries.

Local Go checks passed with Go `1.24.2`:
`gofmt -l .`, `go vet ./...`, `go test ./...`,
`GOOS=windows GOARCH=amd64 go build ./...`, and
`GOOS=windows GOARCH=amd64 go build -tags remote ./...`.

**Next:** push, wait for `build-windows`, then perform the real Windows
hardware smoke with Rutoken ЭЦП in FKC and PKCS#11-active modes.

GitHub Actions `build-windows` run `27470583046` passed on commit `0a12d5d`.
CI logs show both embedded and remote launcher builds overlaid 3 files and
embedded a 65-entry slim archive (`12982910` bytes, SHA-256
`59ae53863b04d7b6e94598cb7c685c5d896ea8a75d405acae7461485b2476304`). Workflow
artifacts: embedded `7611930868` (`188600951` bytes) and remote `7611931175`
(`15714670` bytes).

---

## 2026-06-13 — Add Rutoken FKC / PKCS#11 Mini CSP overlay

**Context:** previous handoff `docs/handoff-rutoken-fkc-pkcs11-payload.md`
requested enabling active Rutoken ЭЦП modes in the embedded Mini CSP bundle:
FKC via CryptoPro `cpfkc.dll`, and PKCS#11-active via CryptoPro
`cryptoki.dll` plus Rutoken `rtPKCS11ECP.dll`. The overlay belongs in the
embedded CryptoPro Browser Plug-in archive, not in `payload.zip`, so
`build/payload-lock.json` does not change.

**Planned:**
- source the three 32-bit DLLs without committing binaries to Git;
- publish them under immutable project static URLs and pin SHA-256/size in a new
  lock file;
- extend `build/fetch-cryptopro-plugin.ps1` so it still verifies the full
  CryptoPro plug-in archive, then writes the slim archive with the three DLLs
  overlaid into `CAdES Browser Plug-in\Mini CSP\`;
- append any missing Rutoken FKC / PKCS#11 config fragment to Mini CSP
  `config.ini` as Windows-1251, preserving the existing config encoding;
- bump the CryptoPro plug-in layout version so old AppData extractions are not
  reused, and guard the three new DLLs as required runtime files.

**Done:**
- extracted `cpfkc.dll` and `cryptoki.dll` from the x86 tree of the public
  CryptoPro CSP 5.0 R3 Windows distribution with CAdES/plugin and partner
  PKCS#11 modules (`CryptoPro-5.0.13000.exe`, MD5
  `cce2be5fac6161f4fd53e46bea1af0b9`);
- downloaded x86 `rtpkcs11ecp.dll` from Rutoken PKCS#11Lib
  `1.4.02.0/Windows/x32/rtpkcs11ecp`;
- verified all three are PE32 Intel 80386 DLLs and published them to immutable
  project static storage:
  - `cpfkc.dll` — size `262448`, SHA-256
    `59e3609f1b2fcafe86d33d8387f6e2bedc861faa45c1dbbd5e4ca89be5ee05d8`;
  - `cryptoki.dll` — size `210304`, SHA-256
    `5f2c3742fa00cf0ec4c4fca0dcf81ffc39e798d86880bb977e5af9436d94fa6a`;
  - `rtPKCS11ECP.dll` — size `1593344`, SHA-256
    `6d61fbac6ebf4e7e71b4b2b968334dbc29b45183fe48c103ce1d9ebb07f089a0`;
- added `build/rutoken-fkc-lock.json` with HTTPS URLs, sizes, hashes, source
  notes, and deterministic zip timestamps;
- extended `build/fetch-cryptopro-plugin.ps1` to download and fail-closed verify
  the overlay DLLs, inject them into the slim archive, and append missing
  `rutokenfkc` / `rutokenfkc_nfc` / `cryptoki_rutoken` config entries to
  `Mini CSP\config.ini` using CP1251. The current `2.0.15000` config already
  contains `rutokenfkc` and `rutokenfkc_nfc`, so the local verification appended
  only the missing `cryptoki_rutoken` PKCS#11-active device;
- bumped `cryptoProPluginLayout` from `3` to `4`;
- added the three overlay DLLs to `requiredCryptoProPluginFiles` and updated
  tests/helpers accordingly.

**Verification:** public re-download of all three static DLL URLs matched the
lock sizes and SHA-256 values. Downloaded a local portable PowerShell and ran
`build/fetch-cryptopro-plugin.ps1` successfully: it verified the full
`2.0.15000` archive, overlaid the three DLLs, preserved CP1251 `config.ini`, and
produced a slim archive `12530145` bytes / 64 entries / SHA-256
`9185ded52ab41ecff674db367f7a69a52b816ab1edd6292786f93cba4517434e` with
`cpfkc.dll`, `cryptoki.dll`, `rtPKCS11ECP.dll`, and the
`cryptoki_rutoken` config stanza present. Local `go`/`gofmt` are unavailable on
Watson's Linux host, so Go tests must be verified by GitHub Actions after push.

GitHub Actions `build-windows` run `27469564250` passed on commit `99a589d`.
CI logs show both embedded and remote launcher builds verified the full
`2.0.15000` source archive, overlaid 3 Rutoken FKC/PKCS#11 files, and embedded a
64-entry slim archive (`12212184` bytes, SHA-256
`388c860708126f2989b90d69bec14a8f43e4d0817829d51746b1e0296fd8d898`). CI
`go test` passed for `github.com/code-agent-43824/kriptosfera/internal/bootstrap`
(`0.720s`). Workflow artifacts: embedded `7611623351` (`187831458` bytes) and
remote `7611623698` (`14945227` bytes).

**Next:** test the new launcher on Windows with a Rutoken ЭЦП token in FKC mode
and PKCS#11-active mode: provider loads, certificate enumerates, and `SignCades`
succeeds.

---

## 2026-06-04 — Prune embedded CryptoPro archive before launcher embed

**Context:** owner asked to verify that launchers no longer carry unnecessary
CryptoPro folders such as `Common`, other-bitness libraries, and MSI files. The
runtime layout v3 already skips those entries during AppData extraction, but the
build still embeds the full downloaded `cryptopro-plugin.zip` into both launcher
variants unless the archive is normalized before `go:embed`.

**Planned:**
- inventory the current pinned `2.0.15000` archive and confirm which entries are
  runtime-required for the current 32-bit native host / Browser Plug-in / Mini
  CSP path;
- change `build/fetch-cryptopro-plugin.ps1` so it still downloads and verifies
  the immutable full static archive against `build/cryptopro-plugin-lock.json`,
  but writes a slim normalized `internal/bootstrap/cryptopro-plugin.zip`;
- keep only the `Program Files\Crypto Pro\CAdES Browser Plug-in\...` subtree,
  stored in the slim archive as `CAdES Browser Plug-in\...`;
- add tests that accept both original static archive layout and normalized slim
  layout, and fail CI if the embedded bundle still contains `Program Files`,
  `Program Files 64`, MSI pseudo-paths, or `.msi` files;
- update docs with before/after size and CI verification.

**Inventory:** the full pinned static archive is `24,052,329` bytes compressed,
112 files, and `61,991,999` bytes raw. The current runtime subtree
`Program Files\Crypto Pro\CAdES Browser Plug-in\...` is 61 files and
`27,618,881` bytes raw; a local zip simulation produced a slim archive around
`11,247,865` bytes. The dropped trees are `Program Files 64`, `Common`,
`Common64`, `CommonAppData`, `System64`, `Windows`, MSI packages, and MSI
pseudo-path entries.

**Done:**
- added build-time normalization in `build/fetch-cryptopro-plugin.ps1`: it
  downloads and verifies the full static archive, then writes a slim
  `internal/bootstrap/cryptopro-plugin.zip`;
- the slim archive stores entries directly as `CAdES Browser Plug-in\...`;
- updated runtime zip mapping to accept both full source archive layout and
  already-normalized slim archive layout;
- extended CI tests so a real embedded bundle fails if it contains `Program Files`,
  `Program Files 64`, `Common`, `Common64`, MSI pseudo-path entries, or `.msi`
  packages;
- updated README, CHANGELOG, and bundle inventory docs.

**Verification:**
- `git diff --check` passed locally;
- local `go test`, `gofmt`, and PowerShell script execution are unavailable on
  Watson's Linux host.
- GitHub Actions `build-windows` run `26942148017` passed on commit `e049f52`,
  confirming the PowerShell normalizer, Windows build, and `internal/bootstrap`
  tests against a real embedded bundle;
- follow-up commit `ec3458e` preserves source zip entry timestamps so the slim
  archive is deterministic across embedded and remote builds;
- GitHub Actions `build-windows` run `26942298368` passed on commit `ec3458e`.
  CI logs show both launcher variants verified the full static source archive
  (`24,052,329` bytes, SHA-256
  `4590391e35c251cd4685d839ab62fad69e08716335931ac1c1b753b0cd346c6a`) and then
  embedded the slim archive (`11,245,901` bytes, 61 entries, SHA-256
  `cb4f8b5cfcecb65311b59e53d03bed8a067b85c427d31bcd639c21d21291917a`);
- CI `go test` passed for `github.com/code-agent-43824/kriptosfera/internal/bootstrap`
  (`0.555s`), including the guard that rejects `Program Files`, `Program Files 64`,
  `Common`, `Common64`, MSI pseudo-path entries, and `.msi` packages in the
  embedded bundle;
- downloaded artifacts contain `KriptosferaDemo.exe` (`190,750,720` bytes) and
  `KriptosferaDemo-remote.exe` (`17,713,152` bytes). Compared with the previous
  layout-v3 artifacts (`203,556,352` / `30,519,296` bytes), each launcher is about
  `12.8 MB` smaller.

**Next:** validate the resulting launchers on Windows. Do not prune inside
`CAdES Browser Plug-in` / `Mini CSP` until a fixed vendor build and token smoke
tests are available.

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

---

## 2026-06-03 — Trusted sites: launcher writes the CryptoPro CAdES trusted-sites list

**Owner request:** add `cryptopro.ru` and `mescheryakov.pro` to the plug-in's
trusted-sites list so the per-operation confirmation dialog is not shown on the
test pages, using the documented plug-in mechanism.

**Documented mechanism (verified in binaries + docs + the extension's own
`trusted_sites.js`):** per-user key `HKCU\Software\Crypto Pro\CAdESplugin`, value
**`TrustedSites`** of type **REG_MULTI_SZ**, entries are `scheme://host` with `*`
wildcards allowed (the extension's own example lists `https://*.cryptopro.ru`,
`https://*.ru`, etc.). Group policy alternative is
`HKLM\SOFTWARE\Policies\Crypto-Pro\CadesPlugin\TrustedSites` (admin); we use the
per-user HKCU path (no admin), consistent with our other registrations.

**Done (config-driven, in the launcher):**
- `AppConfig.trustedSites []string` (validated: each entry needs `scheme://host`,
  wildcards allowed) — `internal/config/config.go`, `validateAppConfig`.
- `internal/bootstrap/trusted_sites.go` (+ `_windows.go` reg.exe REG_MULTI_SZ via
  `\0` separators, `_other.go` no-op): `PrepareCryptoProTrustedSites` writes the
  list, gated by a state file (reuse when unchanged), non-fatal on failure. Wired
  into `bootstrap.Run` right after native-messaging registration.
- `payload-template/config/app-config.json` ships
  `https://cryptopro.ru`, `https://*.cryptopro.ru`, `https://mescheryakov.pro`,
  `https://*.mescheryakov.pro`.
- Tests in `trusted_sites_test.go` (write/reuse/skip/normalize + validation).
  gofmt/vet/`go test ./...`/windows+remote builds all green.
- Updated the plan's safety constraint (trusted sites for owner origins via the
  documented mechanism is allowed, config-scoped) and CHANGELOG.

**Apply now without a rebuild (owner machine):** the value is per-user REG_MULTI_SZ;
a one-off (cmd):
`reg add "HKCU\Software\Crypto Pro\CAdESplugin" /v TrustedSites /t REG_MULTI_SZ /d "https://cryptopro.ru\0https://*.cryptopro.ru\0https://mescheryakov.pro\0https://*.mescheryakov.pro" /f`
Otherwise the launcher writes it on next run (embedded build picks up the config
immediately; remote needs the republished payload).

**Next:** still blocked on the vendor path-resolution fix before the provider comes
up on a clean machine; trusted sites only suppress the confirmation dialog, they do
not affect provider loading.

---

## 2026-06-03 — Trusted sites confirmed working; re-pin remote payload

Owner verified on the machine WITH the system plug-in installed (not a clean-machine
proof): `reg query "HKCU\Software\Crypto Pro\CAdESplugin"` showed **no key** (launcher
reused the old extracted payload without `trustedSites`, so it skipped the write).
Adding the key manually (`reg add … TrustedSites REG_MULTI_SZ …`) + restart →
**the confirmation dialog disappeared**. This proves the registry location/format are
correct and the launcher code is right; it just needs the new app-config to reach it.

Re-pinned `build/payload-lock.json` to the payload built from the trustedSites commit
(`72a87b3`): sha `b8fca45a…`, size `173037195` (verified live on the server, payload.json
matches). So the **remote** launcher now ships the trustedSites config and writes the
key on next run; **embedded** gets it from a fresh build. After the first such run the
key is set (state-gated), no manual `reg add` needed.

Also ruled out the config-file hypothesis: Mini CSP `config.ini` has nothing about
web-site trust (only key devices/RNG/cert stores); the dialog is the plug-in's
(`npcades.dll`: TrustedSites/IsUntrustedSitesDisabled/ShowDocGetConfirm/Silent), driven
by the registry list — as the dialog text itself states.

---

## 2026-06-13 — Rutoken ЭЦП FKC + PKCS#11(active) carriers analyzed from Linux provider

Owner asked to port the missing Rutoken ЭЦП "pkcs11 (active)" and "FKC" carriers from
the Linux provider config into Mini CSP, Windows-adapted. Extracted the Linux reader
packages (`cprocsp-rdr-rutoken/cpfkc/cryptoki` 5.0.13800) from the pinned mirror; the
carrier defs are in each `postinst` (`cpconfig … -add`).

Findings: Mini CSP already has the **passive** Rutoken carriers (rutoken.dll). The
active modes were missing — **and the required reader DLLs are not in the slim bundle**:
FKC needs `cpfkc.dll`, pkcs11 needs `cryptoki.dll` + Rutoken `rtPKCS11ECP.dll` (none
shipped). So config alone is inert. Wrote `docs/cryptopro-rutoken-fkc-pkcs11.md` with the
analysis, the DLL requirement table, and the Windows-adapted `config.ini` fragment
(`rutokenfkc` + `rutokenfkc_nfc` KeyCarriers via cpfkc.dll; `cryptoki_rutoken` KeyDevice
with `pkcs11_dll = rtPKCS11ECP.dll`). ATR/mask copied 1:1; `librdr<X>.so→<X>.dll`;
`-connect`→`\<name>` subkey. Did NOT touch the vendor bundle (not in Git) — integration
is a build-time overlay in `fetch-cryptopro-plugin.ps1` once the SHA-pinned DLLs are
sourced and a token smoke test exists.

**Next (owner):** source `cpfkc.dll`/`cryptoki.dll` (full Win CryptoPro CSP) + x86
`rtPKCS11ECP.dll` (Rutoken drivers), pin them, overlay DLLs+fragment, then test FKC/pkcs11
with a real Rutoken ЭЦП.

---

## 2026-06-13 — Handoff for payload-modifying agent: Rutoken FKC/PKCS#11 DLLs

Wrote `docs/handoff-rutoken-fkc-pkcs11-payload.md` — actionable checklist to enable
Rutoken ЭЦП FKC + PKCS#11(active) in the embedded Mini CSP. Three **x86** DLLs to source
(host `nmcades.exe` is PE32): `cpfkc.dll` + `cryptoki.dll` (full Win CryptoPro CSP),
`rtPKCS11ECP.dll` (**"Драйверы Рутокен"** from rutoken.ru — x86). Pin via new
`build/rutoken-fkc-lock.json`; overlay DLLs into `CAdES Browser Plug-in\Mini CSP\` and
append the prepared config fragment (CP1251!) inside `build/fetch-cryptopro-plugin.ps1`;
bump `cryptoProPluginLayout` 3→4; rebuild (embedded bundle → both variants, no
payload-lock re-pin); verify on real Rutoken ЭЦП (FKC + pkcs11). Fragment + analysis live
in `docs/cryptopro-rutoken-fkc-pkcs11.md`.

---

## 2026-06-23 — Diagnostic runbook for Windows computer-use session (FKC/PKCS#11 still invisible)

Owner reports: Rutoken ЭЦП with certs in all three formats (csp, pkcs11, fkc) — the
bundled Mini CSP enumerates **only csp**; the **full system CSP sees all three**. Manual
attempts (copying the pkcs11 lib + config.ini "everywhere", incl. Program Files) didn't
help. Owner pushed back (correctly) on the reader-DLL "version skew" theory: plug-in and
CSP being different builds is normal CryptoPro practice, so version mismatch was dropped
as the lead hypothesis. Remaining hypotheses: **(A)** bad/misplaced config or overlay in
a folder the runtime never loads, vs **(B)** Mini CSP core doesn't implement the FKC/
pkcs11 reader devices (→ vendor bug).

Decision: run it on a real Windows VM via Claude Code + computer-use. Owner does the
admin-gated steps (install/remove system CSP, Rutoken drivers, insert token); the VM
agent does observation + config edits. Wrote
`docs/handoff-rutoken-fkc-diagnostic-runbook.md` — a self-contained runbook:
- **Phase 1 (no admin):** `ListDLLs nmcades.exe` → the folder of the loaded `cpcspi.dll`
  is the *authoritative* Mini CSP dir. Leading suspicion: it's `Program Files (x86)\...`
  (MSI `ADDMINICSP=1`), so our LOCALAPPDATA overlay is never loaded — which would
  explain the failed manual attempts directly.
- **Phase 2 (admin):** `reg export` the working system-CSP `Cryptography\CurrentVersion`
  tree = proven Windows ground-truth for `KeyDevices\cryptoki...` / `KeyCarriers\rutokenfkc...`;
  compare section-for-section to the authoritative `config.ini` (better reference than Linux).
- **Phase 3:** drop the 3 x86 DLLs (sourced locally from the installed CSP + Rutoken
  drivers, not pinned) + append `cryptoki_rutoken` to the authoritative `config.ini`
  (CP1251), restart, re-`ListDLLs` to confirm the readers bind, re-check enumeration.
- **Decision tree:** certs appear → hypothesis A (placement) → fix launcher overlay
  target, no vendor report. Config matches working registry 1:1 + DLLs present + readers
  never loaded → hypothesis B → file vendor bug with ListDLLs before/after + config diff.

Explicit guardrail in the runbook: config + observation only; **no** disassembly/patching
of vendor binaries — hitting that wall = stop and write the vendor report instead.

**Next:** VM agent runs Phase 1 and reports the authoritative Mini CSP path + whether our
overlay was ever in the load path.

---

## 2026-06-24 — Runbook Phase 1: which Mini CSP actually loads in nmcades.exe

Ran `docs/handoff-rutoken-fkc-diagnostic-runbook.md` Phase 1 on the owner's Windows
box (AMD64) with the launcher already running. Staged Sysinternals `Listdlls.exe` to
`C:\Tools` and snapshotted the live native host (`Listdlls.exe -accepteula nmcades` →
`C:\Tools\nmcades-dlls.txt`). Host: `nmcades.exe` pid 5748, launched from our overlay
`%LOCALAPPDATA%\Kriptosfera\apps\demo\0.5.0\Crypto Pro\CAdES Browser Plug-in\nmcades.exe`
(extension `iifchhfnnmpdbibifmljnfjhpififfog`).

**Decisive finding — the load path is SPLIT:**
- From our overlay: `nmcades.exe`, `npcades.dll`, the whole CAdES runtime, and the
  Mini CSP **helper** DLLs `Mini CSP\capi20.dll`, `asn1*.dll`, `cpsuprt.dll`
  (these are direct process imports, so they resolve from the process dir = our overlay).
- From **`C:\Program Files (x86)\Crypto Pro\CAdES Browser Plug-in\Mini CSP\`**:
  `cpcspi.dll` (the actual CSP provider core), `capi10.dll`, a second `capi20.dll`,
  a second `cpsuprt.dll`, `pcsc.dll`.

**Authoritative Mini CSP folder = `C:\Program Files (x86)\Crypto Pro\CAdES Browser Plug-in\Mini CSP\`**
— the directory of the loaded `cpcspi.dll`. cpcspi is pulled in via the HKLM MSI
provider registration (not by process-dir search), and it reads `config.ini` /
`[KeyDevices…]` / `[KeyCarriers…]` relative to **its own** directory. So the
carrier config that governs enumeration is the **Program Files** `config.ini`, NOT our
overlay copy.

**Target reader DLLs: all ABSENT** (expected pre-Phase-3): `cpfkc.dll`,
`cryptoki.dll`, `rtPKCS11ECP.dll` not loaded. `rutoken.dll` (passive control path)
also not loaded in this snapshot — no token operation had been triggered at capture time.

**Conclusion (runbook decision tree, Phase 1 branch):** confirmed the leading hypothesis
— our LOCALAPPDATA Mini CSP overlay (cpcspi + config.ini) is **dead weight**; the runtime
loads the MSI-installed Program Files Mini CSP. This directly explains the prior failed
manual attempts ("copied files everywhere, nothing changed"): the edited copies were
never the loaded ones. Any Phase 3 edit (drop the 3 x86 reader DLLs + append
`cryptoki_rutoken` to `config.ini`) must target the **Program Files** Mini CSP and
needs an elevated shell.

Evidence kept locally at `C:\Tools\nmcades-dlls.txt` (not committed — contains vendor
module paths only, no binaries).

**Next:** Phase 2 (admin-gated, owner) — confirm Rutoken drivers + full system CryptoPro
CSP are installed and a Rutoken ЭЦП token holding csp/pkcs11/fkc certs is inserted; then
`reg export` the working `HKLM\…\Crypto Pro\Cryptography\CurrentVersion` tree as the
proven ground-truth to diff against the Program Files `config.ini`.
---

## 2026-06-24 — Phase 1 analyzed; runbook revised; consolidated to main, branch removed

Reviewed the VM session's Phase 1 result (split load path: helper DLLs from our overlay,
but `cpcspi.dll` core + `config.ini` from `C:\Program Files (x86)\Crypto Pro\CAdES
Browser Plug-in\Mini CSP\` via HKLM MSI registration). Confirms our overlay provider is
bypassed when the plug-in is MSI-installed — explains the failed manual edits. Recorded
the product implication: reinforces the planned two-mode behavior (ride installed CSP
when present).

Revised `docs/handoff-rutoken-fkc-diagnostic-runbook.md`:
- Phase 1 marked DONE with the finding.
- **New Phase 2 (no admin, no token, do now):** read-only inventory of the Program Files
  Mini CSP — which reader DLLs (`cpfkc`/`cryptoki`/`rtPKCS11ECP`) are already present, and
  which `[KeyCarriers\rutokenfkc]` / `[KeyDevices\cryptoki_rutoken]` sections its
  `config.ini` already defines. Sharpest cheap discriminator: if `rutokenfkc` + `cpfkc.dll`
  are already there yet FKC is invisible → strong lean to hypothesis B (vendor gap).
- Phase 3 (admin/token): added a **positive control** (token in → confirm `rutoken.dll`
  loads + csp cert enumerates) plus the system-CSP `reg export` reference and an optional
  gold ListDLLs trace of the working system CSP.
- Phase 4: decisive edit now explicitly targets the **Program Files** Mini CSP; add only
  what Phase 2 found missing.

Per owner instruction, dropped the feature branch: merged its commits into `main`
(fast-forward) and deleted `claude/exciting-maxwell-SubP5` locally and on origin. Docs go
straight to `main` from here.

**Next (VM session):** run Phase 2 inventory and report the table.

---

## 2026-06-24 — Runbook Phase 2 (no-admin inventory of the authoritative Program Files Mini CSP)

Per the owner's instruction, ran a no-admin/no-token inventory of the **authoritative**
Mini CSP folder identified in Phase 1:
`C:\Program Files (x86)\Crypto Pro\CAdES Browser Plug-in\Mini CSP\` (where the loaded
`cpcspi.dll` lives). Read-only: directory listing + `config.ini` parsed as CP1251.

### Reader-DLL inventory (authoritative folder)

| DLL | Needed for | Present in Program Files Mini CSP? |
| --- | --- | --- |
| `cpcspi.dll` | CSP provider core | ✅ present |
| `rutoken.dll` | passive Rutoken carriers (control path) | ✅ present |
| `cpfkc.dll` | FKC reader | ❌ **MISSING** |
| `cryptoki.dll` | PKCS#11 reader | ❌ **MISSING** |
| `rtPKCS11ECP.dll` | Rutoken PKCS#11 lib (also checked next to `nmcades.exe`) | ❌ **MISSING** (both locations) |

The folder ships exactly the documented slim set (`asn1*, bio, capi10, capi20, cpasn1,
cpcspi, cplib, cpsuprt, cpui, dsrf, fat12, jacarta, pcsc, rutoken, safenet`). The full
MSI install does **not** add the FKC/cryptoki readers either.

### config.ini section inventory (authoritative folder)

| Section | Mode | Present? | Notes |
| --- | --- | --- | --- |
| `[KeyCarriers\rutokenfkc]` (+`\Default`) | FKC | ✅ present | references `DLL = "cpfkc.dll"` (3 mentions total) |
| `[KeyCarriers\rutokenfkc_nfc]` (+`\Default`,`\Contact`) | FKC NFC | ✅ present | references `cpfkc.dll` |
| `[KeyCarriers\RutokenFkcOld]` | legacy FKC | ✅ present | — |
| `[KeyDevices\cryptoki_rutoken]` | PKCS#11 active | ❌ **MISSING** | no `cryptoki.dll` / `rtPKCS11ECP.dll` reference anywhere (0 mentions) |
| `[KeyCarriers\Rutoken]`, `RutokenECP`, `RutokenECPM/MSC/SC` | passive | ✅ present | `rutoken.dll` (10 mentions) — the working control |

`KeyDevices` tree contains only `FAT12`, `HDIMAGE`, `PCSC` — no cryptoki device.

### Preliminary verdict: hypothesis **A** (config/placement gap), **not** B

- **FKC:** the carrier config is *already present* and points at `cpfkc.dll`, but
  `cpfkc.dll` is **absent from disk** → the carrier cannot bind → FKC stays invisible.
  This is a pure missing-DLL placement gap, fully explained without invoking a Mini CSP
  feature gap.
- **PKCS#11 active:** *both* the `[KeyDevices\cryptoki_rutoken]` section **and** the two
  DLLs (`cryptoki.dll` + `rtPKCS11ECP.dll`) are absent → needs config **and** DLLs.
- **We have NOT reached the hypothesis-B wall.** B (Mini CSP core ignores the device
  sections) can only be proven once the DLLs are placed and the config is complete and
  the readers *still* refuse to load (per the runbook decision tree). Right now there is a
  simpler, sufficient cause: the reader DLLs were never installed in the authoritative
  folder. So Phase 3 (drop `cpfkc.dll`/`cryptoki.dll` into the Program Files Mini CSP,
  `rtPKCS11ECP.dll` next to `nmcades.exe`, add the `cryptoki_rutoken` section) has a
  real chance of just working — and is the decisive A-vs-B experiment.

### Next
- **Phase 3 (needs elevation + sourcing the 3 x86 DLLs):** sourcing requires a full
  Windows CryptoPro CSP install (`cpfkc.dll`, `cryptoki.dll`) + Rutoken drivers
  (`rtPKCS11ECP.dll`, x86); editing the Program Files Mini CSP needs an admin shell.
  Owner to confirm both prerequisites before proceeding.
---

## 2026-06-24 — Phase 2 analyzed; runbook refined to an incremental FKC-first experiment

Reviewed the VM session's Phase 2 inventory. Clean result: the 3 reader DLLs
(`cpfkc`/`cryptoki`/`rtPKCS11ECP`) are **absent from disk** in the authoritative Program
Files Mini CSP; the **FKC carrier config is already present and correct** (points at
`cpfkc.dll`); only the `[KeyDevices\cryptoki_rutoken]` PKCS#11 section is missing.
Verdict: **hypothesis A (missing-DLL/placement), B wall not reached.** Also retires the
"version skew" theory for FKC — the file is absent, not the wrong version.

Refined `docs/handoff-rutoken-fkc-diagnostic-runbook.md`:
- Phase 2 marked DONE with the verdict.
- **Phase 4 is now incremental:** 4a = FKC as a *single-variable* test — drop **only**
  `cpfkc.dll` (config already correct), no edits, and see if FKC enumerates. This is the
  pivotal A-vs-B experiment: FKC appearing proves Mini CSP honors these carriers once the
  reader exists. 4b = PKCS#11 (both DLLs + the `cryptoki_rutoken` section) only after 4a.
- Phase 3: the full-CSP + Rutoken-driver installs now double as the **DLL source**; added
  capturing the Mini CSP `cpcspi.dll` FileVersion (fallback lead only) and **deriving the
  `cryptoki_rutoken` config from the working system-CSP registry export** rather than
  trusting our Linux-adapted fragment.
- Decision tree re-centred on Phase 4a.

Branch cleanup: local branch already gone; the stale remote `claude/exciting-maxwell-SubP5`
is fully merged into `main` but the git proxy rejects ref deletion (HTTP 403) and the
GitHub MCP has no delete-branch op — needs a manual delete in the GitHub UI.

**Next (VM session):** owner installs full CSP + Rutoken drivers + token, then run
Phase 4a (drop only `cpfkc.dll`, retest FKC) and report.

---

## 2026-06-24 — Cleaner experiment: copy DLLs from another machine, keep test box CSP-free

Owner's point (correct): no need to install a full system CSP on the test machine just to
obtain the reader DLLs — copy them from another machine that already has a working CSP.
Cleaner and more representative: installing a system CSP on the test box would register its
own provider in HKLM and could change which `cpcspi` loads / how the plug-in resolves the
provider, contaminating the measurement; and the product is meant to run without a system
CSP anyway. Updated Phase 3 of the runbook accordingly:
- Source x86 `cpfkc.dll` / `cryptoki.dll` (from the other machine's `…\Crypto Pro\CSP\`),
  `rtPKCS11ECP.dll` (its Rutoken drivers, pkcs11 only), and the `reg export` of the working
  carrier/device config — all **brought over**, nothing installed on the test box.
- Test box needs only the token inserted (its passive csp path already proves PC/SC + ATR).
- Phase 4a (FKC) needs **nothing** beyond dropping in `cpfkc.dll` — no system CSP, no
  Rutoken drivers, no config edit. Added a gotcha: if `cpfkc.dll` won't load, check its
  imports and copy any missing sibling DLLs from the same source machine (still pure file
  placement, not patching).

**Next (VM session):** owner brings `cpfkc.dll` from a CSP-equipped machine + inserts the
token; run Phase 4a (drop only `cpfkc.dll`, retest FKC) and report.

---

## 2026-06-24 — Runbook Phase 3 (partial): owner placed reader DLLs; FKC works; cryptoki section added

Owner manually placed the three x86 reader DLLs into the **authoritative** Program Files
Mini CSP and inserted a Rutoken ЭЦП. Result observed by owner: the **FKC** container is
now **visible**; the **PKCS#11** container is not yet.

Verified state (read-only):
- `cpfkc.dll` (x86 PE32, 256936 b), `cryptoki.dll` (x86 PE32, 217664 b),
  `rtPKCS11ECP.dll` (x86 PE32, 3867840 b) all present in
  `Program Files (x86)\Crypto Pro\CAdES Browser Plug-in\Mini CSP\`. Architecture is
  correct (host `nmcades.exe` is PE32; an x64 `rtPKCS11ECP.dll` would not have loaded).
- `rtPKCS11ECP.dll` is also present in the LOCALAPPDATA overlay next to the **running**
  `nmcades.exe` — so the bare-name (pkcs11_dll) process-dir load is covered.
- `config.ini` still **lacked** `[KeyDevices\cryptoki_rutoken]` (0 `cryptoki.dll`
  mentions) → the reason PKCS#11 stayed invisible while FKC (whose `rutokenfkc` carrier
  + `cpfkc.dll` reference were already in config) lit up as soon as `cpfkc.dll` landed.

**FKC result confirms hypothesis A** (config/placement gap) for the FKC path: the carrier
was pre-configured and only the reader DLL was missing.

Change applied (elevated, owner-approved UAC):
- Backed up `config.ini` → `config.ini.bak` (pristine 33470 b preserved).
- Idempotently appended the PKCS#11-active device section to the **authoritative**
  `config.ini` (CP1251 preserved), style mirroring the existing `[KeyDevices\PCSC]`:

  ```ini
  [KeyDevices\cryptoki_rutoken]
  "DLL"="cryptoki.dll"
  "Group"=1

  [KeyDevices\cryptoki_rutoken\"PNP cryptoki"]
  [KeyDevices\cryptoki_rutoken\"PNP cryptoki"\Default]
  pkcs11_dll = "rtPKCS11ECP.dll"
  ```

  config.ini 33470 → 33670 b. Edit script + result log kept locally under `C:\Tools`
  (not committed). No vendor binaries committed.

**Next:** owner fully restarts the launcher (close Chrome + any lingering `nmcades.exe`
so `cpcspi.dll` re-reads `config.ini`), reopens the internal-csp demo page with the
token in, and checks whether the **PKCS#11** container now enumerates.
- If it appears → hypothesis **A** confirmed for PKCS#11 too; fold the `cryptoki_rutoken`
  section into the build-time overlay (`build/fetch-cryptopro-plugin.ps1` /
  `docs/cryptopro-rutoken-fkc-pkcs11.md` already carry the fragment) and re-snapshot
  `ListDLLs` to confirm `cryptoki.dll` + `rtPKCS11ECP.dll` bind.
- If it still does not appear despite the DLLs being present and the section now matching
  the documented form → this is the first real signal toward hypothesis **B**; capture a
  ProcMon trace of the load and compare against the system-CSP path before concluding.

---

## 2026-06-24 — Phase 3 cont.: cryptoki.dll never loads; KeyDevice form likely wrong, not yet B

Owner restarted the launcher with the `cryptoki_rutoken` section in the authoritative
`config.ini` and the token in; the **PKCS#11 container is still not visible**. Snapshotted
the fresh `nmcades.exe` (pid 11952, started after the config edit) with ListDLLs
(`C:\Tools\nmcades-dlls-after.txt`):

| Reader DLL | Loaded into nmcades.exe? |
| --- | --- |
| `cpfkc.dll` (FKC) | ✅ LOADED (Program Files Mini CSP) — matches working FKC |
| `rutoken.dll` (passive control) | ✅ LOADED |
| `cryptoki.dll` (PKCS#11 reader) | ❌ **NOT loaded** |
| `rtPKCS11ECP.dll` | ❌ not loaded (it's pulled in *by* cryptoki.dll, which never loads) |

So even with the section present and the DLL on disk, Mini CSP **never loads
`cryptoki.dll`**. That is the decisive observation.

Why FKC ≠ PKCS#11 (mechanism):
- FKC `rutokenfkc` is a **KeyCarrier**: matched by ATR against a token already present in
  an active **PC/SC** reader (Rutoken is a smart card; `pcsc.dll` is loaded). The carrier
  fires and loads `cpfkc.dll`. This is why dropping `cpfkc.dll` alone was enough.
- `cryptoki_rutoken` is a **KeyDevice** — a separate, non-PnP reader class. Unlike PCSC it
  is not auto-enumerated; in the full CSP a PKCS#11 reader is added as an explicit reader
  **instance** (`cpconfig -hardware reader -add … -type cryptoki`), which is what makes the
  reader active and loads the cryptoki backend. Our Linux-adapted `[KeyDevices\cryptoki_rutoken]`
  (device type only, no reader instance) is likely **incomplete for Windows**.

Config schema notes from the authoritative `config.ini`:
- Top-level `[PKCS11]` section is the *wrong direction*: its (commented) `slotN` /
  `ProvGOST` / `ProvRSA` keys configure CryptoPro as a PKCS#11 **provider/slot**, not for
  consuming a token's own PKCS#11 library.
- `[Parameters]` has commented `EnabledCarrierTypes = 2` /
  `EnabledOperationsForDisabledCarriers = 0` — a possible carrier-type gate, bit semantics
  unknown; not safe to guess.
- This machine has **no system CryptoPro CSP** in the registry
  (`HKLM\…\Crypto Pro\Cryptography\CurrentVersion` absent under both native and WOW6432Node),
  so the proven Windows ground-truth cannot be exported here.

**Verdict: A-vs-B still undecided — NOT yet hypothesis B.** "cryptoki.dll never loads" is
consistent with B (core ignores added cryptoki KeyDevices) but equally with A (the Windows
reader-instance schema is missing from our Linux-adapted fragment). Per the runbook we must
not conclude B without the proven system-CSP reference.

**Next (to decide A vs B):**
1. **Ground-truth reference (preferred):** on the machine the DLLs came from (which has a
   full CSP that enumerates this Rutoken via PKCS#11), `reg export`
   `HKLM\SOFTWARE\(WOW6432Node\)Crypto Pro\Cryptography\CurrentVersion` — specifically the
   `KeyDevices`/reader entries for the working cryptoki reader — and diff against our
   section to recover the correct Windows form (likely a reader instance + GUIDs).
2. **Local ProcMon trace (parallel evidence):** capture cpcspi during enumeration to see
   whether it ever reads the `cryptoki_rutoken` section / attempts `LoadLibrary
   cryptoki.dll`. No attempt at all → strong B; an attempt that fails → A (path/schema).
---

## 2026-06-24 — ProcMon headless capture unusable on this box; pivot to reference-config path

Attempted the runbook's optional ProcMon trace to see whether `cpcspi` probes/loads
`cryptoki.dll` during a fresh provider init. Killed the stale `nmcades` so the owner's
re-enumeration would spawn a fresh host inside the capture window (confirmed: fresh
`nmcades` pid 4552 started 08:08 during capture).

**ProcMon headless does not record events on this machine.** Verified independently: with
`Procmon64.exe /AcceptEula /Quiet /Minimized /BackingFile` running, even deliberately
generated file I/O produced a backing file of only ~1 KB (header only), and `/SaveAs`
exports were empty (76 b / 0 b). The capture process arms and pre-allocates the 128 MB
backing file but writes no events (likely AV/driver interference with the ProcMon kernel
driver). Not worth more cycles; a GUI-driven capture would need manual interaction.

**Decisive local evidence already stands (ListDLLs):** `cpfkc.dll` loads (FKC works),
`cryptoki.dll` never loads despite the `[KeyDevices\cryptoki_rutoken]` section + DLL on
disk. The `cryptoki_rutoken` KeyDevice (Linux-adapted, device-type-only) does not bring up
a reader.

**Product angle worth noting:** FKC = the token computing GOST itself = the **active mode**.
Its ATR is identical to passive `RutokenECP` (same physical Rutoken ЭЦП); FKC vs passive is
chosen by which reader DLL binds. So **active-mode signing with the Rutoken ЭЦП already works
via the FKC carrier** now that `cpfkc.dll` is present. PKCS#11-active is an *alternative*
path to the same token, not a second capability — it may be unnecessary for the MVP signing
goal.

**A-vs-B verdict: still undecided, and PKCS#11 may simply not be needed.** To settle it
properly (if pursued) the runbook-mandated reference is required: a `reg export` of
`Crypto Pro\Cryptography\CurrentVersion` from a machine whose full CSP enumerates this
Rutoken via PKCS#11, to recover the correct Windows reader-instance schema and diff against
our section. Absent that, do not claim hypothesis B.

**Recommended next:** (1) validate end-to-end **test signature via the FKC carrier** (active
mode) — likely already satisfies MVP stage 7; (2) only if PKCS#11-active is explicitly wanted,
obtain the reference reg-export and rebuild the `cryptoki` reader config from it. The
`cryptoki_rutoken` section + the 3 DLLs remain in place on the test box (harmless; the
reader just isn't activated). config.ini backup at `config.ini.bak` if reverting is wanted.
---

## 2026-06-24 — PKCS#11 verdict: config proven correct vs vendor postinst, reader still never loads → hypothesis B

Pulled the **authoritative source** the repo's fragment was derived from: the Linux
`cprocsp-rdr-cryptoki-64_5.0.13800-7_amd64.deb` postinst (downloaded from the project
mirror, SHA-256 verified `d7382ecc…`). Its `cpconfig` commands define the cryptoki
reader exactly as:

```
\config\apppath  librdrcryptoki.so  → /opt/cprocsp/lib/amd64/librdrcryptoki.so
\config\apppath  librtpkcs11ecp.so  → librtpkcs11ecp.so
\config\KeyDevices\cryptoki_rutoken              Group=1, DLL=librdrcryptoki.so
\config\KeyDevices\cryptoki_rutoken\PNP cryptoki\Default   pkcs11_dll=librtpkcs11ecp.so
\config\debug    cryptoki=1
```

Two findings from this:
1. Our `[KeyDevices\cryptoki_rutoken]` section already matched the postinst **1:1** in
   structure (Group/DLL/PNP cryptoki\Default/pkcs11_dll).
2. The postinst ALSO populates `[apppath]` (mapping the reader + token DLL names to
   loadable modules) — the one step the earlier handoff deliberately skipped. So we added,
   to the authoritative Program Files `config.ini`:
   ```ini
   [apppath]
   cryptoki.dll = "cryptoki.dll"
   rtPKCS11ECP.dll = "rtPKCS11ECP.dll"
   ```
   and copied `cryptoki.dll` next to `nmcades.exe` (process dir) so the bare names
   resolve (`rtPKCS11ECP.dll` was already there).

**Result: still no PKCS#11.** Fresh `nmcades` (pid 5528) after the apppath fix —
ListDLLs (`C:\Tools\nmcades-after2.txt`): `cpfkc.dll` loads (FKC), `rutoken.dll`
loads (passive), **`cryptoki.dll` still NOT loaded**. The cryptoki KeyDevice is not
instantiated even with a config that matches the vendor's own postinst 1:1 and both DLLs
present in the authoritative folder + the process dir.

**Verdict — hypothesis B (Mini CSP feature gap).** Per the runbook decision tree: config
matches the proven-good source 1:1, reader DLLs present, yet Mini CSP never loads
`cryptoki.dll` → the bundled Mini CSP core does not implement the cryptoki/PKCS#11-active
reader device. Architecturally consistent: Mini CSP is a stripped CSP; FKC rides the
built-in PC/SC reader (`pcsc.dll`), whereas the cryptoki reader is a separate reader
backend that on Linux ships as its own packages (`cprocsp-rdr-cryptoki` + the
`lsb-cprocsp-rdr` subsystem) — that plumbing appears absent from the bundled Mini CSP.

**Residual unknowns (kept honest, not blockers):** (a) not compared against a *Windows*
full-CSP registry export (the postinst is Linux, but the CryptoPro config schema is shared);
(b) `[Parameters] EnabledCarrierTypes` (commented) not explored — a possible carrier-type
gate, but bit semantics undocumented and risky to guess (could disable the working FKC/passive
carriers). Neither is likely to overturn the verdict.

**Why this is fine for the product:** FKC = the token computing GOST itself = **active mode**,
same Rutoken ЭЦП (identical ATR to passive `RutokenECP`). Active-mode signing already works
via the FKC carrier. PKCS#11-active is a redundant alternative path, not a missing capability.

**Recommendation:** treat bundled-Mini-CSP PKCS#11-active as **not supported (vendor
limitation)**; rely on FKC for active mode. If PKCS#11-active is ever required, it needs a
CryptoPro Mini CSP build that includes the cryptoki reader subsystem (vendor ask). The
`cryptoki_rutoken` section + `[apppath]` entries are harmless and left in place on the
test box (`config.ini.bak` available to revert).
---

## 2026-06-24 — Pushing PKCS#11 further: version swap, cpconfig, debug capture — still no cryptoki load

User asked to exhaust the PKCS#11 path. Ruled out, one lever at a time:

**1. DLL version mismatch — ruled OUT.** Versions: `cpcspi.dll` (Mini CSP core)
ProductVersion **5.0.13000**; user-placed `cryptoki.dll` was Prod **5.0.13800** (newer,
from a full-CSP machine); `cpfkc.dll` also 5.0.13800 yet loads. Swapped both
`cryptoki.dll` copies (Mini CSP + process dir) for the mirror's pinned **5.0.13000** build
(matches core; SHA-verified). Fresh `nmcades` (pid 1128) — `cryptoki.dll` **still not
loaded**. Version is not the cause. (Backups: `cryptoki.dll.user13800.bak`.)

**2. cpconfig.exe inspection (canonical tool, present in Mini CSP).** `-hardware reader
-view` lists exactly ONE reader instance: `Aktiv Rutoken ECP 0` → `Reader name: All PC/SC
readers` — i.e. only the PC/SC reader. No cryptoki reader. `-hardware media -view` lists all
carriers incl. `rutokenfkc`/`rutokenfkc_nfc` (FKC works). Key structural insight: **reader
instances are NOT stored in config.ini — they are PnP-enumerated at runtime** by each
KeyDevice's PNP node (the `Aktiv Rutoken ECP 0` entry is absent from config.ini). A reader
instance's `Reader name` = the device's `PNP X\Default\Name`. The PCSC PnP found the
physical reader; the `PNP cryptoki` enumerator produced nothing — because its reader DLL
(`cryptoki.dll`) never loads. So the cryptoki reader can't be made to appear via static
config; it depends on the core loading the device DLL and running its PnP enumerator.

**3. apppath mappings — added, no effect.** (prior entry) Mirrored the postinst's
`[apppath]` (cryptoki.dll/rtPKCS11ECP.dll) + put both DLLs in the nmcades process dir;
`cryptoki.dll` still never loads.

**4. CryptoPro internal debug log — could not capture (inconclusive, not evidence).** The
`[debug]` section has subsystem logging on, but no log file appears (no registry log-folder
on this CSP-less machine). Tried a hand-rolled user-mode DBWIN/`OutputDebugString` listener:
it captured a **64-bit** test probe but NOT a **32-bit** one, and `nmcades.exe` is 32-bit —
so it would not see nmcades's output regardless. Treat the empty capture as **inconclusive**,
not as proof. (A proper tool — Sysinternals DebugView Win32 capture — handles 32-bit and could
still yield the core's reason; kernel-driver tools, ProcMon-style, are AV-blocked on this box.)

### Verdict (high confidence): hypothesis B — bundled Mini CSP core lacks the cryptoki reader
The `cryptoki_rutoken` KeyDevice is the only **config-added** reader device; the core loads
every **built-in** device/carrier DLL (pcsc, cpfkc, rutoken) but never loads `cryptoki.dll`,
across correct config (matches vendor postinst 1:1), apppath, process-dir placement, and a
version-matched DLL. The cryptoki reader support ships on Linux as separate packages
(`cprocsp-rdr-cryptoki` + `lsb-cprocsp-rdr` subsystem); that plumbing appears absent from
this Mini CSP (CSP core 5.0.13000).

### Remaining forward options (none quick; FKC already covers active-mode signing)
1. **Swap the whole Mini CSP to a 5.0.13800-core build** (matches the postinst we proved
   against). If cryptoki support is "present in a newer core, absent in 5.0.13000", this is the
   real fix — but it means re-pinning the bundled Mini CSP (bigger change; the 2.0.15700 plugin
   was rejected earlier for unrelated breakage).
2. **DebugView (proper)** capture to get the core's own "why" before finalizing B.
3. **`EnabledCarrierTypes`** experiment — undocumented bitmask; risky (could disable working
   FKC/passive); low probability.
4. **Vendor ask** — CryptoPro: does bundled Mini CSP support the cryptoki/PKCS#11-active reader,
   and if so how.

Config left in place (harmless); `config.ini.bak` (pristine) + `*.user13800.bak` available.
---

## 2026-06-24 — DebugView: tool works, but Mini CSP emits no OutputDebugString (verdict unchanged)

User picked the DebugView route to get the core's own "why". Set up Sysinternals DebugView
(`Dbgview.exe /accepteula /t /g /l C:\Tools\dbgview.log`, elevated, global Win32 capture).
**Validated it captures 32-bit output** — a WOW64 `OutputDebugString` test probe
(`KRIPTO_TEST_PROBE_32bit_WOW64`) was logged (this is what my earlier hand-rolled DBWIN
listener missed). Killed `nmcades`, user re-triggered enumeration (fresh `nmcades` pid
10388 captured in-window).

**Result: zero output from `nmcades`/CryptoPro.** Despite `[debug]` toggles being on,
the Mini CSP emits nothing via `OutputDebugString`, and no CSP log file is produced anywhere
(searched process dir, Mini CSP, Program Files, TEMP, C:\ — nothing fresh). CryptoPro's debug
**output channel** is configured separately (registry `…\Crypto Pro\…\Debug` log-folder),
which is absent on this system-CSP-less machine, so the `[debug]` toggles are inert. The
DebugView avenue is therefore exhausted: the tool is fine, the CSP is simply silent on the
debugger channel.

**Net:** no new info to overturn the verdict. All local diagnostic channels are now exhausted:
ProcMon (kernel driver AV-blocked), hand-rolled DBWIN (32-bit gap), DebugView (CSP emits
nothing), CSP file log (unconfigured/absent). Hypothesis **B** stands on the behavioral
evidence: correct config (vendor postinst 1:1) + apppath + DLLs placed + version-matched, yet
`cryptoki.dll` never loads and no cryptoki reader is ever instantiated (cpconfig), while every
built-in device/carrier DLL loads.

**Most promising remaining lever** = the CSP **core version**: this Mini CSP core is
`cpcspi.dll` Prod **5.0.13000**; the proven cryptoki postinst is from **5.0.13800**, and the
owner has a full **5.0.13800** CSP machine. Next concrete step to decide if PKCS#11-active is
achievable at all: confirm on that 5.0.13800 machine whether the Rutoken enumerates via
PKCS#11-active (`cpconfig -hardware reader -view` shows a cryptoki reader; demo page shows the
pkcs11 container). If yes → the gap is core-version; pursue a 5.0.13800-core Mini CSP. If no →
it's a token/mode issue, not Mini CSP. FKC remains the working active-mode path either way.
---

## 2026-06-24 — SESSION CLOSE: consolidated report + critical ARM-emulation caveat

Session paused by owner. All results consolidated into
docs/pkcs11-active-investigation-2026-06.md (self-contained; read it first next time) and
indexed in docs/README.md. Nothing from this session is lost.

**Critical environment fact discovered at close:** this entire run was on **Apple Silicon
(ARM64) / Parallels** — the Windows guest reports PROCESSOR_ARCHITECTURE=AMD64 but the
native CPU is ARM64 (PROCESSOR_IDENTIFIER=ARMv8 (64-bit), Parallels ARM Virtual Machine),
so the x86 
mcades.exe and all CryptoPro DLLs ran under **x64/x86-on-ARM emulation**.

This **re-frames the PKCS#11 = hypothesis-B verdict as provisional**: FKC and passive PC/SC
work under emulation, but the cryptoki path (CryptoPro cryptoki.dll → Rutoken
tPKCS11ECP.dll → token over USB/PC-SC) is exactly the nested native-lib + device path that
can fail under emulation while simpler paths succeed. So "Mini CSP lacks the cryptoki reader"
is NOT proven — it may be an ARM-emulation artifact.

**Owner will retest on native x64 later.** That settles emulation-vs-feature-gap. The PKCS#11
config + DLL placement to reproduce, the exact backups/state changes on the test box, the
authoritative postinst form, the DLL version table, and the mirror SHAs are all in
docs/pkcs11-active-investigation-2026-06.md.

**Settled this session (holds regardless of arch):** FKC (active-mode) signing works on the
bundled Mini CSP once cpfkc.dll is present — Phase-1/2 placement findings, the authoritative
Mini CSP folder, and the FKC = active-mode insight are all confirmed.
