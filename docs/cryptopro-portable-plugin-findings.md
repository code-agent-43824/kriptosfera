# CryptoPro CAdES plug-in — portable (no-install) blocker: findings

Status: **investigation paused, awaiting a vendor fix from CryptoPro.** This
document is the consolidated record of what we found while trying to run the
bundled CAdES Browser Plug-in (with Mini CSP) **from our own extracted directory
on a clean machine, without an MSI install**. Day-by-day detail is in
`docs/worklog.md`; this file is the standalone summary for the next agent / for
the vendor conversation.

## TL;DR

- The MV2 stack itself is correct: plug-in **2.0.15000** + MV2 extension
  **1.2.13** + Chrome **138** + `ExtensionManifestV2Availability` (see
  `docs/cryptopro-csp-lite-plan.md`). On a machine where the plug-in is **installed
  via MSI (`ADDMINICSP=1`)** the launcher works end to end.
- On a **clean machine** (our portable extraction, no MSI) the provider does not
  come up. Root cause: the plug-in resolves its own module/provider paths from a
  **hardcoded preferred image base** passed to `GetModuleFileName`, not from the
  real module location. Under ASLR this misses and the plug-in falls back to
  `%ProgramFiles%\Crypto Pro\CAdES Browser Plug-in`, which does not exist on a
  clean machine.
- This is a **CryptoPro bug**, replicated across modules. Byte-patching it on our
  side is a dead end (see below). The clean fix is on the vendor: resolve paths
  relative to each module's real `HINSTANCE`.

## Reproduction / environment

- Launcher run (remote or embedded) on a clean Windows 11 box, no CryptoPro
  installed. Start URL = the internal-csp demo page (sets `EnableInternalCSP`).
- At the time of the reproduction, bundle `2.0.15000` was extracted to the
  pre-layout-v3 path:
  `%LOCALAPPDATA%\Kriptosfera\apps\demo\0.5.0\cryptopro\plugin\cryptopro-cades-plugin-2.0.15000\Program Files\Crypto Pro\CAdES Browser Plug-in\`.
- Current layout v3 shortens the AppData path to:
  `%LOCALAPPDATA%\Kriptosfera\apps\demo\0.5.0\Crypto Pro\CAdES Browser Plug-in\`.
- Native messaging host `ru.cryptopro.nmcades` registered in HKCU pointing at our
  extracted `nmcades.exe`. Extension `iifchhfnnmpdbibifmljnfjhpififfog` loads.
- Symptom on the page: "Расширение загружено" ✓, but the plug-in never reports
  loaded / the provider never comes up → "истекло время ожидания загрузки плагина".
  Earlier (before patching) a popup: **"Error occured while trying to get mydss.dll
  installation path. Maybe CryptoPro Browser plug-in was not correctly installed."**

## Root cause (confirmed by static + dynamic analysis)

Analysis with `pefile`/`capstone` on the `2.0.15000` binaries (no Ghidra):

- **`npcades.dll`** — `nmcades.exe` loads it; it builds module/provider paths via
  `GetModuleFileNameA/W(hModule = 0x10000000, …)` — the **hardcoded preferred
  ImageBase**, not the real `HINSTANCE`. Sites (RVA): `0x4069` (path helper, used
  for `mydss.dll` etc.), `0x54cf2` (builds `…\Mini CSP\capi20.dll`), `0x56637`
  (`GetModuleFileNameW` + HKLM registry read). It also resolves `SHGetFolderPathW`
  dynamically (`LoadLibrary("SHELL32.dll")`+`GetProcAddress`); `SHELL32` is not in
  the static import table. The provider (`capi20.dll`) is loaded via `LoadLibrary`
  (+`GetModuleHandleA("capi20.dll")`), i.e. **internal CSP, no `CryptAcquireContext`,
  no registry** — the approach is right.
- **`cades.dll`** — same bug: `push 0x10000000` → `GetModuleFileName` at RVA
  `0x252a` (path helper) and `0x4f09a` (GMFW + HKLM). This is why
  "Версия плагина" reads **`0.0.0000`** — `cades.dll` can't locate itself to read
  its own `GetFileVersionInfo`.
- DLL characteristics for both = `0x140` (**ASLR / DYNAMIC_BASE on**), preferred
  base `0x10000000`. Under ASLR the DLLs load elsewhere, so
  `GetModuleFileName(0x10000000)` returns the wrong/empty path → fall back to
  `%ProgramFiles%`.
- No self-integrity strings in `npcades.dll`/`cades.dll` (only document-signing
  APIs). `capi20.dll`/`cpcspi.dll` are the **certified CSP** — never patched.

This also explains the owner's bisect: renaming the *system* `nmcades.exe` did not
break a working install (our host is used), but renaming the *system* `Mini CSP`
folder did — the provider was loading from `%ProgramFiles%`, not from our dir.
Copying the plug-in folder elsewhere does not help, because the bug depends on the
DLL's load address (and on MSI's system registration), not on file location.

## What we tried (and why each failed)

1. **Registry `AppPath`** (`HKCU`/`HKLM\…\WOW6432Node\Crypto Pro\Cryptography\
   CurrentVersion\AppPath` → our dir). No effect; the key is absent even on the
   working install, so it isn't the path mechanism.
2. **Junction** `C:\Program Files (x86)\Crypto Pro\CAdES Browser Plug-in` → our
   dir. Doesn't help with the original DLLs because they don't reference the
   `Program Files` string — they call `GetModuleFileName(0x10000000)`.
3. **Disable ASLR** (header `DYNAMIC_BASE` clear; system Exploit Protection off).
   Didn't make the DLL land on `0x10000000` (address occupied / mandatory bottom-up).
4. **Byte-patch `push 0x10000000 → push 0`** (so `GetModuleFileName(NULL)` returns
   the host-exe path = our dir). It *did* move modules to load from our dir and
   removed the `mydss` error — but **broke the version/handshake**, because the
   version path needs the path to `cades.dll` (for `GetFileVersionInfo`), not the
   exe. Patching `cades.dll`'s helper the same way did not fix it either.
   **Conclusion: a single NULL/one-byte rewrite can't satisfy all sites** — different
   call sites need paths to *different* modules (`cades.dll` for version,
   `…\Mini CSP\` for the provider). The correct fix is `GetModuleHandle("<module>")`
   per site, which is not a byte patch.

## What we proved about the host (Go stdin/stdout probe)

Since a `> file` redirect just buffered, we built a tiny Go probe (Windows) that
launches `nmcades.exe`, writes a length-prefixed native-messaging message and
**actively reads** stdout. Findings:

- **`nmcades` is NOT hung** — it reads, processes, replies and flushes. The
  `ReadFile(stdin)` idle seen in all minidumps is just normal between-message wait.
- It answers `CreateObject CAdESCOM.About` (objid=0) with
  `{"type":"error","message":"Can't find object by id"}` — and **the same on the
  working-install machine** through the probe. So our replayed handshake is
  incomplete (objid=0 root object isn't registered without the full browser
  sequence); this is a probe limitation, not a machine difference.
- From a dump's process memory we recovered the real dialog:
  `cadesplugin.EnableInternalCSP` → `true`, then `CreateObject CAdESCOM.About`
  (requestid 3) — i.e. internal-CSP engages and the protocol gets that far.

To fully reproduce the browser-side handshake one needs the real protocol from
`cadesplugin_api.js` (referenced by the demo page as `../cadesplugin_api.js`).

## Vendor report (the ask to CryptoPro)

> `npcades.dll` and `cades.dll` call `GetModuleFileNameA/W` with a hardcoded
> `0x10000000` (the DLL's preferred ImageBase) instead of the module's real
> `HINSTANCE`. With `/DYNAMICBASE` (ASLR) the DLL rarely loads at that base, so the
> call returns the wrong/empty path and the plug-in falls back to
> `%ProgramFiles%\Crypto Pro\CAdES Browser Plug-in`. This blocks any side-by-side /
> portable deployment next to `nmcades.exe` without an MSI install. Fix: pass the
> real `HINSTANCE` (captured in `DllMain`) or `GetModuleHandleW(L"npcades.dll")` /
> `GetModuleHandleW(L"cades.dll")` at each path-resolution site. Affected RVAs in
> `2.0.15000`: npcades `0x4069/0x54cf2/0x56637`, cades `0x252a/0x4f09a`.

## Status / next

- **Paused**; owner is in contact with CryptoPro and awaiting a fixed build.
- When a fixed plug-in arrives: re-pin it (`build/cryptopro-plugin-lock.json`),
  re-run the launcher on a clean machine, expect provider load + certificate
  enumeration + `SignCades` with a Rutoken. Long-term, also migrate back to the
  latest Chromium + Manifest V3 once a compatible plug-in build exists (see
  `docs/cryptopro-csp-lite-plan.md` → Future goals).
- The diagnostic patches (`npcades`/`cades`) and the Go probes were owner-side / in
  `/tmp` only and are **not** committed (no CryptoPro binaries in Git).
