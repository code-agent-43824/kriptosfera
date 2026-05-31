# Conclusions — CryptoPro Mini CSP install states (Windows, by observation)

Cross-phase summary of the snapshots in this directory, captured `2026-05-31` on a
clean `VIRTUALPC` (Windows 11 Pro, no system CryptoPro CSP). Each phase used the
same `tools/windows/snapshot-cryptopro-state.ps1`; all claims below are backed by
`reg export`/`reg query`, recursive file listings with SHA-256, and verbatim
`config.ini`/`license.ini` copies. Phases:

| Phase | Dir | State |
| --- | --- | --- |
| 1 | `clean/` | no plug-in installed |
| 2 | `installed-noflags/` | `cadesplugin.exe`, default install |
| 2.5 | `uninstalled/` | after standard uninstall (residue check) |
| 3 | `installed-addminicsp/` | reinstall with `-cadesargs "ADDMINICSP=1"` |

## 1. What each state writes to the registry

- **Clean:** no `Crypto Pro` branch anywhere (HKLM/HKCU, native + WOW6432Node); no
  `ru.cryptopro.nmcades` host; only stock Microsoft CryptoAPI providers
  (Provider Types 001/003/012/013/018/024).
- **Flagless install:** adds a `Crypto Pro` branch that is **PKI-only** —
  `HKLM\SOFTWARE\Crypto Pro` (`cpoids1`) and `HKLM\SOFTWARE\WOW6432Node\Crypto Pro`
  (`pkimgmt.ru`, `cpoids1`, `OCSPAPI\2.0` + `TSPAPI\2.0` with `ProductID`/
  `Version`). **No CSP/provider key, no `AppPath`/`CurrentVersion`.**
  `Cryptography\Defaults\Provider(+Types)` is byte-identical to clean — **no GOST
  provider is registered.**
- **`ADDMINICSP=1`:** **adds no registry keys at all** — every dump is
  byte-identical to the flagless phase (verified with `Compare-Object`, including
  both `Crypto Pro` keys). The flag is a filesystem-only change.
- **Uninstall:** removes the `Crypto Pro` branch, the COM `CAdESCOM.*` ProgIDs,
  the Uninstall entry, and the whole `Program Files` tree. Only an **empty**
  `C:\ProgramData\Crypto Pro\Installer Cache` folder skeleton remains.

## 2. What `ADDMINICSP=1` adds on disk (and the registry answer)

On disk it adds two `Mini CSP` trees and **nothing in the registry**:

- `C:\Program Files (x86)\Crypto Pro\CAdES Browser Plug-in\Mini CSP\` (32-bit):
  `config.ini`, `capi20.dll`, `cpcspi.dll`, plus `capi10/cplib/cpasn1/asn1*/bio/
  fat12/dsrf/cpsuprt/cpui/cpconfig.exe`, token providers `rutoken.dll`/
  `jacarta.dll`/`safenet.dll`/`pcsc.dll`, and `license.ini`.
- `C:\Program Files\Crypto Pro\CAdES Browser Plug-in\Mini CSP\` (64-bit, new):
  same set, 64-bit, with `config64.ini` (byte-identical to the 32-bit
  `config.ini`). No 64-bit root plug-in binaries — only the `Mini CSP` subfolder.

**Registry: untouched** (see §1). This settles, by observation, that
`ADDMINICSP=1` is a config.ini-only model.

## 3. Where provider registration lives

**Only in `Mini CSP\config.ini` — never in the Windows registry.** Even with Mini
CSP installed, `HKLM\…\Cryptography\Defaults\Provider(+Types)` stays exactly as on
a clean machine. `config.ini` carries CryptoPro's own `Defaults\Provider` table:

| Provider | `Image Path` | `Type` |
| --- | --- | --- |
| Crypto-Pro ECDSA and AES CSP | `cpcspi.dll` | 16 |
| Crypto-Pro Enhanced RSA and AES CSP | `cpcspi.dll` | 24 |
| Crypto-Pro GOST R 34.10-2001 CSP | `cpcspi.dll` | 75 |
| Crypto-Pro GOST R 34.10-2012 CSP | `cpcspi.dll` | 80 |
| Crypto-Pro GOST R 34.10-2012 Strong CSP | `cpcspi.dll` | 81 |

The native host is 32-bit, so it loads the **32-bit** `Mini CSP\capi20.dll`
(module-relative) and reads **`config.ini`**; `config64.ini` is the byte-identical
64-bit twin.

## 4. Implications for the portable launcher

To make the **bundled** Mini CSP enumerate providers **without a full MSI
install**, we must replicate only the on-disk layout — and the snapshots confirm
the bundle already has everything:

- Ship `Mini CSP\` **next to the 32-bit host** (module-relative path), containing
  `capi20.dll` + `cpcspi.dll` + their deps + `config.ini` + `license.ini`. Our
  `cryptopro-plugin.zip` already matches a real `ADDMINICSP=1` install (the
  phase-3 file set + hashes can be diffed against the bundle to prove byte
  identity).
- **Do NOT** write to the Windows registry, **do NOT** flatten the `Mini CSP`
  folder, and **do NOT** build a wrapper host — proven unnecessary here.
- License is self-contained in `Mini CSP\license.ini` (`ProductID` under GUID
  `{50F91F80-…}`); no separate license step.

So provider *registration* is a solved, no-op problem for the launcher. The
**only** remaining blocker for activation is getting `npcades.dll` to actually
load `Mini CSP\capi20.dll` — i.e. delivering `cadesplugin.EnableInternalCSP = true`
**early enough** (the `internal-csp-early` hypothesis), and, if that is not
sufficient, the `capi20.dll` load + dependency search path (ProcMon on
`nmcades.exe`: `Load Image` for `Mini CSP\capi20.dll`, `CreateFile` for
`config.ini`).

## 5. Next experiment

On this `ADDMINICSP=1` machine, open the deployed test pages in plain Chrome and
record whether providers enumerate:

- `…/cryptopro-cades-test/internal-csp/demopage/cades_bes_sample.html`
- `…/cryptopro-cades-test/internal-csp-early/demopage/cades_bes_sample.html`

If `internal-csp-early` enumerates providers and `internal-csp` does not →
hypothesis A (flag timing) is confirmed; port the early-flag pattern to the
extension/diagnostics. If both stay silent → ProcMon `nmcades.exe` for
`Load Image` on `Mini CSP\capi20.dll` and `CreateFile` on `config.ini`/`asn1*.dll`
(look for `NAME NOT FOUND` / `PATH NOT FOUND`). The hosted
`diagnostics/diagnostics.html` already sets the flag early and prints an A/B/C
verdict.

## 6. Diagnostics run (2026-05-31) — hypothesis A refuted

Ran the public `diagnostics.html` in **regular Chrome** against this
`ADDMINICSP=1` **system** install (not the bundled launcher). Observed:

- **Plugin works:** `cadesplugin ready`; `CAdESCOM.About` →
  `PluginVersion`/`Version` = `2.0.15700`; `CAdESCOM.Store` opens `My`
  (`Certificates.Count = 0`, no token inserted).
- **Flag delivered early:** `EnableInternalCSP` = `true` *before*
  `cadesplugin_api.js` (`after-set (pre-api inline)` at +0 ms) and still `true`
  at +82 ms / after `cadesplugin ready`.
- **Providers absent:** `About.CSPName`/`CSPVersion` for **75 / 80 / 81** →
  **`0x80090017` (`NTE_PROV_TYPE_NOT_DEF`)**.
- **Anomaly:** `extension version response timed out after 3000 ms` although
  CAdES `CreateObjectAsync` calls succeeded.

**Result: hypothesis A (flag timing) is refuted** — the flag is delivered early
and correctly, yet Mini CSP providers never enumerate. Verdict is **B/C**:
`npcades` did not load `Mini CSP\capi20.dll` (or `asn1*.dll`/`config.ini` deps),
or an integrity self-test failed. Because this is the *official* `ADDMINICSP=1`
install, the gap is in the internal-CSP activation mechanism itself, not in our
repackaging. The early-flag test variant (`internal-csp-early`) is therefore not
the fix.

A read-only module probe of the live `nmcades.exe`
(`tools/windows/inspect-cryptopro-modules.ps1`) then settled the next layer:
**`Mini CSP\capi20.dll` is loaded** (3 of 4 hosts) with its `asn1*`/`cpsuprt`
deps, and one host also loaded `cpcspi.dll` (the `config.ini` `Image Path`). So
the flag reached native, the search path is fine, and `config.ini` was read —
**hypothesis B is refuted too** (evidence:
`installed-addminicsp/files/nmcades-loaded-modules.txt`). The remaining
candidates for `0x80090017` are (C) provider registration / self-test not
completing, or `About.CSPName` not seeing an in-process internal CSP
(false-negative); the decisive test is an actual sign with a GOST token. See
`docs/cryptopro-csp-lite-plan.md` → "Diagnostics run result (2026-05-31)".
