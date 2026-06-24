# PKCS#11-active (cryptoki) on bundled Mini CSP — investigation report (2026-06-24)

Self-contained record of the FKC + PKCS#11-active diagnostic run so a future session
(especially the planned **native x64** retest) does not repeat it. Companion to
`docs/handoff-rutoken-fkc-diagnostic-runbook.md` and `docs/cryptopro-rutoken-fkc-pkcs11.md`;
blow-by-blow timeline is in `docs/worklog.md` (2026-06-24 entries).

## TL;DR

- **FKC (active mode) WORKS** on the bundled Mini CSP once `cpfkc.dll` is present — the
  Rutoken ЭЦП FKC container enumerates. FKC = the token computing GOST itself = active mode,
  same physical token as passive `RutokenECP` (identical ATR). **This already satisfies the
  MVP active-signing goal.**
- **PKCS#11-active (cryptoki reader) does NOT work** on this build: `cryptoki.dll` is never
  loaded by the Mini CSP host, so no cryptoki reader is instantiated and the PKCS#11 container
  never appears — *even with* a config that matches the vendor's own Linux postinst 1:1, the
  `[apppath]` mappings added, both DLLs placed in the process dir, and a version-matched
  `cryptoki.dll`.
- **Working verdict: hypothesis B** (the bundled Mini CSP core lacks the cryptoki reader
  subsystem) — BUT see the big caveat below.

## ⚠️ BIG CAVEAT — everything here ran under ARM emulation

The test machine is **Apple Silicon (ARM64), Parallels ARM Virtual Machine**, running a
Windows guest whose `PROCESSOR_ARCHITECTURE` reports `AMD64` while the native CPU is ARM64
(`PROCESSOR_IDENTIFIER = ARMv8 (64-bit)`, Win32_Processor Architecture = 12). i.e. the x86
`nmcades.exe` (PE32) and all CryptoPro DLLs execute under **x64/x86-on-ARM emulation**.

FKC and the passive PC/SC path work under this emulation, so emulation is not a blanket
blocker. But the cryptoki reader path (CryptoPro `cryptoki.dll` → Rutoken `rtPKCS11ECP.dll`
→ token over USB/PC-SC) is exactly the kind of nested native-library + device path that can
fail under emulation while simpler paths succeed. **So "B" is not proven to be a true Mini CSP
feature gap — it may be an ARM-emulation artifact.**

### → Cleanest next step: retest on a NATIVE x64 (Intel/AMD) Windows machine
On native x64, redo just the PKCS#11 portion (the config + DLLs below) and re-check
`cpconfig -hardware reader -view` / the demo page. If the cryptoki reader appears there, the
blocker was emulation, not Mini CSP. The owner plans this x64 retest.

## Environment / key facts

- Authoritative Mini CSP (where the loaded `cpcspi.dll` lives, confirmed by ListDLLs):
  `C:\Program Files (x86)\Crypto Pro\CAdES Browser Plug-in\Mini CSP\`.
  Edits to the *runtime overlay* copy under `%LOCALAPPDATA%\Kriptosfera\...` are **dead weight**
  — the provider loads `cpcspi.dll` from Program Files (MSI/`ADDMINICSP`) and reads its
  `config.ini` from its own dir. This explains earlier "copied files everywhere, nothing
  changed" failures.
- `nmcades.exe` (the native-messaging host where crypto happens) is **PE32 / x86** and runs
  from the overlay: `%LOCALAPPDATA%\Kriptosfera\apps\demo\0.5.0\Crypto Pro\CAdES Browser Plug-in\nmcades.exe`.
- This machine has **no system CryptoPro CSP** in the registry
  (`HKLM\SOFTWARE\(WOW6432Node\)Crypto Pro\Cryptography\CurrentVersion` absent) — so no
  Windows ground-truth registry export is available here, and CryptoPro's debug-log output
  channel (registry-configured) is unset.

## DLL versions observed (FileVersion / ProductVersion)

| DLL | FileVer | **ProdVer** | size | loads in nmcades? |
| --- | --- | --- | --- | --- |
| `cpcspi.dll` (Mini CSP core) | 5.0.18886 | **5.0.13000** | 2238968 | n/a (the core) |
| `cpfkc.dll` (FKC reader) | 5.0.12740 | 5.0.13800 | 256936 | ✅ yes — FKC works |
| `rutoken.dll` (passive) | 5.0.16145 | 5.0.16145 | 458224 | ✅ yes |
| `cryptoki.dll` (owner-placed) | 5.0.16556 | 5.0.13800 | 217664 | ❌ never |
| `cryptoki.dll` (mirror, pinned) | 5.0.16305 | **5.0.13000** | 210304 | ❌ never (after swap) |
| `rtPKCS11ECP.dll` (owner, Mini CSP) | 2.15.1.0 | — | 3867840 | ❌ (loaded by cryptoki.dll, which never loads) |
| `rtPKCS11ECP.dll` (mirror, overlay) | 1.4.02.0 | — | 1593344 | ❌ |

Note the **core is Prod 5.0.13000** while the owner-sourced readers are Prod 5.0.13800.
Version was tested and ruled out (see below), but the core-vs-reader version split is the
leading remaining lever if the x64 retest is inconclusive.

## What was tried, and ruled out (PKCS#11 path)

1. **Config structure** — added `[KeyDevices\cryptoki_rutoken]` (Group=1, DLL=cryptoki.dll,
   `PNP cryptoki\Default` pkcs11_dll=rtPKCS11ECP.dll). Confirmed **1:1 identical** to the
   authoritative vendor source (the `cprocsp-rdr-cryptoki-64_5.0.13800-7` postinst — see
   below). Not the cause.
2. **`[apppath]` mappings** — the postinst also populates `[apppath]` (name→module), which the
   earlier handoff deliberately skipped. Added `cryptoki.dll = "cryptoki.dll"` and
   `rtPKCS11ECP.dll = "rtPKCS11ECP.dll"`, and copied `cryptoki.dll` next to `nmcades.exe` so
   bare names resolve from the process dir. No effect.
3. **DLL placement** — both `cryptoki.dll` and `rtPKCS11ECP.dll` present in the Mini CSP dir
   AND the nmcades process dir. No effect.
4. **DLL version mismatch** — swapped both `cryptoki.dll` copies for the mirror's **5.0.13000**
   build (matches `cpcspi.dll` ProductVersion; SHA-verified). Still never loads. Not the cause.
5. **cpconfig.exe inspection** (`Mini CSP\cpconfig.exe`): `-hardware reader -view` shows only
   the PC/SC reader instance (`Aktiv Rutoken ECP 0` → `All PC/SC readers`); no cryptoki reader.
   **Reader instances are PnP-enumerated at runtime, not stored in config.ini** — so the
   cryptoki reader cannot be hand-added via static config; it depends on the core loading the
   device DLL and running the `PNP cryptoki` enumerator (which it never does).
6. **Diagnostic channels for the core's own "why" — all exhausted:**
   - **ProcMon**: kernel-driver capture is blocked by AV on this box (empty backing file).
   - **Hand-rolled DBWIN/OutputDebugString listener**: captures 64-bit but NOT 32-bit; nmcades
     is 32-bit → inconclusive, discarded.
   - **Sysinternals DebugView** (the right tool, captures 32-bit — validated with a WOW64 test
     probe): **zero output from nmcades/CryptoPro**. Mini CSP emits nothing via OutputDebugString.
   - **CSP file log**: none produced anywhere (output channel is registry-configured; registry
     absent). `[debug]` toggles in `config.ini` are therefore inert here.

**Conclusion:** every *built-in* device/carrier DLL loads (pcsc, cpfkc, rutoken); the only
*config-added* reader device (`cryptoki_rutoken`) never loads its DLL → the bundled Mini CSP
(core Prod 5.0.13000) does not bring up the cryptoki reader. Caveat: ARM emulation (above).

## Authoritative source for the cryptoki config (re-pullable)

The repo fragment derives from the Linux `cprocsp-rdr-cryptoki` postinst. Pulled & SHA-verified:

- Package list (SHA256SUMS):
  `https://mescheryakov.pro/kriptosfera/cryptopro/csp/linux/5.0.13800/6928220796ea0bbf36985b15bbf4f1d673c971c337833220ab6511fb6b481bc5/SHA256SUMS`
- cryptoki reader deb:
  `…/amd64/deb/cprocsp-rdr-cryptoki-64_5.0.13800-7_amd64.deb` — sha256
  `d7382eccf1516eeac82add80bd141eb4358d9f02a3d3921d24344fcc01802622`
- mirror windows-x86 `cryptoki.dll` 5.0.13000 — sha256
  `5f2c3742fa00cf0ec4c4fca0dcf81ffc39e798d86880bb977e5af9436d94fa6a`

postinst cryptoki definition (the proven-correct form; ours matches it):
```
\config\apppath  librdrcryptoki.so → /opt/cprocsp/lib/amd64/librdrcryptoki.so
\config\apppath  librtpkcs11ecp.so → librtpkcs11ecp.so
\config\KeyDevices\cryptoki_rutoken              Group=1, DLL=librdrcryptoki.so
\config\KeyDevices\cryptoki_rutoken\PNP cryptoki\Default   pkcs11_dll=librtpkcs11ecp.so
\config\debug    cryptoki=1
```

Windows-adapted form applied (CP1251 `config.ini`):
```ini
[apppath]
cryptoki.dll = "cryptoki.dll"
rtPKCS11ECP.dll = "rtPKCS11ECP.dll"

[KeyDevices\cryptoki_rutoken]
"DLL"="cryptoki.dll"
"Group"=1
[KeyDevices\cryptoki_rutoken\"PNP cryptoki"]
[KeyDevices\cryptoki_rutoken\"PNP cryptoki"\Default]
pkcs11_dll = "rtPKCS11ECP.dll"
```

## Changes made to the TEST MACHINE (for reproduction / cleanup)

All under `C:\Program Files (x86)\Crypto Pro\CAdES Browser Plug-in\` (authoritative). Harmless
(the reader just doesn't activate); left in place. Backups exist to revert:

- `Mini CSP\config.ini` — added the `cryptoki_rutoken` section + `[apppath]` entries.
  Pristine backup: `Mini CSP\config.ini.bak` (33470 b).
- `Mini CSP\cryptoki.dll` and `…\nmcades.exe`-dir `cryptoki.dll` — swapped to mirror 5.0.13000.
  Backups of the owner-placed 5.0.13800 copies: `cryptoki.dll.user13800.bak` (both dirs).
- Owner manually placed `cpfkc.dll`, `cryptoki.dll`, `rtPKCS11ECP.dll` into `Mini CSP\` (the
  FKC/PKCS#11 reader DLLs). These are NOT committed to Git (vendor binaries).

Local-only artifacts (not committed): `C:\Tools\` holds ListDLLs/ProcMon/DebugView, the
ListDLLs snapshots (`nmcades-dlls*.txt`), the extracted cryptoki `.deb`/postinst, and the
edit/probe scripts (`add-cryptoki-section.ps1`, `add-apppath.ps1`, `swap-cryptoki.ps1`,
`dbwin.ps1`, `probe32.ps1`).

## Recommendations / next steps (priority order)

1. **Native x64 retest** of the PKCS#11 portion (config + DLLs above). Settles the
   emulation-vs-feature-gap question. — *owner will do this.*
2. If still failing on x64: try a **5.0.13800-core Mini CSP** (the postinst/owner full CSP are
   5.0.13800; this bundle's core is 5.0.13000) — swap `cpcspi.dll`/`capi20.dll`/reader subsystem
   or re-pin the whole Mini CSP. The cryptoki reader support may simply not exist in 5.0.13000.
3. Reference check on the owner's full **5.0.13800** CSP machine: does the Rutoken enumerate via
   PKCS#11-active there (`cpconfig -hardware reader -view` shows a cryptoki reader; demo page
   shows the pkcs11 container)? If not, it's a token/mode issue, not Mini CSP.
4. Vendor ask to CryptoPro: does bundled Mini CSP support the cryptoki/PKCS#11-active reader,
   and if so how.

**Either way, FKC already delivers active-mode Rutoken signing**, so PKCS#11-active is not an
MVP blocker — it is a redundant alternative path to the same token.
