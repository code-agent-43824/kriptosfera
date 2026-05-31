# Phase 1 analysis — clean machine (no plugin installed)

Snapshot captured `2026-05-31T16:44:52Z` on `VIRTUALPC` (Windows 11 Pro), before
any CryptoPro plugin or Mini CSP was installed. This is the baseline that phases 2
and 3 diff against. Raw evidence: `registry/`, `files/`, `summary.txt`.

## 1. No `Crypto Pro` registry branch (as expected)

All four `Crypto Pro` keys are **absent** — confirmed by `reg query` returning a
non-zero exit (the script drops the `.txt` dump for absent keys, so there is no
`hk*-cryptopro.*` file in `registry/`):

| Key | State |
| --- | --- |
| `HKLM\SOFTWARE\Crypto Pro` | absent |
| `HKLM\SOFTWARE\WOW6432Node\Crypto Pro` | absent |
| `HKCU\SOFTWARE\Crypto Pro` | absent |
| `HKCU\SOFTWARE\WOW6432Node\Crypto Pro` | absent |

The Chrome native-messaging host is likewise absent in both hives:

| Key | State |
| --- | --- |
| `HKCU\...\NativeMessagingHosts\ru.cryptopro.nmcades` | absent |
| `HKLM\...\NativeMessagingHosts\ru.cryptopro.nmcades` | absent |

## 2. Baseline CryptoAPI providers (stock Windows only)

`HKLM\SOFTWARE\Microsoft\Cryptography\Defaults\Provider Types` and `...\Provider`
exist on stock Windows and contain **only Microsoft providers** — no GOST
provider types (no Type 75 / 80 / 81), which is what a CryptoPro install would
later add. Full dumps in `registry/hklm-capi-defaults*.{reg,txt}` (and the
`WOW6432Node` view).

Provider **Types** present (all Microsoft):

| Type | Name |
| --- | --- |
| 001 | Microsoft Strong Cryptographic Provider — *RSA Full (Signature and Key Exchange)* |
| 003 | Microsoft Base DSS Cryptographic Provider — *DSS Signature* |
| 012 | Microsoft RSA SChannel Cryptographic Provider — *RSA SChannel* |
| 013 | Microsoft Enhanced DSS and Diffie-Hellman Cryptographic Provider |
| 018 | Microsoft DH SChannel Cryptographic Provider |
| 024 | Microsoft Enhanced RSA and AES Cryptographic Provider — *RSA Full and AES* |

Provider **entries** present (all backed by in-box `rsaenh.dll` / `dssenh.dll` /
`basecsp.dll` under `%SystemRoot%\system32`): Microsoft Base / Strong / Enhanced
Cryptographic Provider v1.0, Enhanced RSA and AES, Base + Enhanced DSS and
Diffie-Hellman, DH/RSA SChannel, and Base Smart Card Crypto Provider.

The native and `WOW6432Node` views are captured separately so phases 2/3 can show
exactly which bitness view a GOST provider registers under (if any).

## 3. No `Crypto Pro` directory on disk

Both candidate roots are **absent**:

- `C:\Program Files (x86)\Crypto Pro` — absent
- `C:\Program Files\Crypto Pro` — absent

So there is no `CAdES Browser Plug-in`, no `Mini CSP`, and no `config.ini` /
`license.ini` to copy yet. The "Binary bitness" section of `summary.txt` is
correspondingly empty.

## Verdict

The machine is in a genuine clean state: no CryptoPro registry footprint, no
native-messaging host, and no CryptoPro files on disk. Only in-box Microsoft
CryptoAPI providers exist. This is the correct baseline for measuring what the
phase-2 (plugin, no flags) and phase-3 (`ADDMINICSP=1`) installs add.

## Tooling note

`tools/windows/snapshot-cryptopro-state.ps1` set `$ErrorActionPreference = "Stop"`
globally, which (under Windows PowerShell 5.1) promotes `reg.exe`'s stderr for an
absent key into a terminating `NativeCommandError` — aborting the clean snapshot
on the very first missing key. The error preference is now relaxed to `Continue`
only around the two native `reg.exe` calls (absent keys are an expected outcome
here), leaving `Stop` in force everywhere else. Behavior is otherwise unchanged,
so phases 2/3 remain directly diffable against this phase.
