# Step 2.5 analysis — after standard uninstall (residue check)

Intermediate snapshot captured `2026-05-31T17:02:18Z` on `VIRTUALPC`, after the
owner uninstalled the phase-2 plug-in the standard way (Apps & Features). Purpose:
detect leftover "garbage" before reinstalling with `ADDMINICSP=1` in phase 3.
Raw evidence: `registry/`, `files/` (incl. `programdata-cryptopro.txt`),
`summary.txt`.

## Result: uninstall is clean, with one empty-folder exception

Within the snapshot script's scope, the machine returned to the clean state:

- **`Crypto Pro` registry branch — gone** (HKLM 64-bit and WOW6432Node both
  absent again). The phase-2 `OCSPAPI\2.0` / `TSPAPI\2.0` `ProductID` keys were
  removed.
- **`Program Files (x86)\Crypto Pro` — removed** entirely (so no
  `CAdES Browser Plug-in`, no binaries; `summary.txt` "Binary bitness" is empty).
- **CryptoAPI providers — unchanged**: `Defaults\Provider` and `Provider Types`
  (native + WOW6432Node) are **byte-identical to the clean phase** (verified with
  `Compare-Object`), so the uninstall left no GOST provider residue.
- **Native-messaging host** `ru.cryptopro.nmcades` — absent (as before).

## Broader residue sweep (beyond the script's fixed key list)

The snapshot script only inspects `Program Files` + a fixed registry key list, so
I additionally checked the places an uninstall typically misses (read-only —
details in `files/programdata-cryptopro.txt`):

| Location | After uninstall |
| --- | --- |
| `C:\ProgramData\Crypto Pro` | **EXISTS — leftover** (only child: `Installer Cache`, **empty**) |
| `…\AppData\{Roaming,Local,LocalLow}\Crypto Pro` | absent |
| COM ProgIDs `CAdESCOM.CPSigner/.Store/.About` (HKLM/WOW6432Node/HKCU Classes) | none |
| Uninstall entries matching "Crypto" (HKLM + WOW6432Node) | none |

**The only leftover is an empty `C:\ProgramData\Crypto Pro\Installer Cache`
folder skeleton** — standard ACL (SYSTEM/Administrators FullControl, Users
read+write). It holds no config, keys, or provider data, so it does not affect
provider enumeration. It will most likely be repopulated by the phase-3 install.

## Implication for phase 3

This is effectively a clean starting point for the `ADDMINICSP=1` reinstall. The
empty `ProgramData\Crypto Pro\Installer Cache` is the only carry-over and is
inert. Note the snapshot script does not capture `ProgramData\Crypto Pro`; if a
later phase needs it as formal evidence, the script's path list should be
extended (left as-is here to keep phases 1–3 diffable against the same tool).
