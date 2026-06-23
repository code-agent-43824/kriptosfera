# RUNBOOK — diagnose why Mini CSP shows only `csp` carriers (FKC / PKCS#11 invisible)

Audience: a **Claude Code session running on a real Windows VM with computer-use**,
driving the already-running Kriptosfera launcher. Goal: determine, with evidence,
**why the bundled Mini CSP enumerates only the passive `csp` certificate format**
and not the **FKC** and **PKCS#11-active** formats of a Rutoken ЭЦП — when the full
**system** CryptoPro CSP on the same machine sees all three.

This is a **diagnostic** runbook, not a build task. You are deciding between two
hypotheses:

- **(A) Config / placement problem on our side** — the FKC/PKCS#11 carrier entries
  (or the reader DLLs) are missing, malformed, or sitting in a Mini CSP folder that
  the runtime never loads. Fixable by us.
- **(B) Mini CSP feature gap** — the shipped Mini CSP core simply does not implement
  these reader devices, no matter how correct the config is. → vendor bug report.

## ⛔ Guardrail boundary — stay inside it

You are doing **interop configuration + passive observation** of licensed software
on the owner's own VM. That is fine. Specifically:

- ✅ Allowed: run `ListDLLs` / `Process Monitor` / `reg export`, read `config.ini`,
  copy vendor DLLs the machine already has, append carrier sections to `config.ini`
  using CryptoPro's **own documented** config mechanism, restart the launcher,
  observe the demo page.
- ⛔ Do **NOT**: disassemble, byte-patch, or otherwise modify any CryptoPro/Rutoken
  binary; defeat licensing/anti-tamper; or bypass the user-confirmation dialog by any
  hack. If the diagnosis ever appears to *require* opening or modifying a vendor
  binary, **stop** — that is the signal the road ends on our side and the result is a
  vendor bug report, not a workaround. Report back instead of proceeding.

## Background you can rely on (don't re-derive)

- The only configuration where the provider currently loads at all is **plug-in
  installed via MSI with `ADDMINICSP=1`** (portable-from-our-folder is blocked by a
  separate vendor bug — `docs/cryptopro-portable-plugin-findings.md`). So Mini CSP
  most likely loads from `C:\Program Files (x86)\Crypto Pro\CAdES Browser Plug-in\Mini CSP\`,
  **not** from our `%LOCALAPPDATA%\Kriptosfera\...` extraction. Confirm this in Phase 1
  before changing anything — it is the leading explanation for "I copied files
  everywhere and nothing worked": the edited copies were never the loaded ones.
- The crypto work happens inside the native-messaging host **`nmcades.exe`** (PE32 /
  x86), spawned by Chrome. Mini CSP DLLs (`cpcspi.dll`, `rutoken.dll`, …) load into
  that process. So `nmcades.exe` is the inspection target.
- The Windows full CSP stores carriers/devices/readers in the **registry**
  (`HKLM\SOFTWARE\Crypto Pro\Cryptography\CurrentVersion\…`). Mini CSP is the same CSP
  without the registry: `config.ini` mirrors that registry tree section-for-section
  (`[KeyDevices\…]` = a registry subkey). So the working system CSP's registry is the
  **proven Windows ground-truth** to compare our `config.ini` against (better than the
  Linux source we adapted from).
- The carrier config fragment and the DLL roles are documented in
  `docs/cryptopro-rutoken-fkc-pkcs11.md`. The current `2.0.15000` vendor `config.ini`
  already contains `rutokenfkc` / `rutokenfkc_nfc`; only the PKCS#11 device
  `cryptoki_rutoken` is genuinely missing.

## Tools to stage (no admin needed for these)

- **Sysinternals ListDLLs** — download `https://download.sysinternals.com/files/ListDlls.zip`,
  unzip anywhere (e.g. `C:\Tools\`). First run wants `-accepteula`.
- (Optional, only if Phase 1 is inconclusive) **Process Monitor** —
  `https://download.sysinternals.com/files/ProcessMonitor.zip`. Can run headless:
  `Procmon.exe /AcceptEula /Quiet /Minimized /BackingFile C:\Tools\trace.pml`, later
  `Procmon.exe /OpenLog C:\Tools\trace.pml /SaveAs C:\Tools\trace.csv`.

## Where to get the 3 reader DLLs — from the machine, not the network

For Phase 3 you need x86 `cpfkc.dll`, `cryptoki.dll`, `rtPKCS11ECP.dll`. **Do not
download/pin them here** — this is a local experiment, so copy the ones the VM already
has after the Phase-2 installs:

- `cpfkc.dll`, `cryptoki.dll` → from the installed full CryptoPro CSP, typically
  `C:\Program Files (x86)\Crypto Pro\CSP\`.
- `rtPKCS11ECP.dll` → from the installed Rutoken drivers (Rutoken install dir / system32
  for the x86 build). Take the **32-bit** build (host is PE32).

(These are reference-only for the experiment; the committed `build/rutoken-fkc-lock.json`
is what ships. Don't commit any of these binaries.)

---

## Phase 1 — what actually loads (no admin, no token needed)

The launcher is already running. On the internal-csp demo page, trigger provider load
/ certificate listing so `nmcades.exe` exists and Mini CSP is loaded, leave it open.

```powershell
# 1. Is the host process alive?
Get-Process nmcades -ErrorAction SilentlyContinue | Format-Table Id, Path

# 2. Which modules are loaded, and FROM WHERE (the decisive question)
C:\Tools\listdlls.exe -accepteula nmcades > C:\Tools\nmcades-dlls.txt
# inspect: full path of cpcspi.dll  -> its folder IS the authoritative Mini CSP dir
#          presence/path of rutoken.dll, capi20.dll
#          presence of cpfkc.dll / cryptoki.dll / rtPKCS11ECP.dll (expected: ABSENT now)
```

Record from `nmcades-dlls.txt`:
- **Authoritative Mini CSP folder** = the directory of the loaded `cpcspi.dll`
  (Program Files vs our LOCALAPPDATA). *This tells you where every later edit must go.*
- Whether any of the three target DLLs are already loaded (they should not be yet).

**If `cpcspi.dll` loads from `Program Files (x86)\...\Mini CSP\`** → confirmed: our
LOCALAPPDATA overlay is dead weight in this setup, and Phase 3 edits must target the
Program Files copy (needs an elevated shell — see below). This very likely already
explains the user's failed manual attempts.

---

## Phase 2 — capture the proven-good reference (admin steps done by the user)

Ask the user (admin-gated; you cannot do these):
1. Ensure **Rutoken drivers** installed and the **token inserted** (it must hold certs
   in all three formats: csp, pkcs11, fkc).
2. Ensure the **full system CryptoPro CSP** is installed (the one that already sees all
   three formats).

Then you capture the ground-truth, no admin needed to read:

```powershell
reg export "HKLM\SOFTWARE\Crypto Pro\Cryptography\CurrentVersion" C:\Tools\sys-csp.reg /y
reg export "HKLM\SOFTWARE\WOW6432Node\Crypto Pro\Cryptography\CurrentVersion" C:\Tools\sys-csp-wow.reg /y
```

In those exports, locate the working **`KeyDevices\cryptoki...`**, **`KeyCarriers\rutokenfkc...`**
and reader subkeys. This is the reference structure/values our `config.ini` must match.
Compare them section-for-section against the authoritative Mini CSP `config.ini` from
Phase 1. Note every difference (key names, `Group`, `DLL`, `pkcs11_dll`, ATR/mask).

> Optional sanity check that the readers really engage in the **system** CSP: with the
> token in, the full CSP's own tooling enumerates all three. That confirms the token and
> the reference are good before blaming Mini CSP.

---

## Phase 3 — the decisive test (edit the authoritative Mini CSP)

Target = the Mini CSP folder identified in Phase 1 (likely Program Files → **needs the
elevated shell the user granted; if not granted, dictate these steps to the user**).

1. **Back up first:** copy `<authoritative Mini CSP>\config.ini` to `config.ini.bak`.
2. **Place the three x86 DLLs** sourced in the section above:
   - `cpfkc.dll`, `cryptoki.dll` → `<authoritative Mini CSP>\`
   - `rtPKCS11ECP.dll` → the **`CAdES Browser Plug-in\`** dir (next to `nmcades.exe`),
     because `pkcs11_dll = "rtPKCS11ECP.dll"` is loaded by bare name (process dir). If
     unsure, also drop a copy in `Mini CSP\` (harmless).
3. **Append the missing carrier config** to `<authoritative Mini CSP>\config.ini`,
   preserving **Windows-1251** encoding. The FKC block is usually already present in
   `2.0.15000`; the `cryptoki_rutoken` device is the one that's missing. Use the exact
   fragment from `docs/cryptopro-rutoken-fkc-pkcs11.md`. Idempotent append:

   ```powershell
   $enc = [System.Text.Encoding]::GetEncoding(1251)
   $cfg = "<authoritative Mini CSP>\config.ini"
   $text = [System.IO.File]::ReadAllText($cfg, $enc)
   $pkcs11 = @"

[KeyDevices\cryptoki_rutoken]
"DLL"="cryptoki.dll"
"Group"=1
[KeyDevices\cryptoki_rutoken\"PNP cryptoki"]
[KeyDevices\cryptoki_rutoken\"PNP cryptoki"\Default]
pkcs11_dll = "rtPKCS11ECP.dll"
"@
   if ($text -notmatch '\[KeyDevices\\cryptoki_rutoken\]') {
     [System.IO.File]::WriteAllText($cfg, $text + "`r`n" + $pkcs11, $enc)
   }
   # If Phase-1/Phase-2 showed rutokenfkc is ALSO absent from this config.ini,
   # append the FKC block from docs/cryptopro-rutoken-fkc-pkcs11.md the same way.
   ```

4. **Fully restart** the launcher (close Chrome + any lingering `nmcades.exe`), reopen
   the internal-csp demo page, re-trigger certificate enumeration with the token in.
5. **Re-snapshot loaded modules** to confirm the readers now actually bind:

   ```powershell
   C:\Tools\listdlls.exe -accepteula nmcades > C:\Tools\nmcades-dlls-after.txt
   # expect cpfkc.dll / cryptoki.dll / rtPKCS11ECP.dll to now appear, with full paths
   ```

---

## Decision tree (what the result means)

- **FKC / PKCS#11 certs now enumerate** → hypothesis **A**. The real bug was
  **placement** (our overlay landed in a folder the runtime never loads) and/or a
  missing section. Fix: make the launcher overlay target the folder the runtime
  actually loads, and ensure `cryptoki_rutoken` is present. **No vendor report.**
- **`ListDLLs` shows the reader DLLs are loaded, but the certs still don't enumerate**
  → reader binds but device init fails. Capture a ProcMon trace of the load + compare
  the failing calls against the working system-CSP trace; report findings before
  concluding.
- **`config.ini` matches the proven system-CSP registry 1:1, DLLs are present in the
  authoritative folder, yet `ListDLLs` shows Mini CSP never even loads `cpfkc.dll` /
  `cryptoki.dll`** → hypothesis **B**: the Mini CSP core ignores these device sections
  → **genuine vendor defect, file the bug report** with: the `ListDLLs` before/after,
  the `config.ini` diff vs the working registry export, and the note that the full CSP
  of the same line sees all three formats on the same token.

## Report back

Write findings into `docs/worklog.md` (and summarize to the user): authoritative Mini
CSP path, whether our overlay was even in the load path, the `config.ini`-vs-registry
diff, the before/after `ListDLLs`, and which branch of the decision tree we landed on.
Do not commit any vendor binaries or `.reg`/trace dumps containing them.
