# Phase 2 analysis — plugin installed WITHOUT flags

Snapshot captured `2026-05-31T16:56:48Z` on `VIRTUALPC`, after a default install of
the CryptoPro CAdES Browser Plug-in (`cadesplugin.exe`, no extra args). Diffed
against the phase-1 `clean` baseline. Raw evidence: `registry/`, `files/`,
`summary.txt`.

## 1. A `Crypto Pro` registry branch appeared — but only PKI service registration

Both the 64-bit and the WOW6432Node (32-bit) views gained a `Crypto Pro` key:

- `HKLM\SOFTWARE\Crypto Pro` (64-bit view) — minimal: `cpoids1 = 0x1` (DWORD).
- `HKLM\SOFTWARE\WOW6432Node\Crypto Pro` (32-bit view) — the real content:
  - values `pkimgmt.ru = 0x1`, `cpoids1 = 0x1`
  - `…\OCSPAPI\2.0` → `ProductID = 0A202U003000ECWRRLMFUU2WK`, `Version = 2.0`
  - `…\TSPAPI\2.0`  → `ProductID = TA200G003000ECWRRLNEBTDVV`, `Version = 2.0`

These are the **PKI runtime** components of the browser plug-in (OCSP / TSP
clients). There is **no `AppPath`, no `CurrentVersion`, and no CSP / GOST provider
key** — i.e. nothing here registers a cryptographic *provider*. (HKCU has no
`Crypto Pro` branch at all.) The `ProductID`s are committed verbatim as part of the
registry dump (the license is expired / non-secret, per the owner).

## 2. No GOST providers were registered (the key finding)

`HKLM\…\Cryptography\Defaults\Provider` and `…\Provider Types` (and their
`WOW6432Node` views) are **byte-identical to the clean phase** — verified with
`Compare-Object` across all four dumps (no diff). So a flagless plug-in install
adds **no** GOST provider type (no 75 / 80 / 81) and no GOST provider entry. The
provider set is still the stock Microsoft list from phase 1.

This is consistent with the investigation's premise: without Mini CSP activation
(and without a system CSP), there is no GOST provider to enumerate, so
`CSPName(80)` would still fail on this machine.

## 3. No `Mini CSP` folder

`C:\Program Files (x86)\Crypto Pro\` now exists and holds the `CAdES Browser
Plug-in\` directory (~33 files: `nmcades.exe`, `npcades.dll`, `cplib.dll`,
`cades.dll`, `xades.dll`, `cpasn1/asn1*`, `ocsp*/tsp*`, `mydss.dll`, the
`CryptoPro.*.cat`/`.manifest` set, `nmcades.json`/`nmcades_firefox.json`, etc.).
Full list with sizes + SHA-256 in `files/pf-x86-cryptopro.txt`.

There is **no `Mini CSP` subfolder**, hence no `config.ini` / `license.ini` to
copy (so `files/` holds only the directory listing). `C:\Program Files\Crypto Pro`
(64-bit) is absent. This matches the handoff's expectation that a default install
does not add Mini CSP.

## 4. Binary bitness

All present binaries are 32-bit, as the investigation found:

| Binary | Bitness |
| --- | --- |
| `CAdES Browser Plug-in\nmcades.exe` | x86 (32) |
| `CAdES Browser Plug-in\npcades.dll` | x86 (32) |
| `CAdES Browser Plug-in\cplib.dll` | x86 (32) |
| `Mini CSP\capi20.dll` | absent (no Mini CSP) |

## 5. Native-messaging host

`HKCU\…\NativeMessagingHosts\ru.cryptopro.nmcades` and the HKLM equivalent are
**still absent**. The installer ships `nmcades.json` / `nmcades_firefox.json` in
the plug-in directory but does not populate the standard Chrome
`NativeMessagingHosts` registry keys at install time (in our product the launcher
registers this host per-user).

## Diff vs phase 1 — what the flagless install added

- **Registry:** the `Crypto Pro` branch (64-bit `cpoids1`; 32-bit
  `pkimgmt.ru`/`cpoids1` + `OCSPAPI\2.0` + `TSPAPI\2.0`). **Nothing** changed
  under `Cryptography\Defaults\Provider*`.
- **Disk:** the entire `Program Files (x86)\Crypto Pro\CAdES Browser Plug-in`
  tree appeared. No `Mini CSP`.
- **Net:** PKI/plug-in runtime is registered; **no crypto provider and no Mini
  CSP**. Sets up the phase-3 question: does `ADDMINICSP=1` add the `Mini CSP`
  folder (and does it touch the registry at all, or only `config.ini`?).
