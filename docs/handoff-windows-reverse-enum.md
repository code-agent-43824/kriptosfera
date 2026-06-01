# Handoff: Windows reverse enumeration test (config-view vs registry-view)

Adressee: the local Windows Claude Code session (git + `GH_TOKEN`). Author: the
web-session diagnostic agent (Linux only).

## Why

We proved on a clean Linux CryptoPro install (no registry at all) that the exact
CryptoAPI calls the plugin uses — `CryptEnumProviderTypes` / `CryptEnumProviders`
/ `CryptGetDefaultProvider` — enumerate GOST providers 75/80/81 straight from
`config64.ini` (strace-confirmed), and that removing the config reproduces the
**`0x80090017`** we see on Windows. Evidence + analysis:
`docs/minicsp-snapshots/linux-enum-control/`.

But static export analysis of the **Windows** Mini CSP shows the OS is different:

- `Mini CSP\cpcspi.dll` exports the CSP engine (`CPAcquireContext`, `CPSignHash`…).
- `Mini CSP\capi20.dll` exports only high-level `Cert*`/`CryptMsg*` and does
  **not** export the base `CryptAcquireContext`/`CryptEnumProviderTypes`/
  `CryptGetDefaultProvider`. On Windows those base calls are **Microsoft's
  `advapi32.dll`**, which reads providers from the **registry** — where Mini CSP
  is not registered (`ADDMINICSP=1` writes nothing to the registry).

Hypothesis to confirm on Windows: **`About.CSPName(80)=0x80090017` is by OS
architecture, not a fixable config-path bug.** CryptoPro's own (config-based)
view should see the providers; the Microsoft (registry-based) view should not.

## Task

On the `ADDMINICSP=1` machine, run the prepared script (no compiler needed):

```powershell
cd <repo>; git pull origin main
./tools/windows/reverse-enum-test.ps1
```

It writes `docs/minicsp-snapshots/windows-reverse-enum/`:

- `cpconfig-view-type.txt` — bundled `Mini CSP\cpconfig.exe -defprov -view_type`
  run **from** the `Mini CSP` folder (CryptoPro config-based enumeration).
- `cpconfig-view-type-elsewhere.txt` — same, run from `C:\Windows` (config-path
  probe: does cpconfig still find `config.ini` from another cwd?).
- `cpconfig-view-prov80.txt` — `-defprov -view -provtype 80`.
- `certutil-csplist.txt` — Microsoft registry view of installed CSPs.
- `advapi32-enum.txt` — `advapi32` `CryptEnumProviderTypes` / `CryptGetDefaultProvider(80)`
  via P/Invoke (the OS path the plugin's `About.CSPName` uses).
- `summary.txt` — side-by-side verdict.

## Read the result

- **If `cpconfig` lists type 80 but `certutil`/`advapi32` do not** → split
  confirmed. Mini CSP providers live only in `config.ini`; the registry-based
  enumeration the plugin uses can't see them. Then `About.CSPName(80)` is the
  **wrong success signal** and the path forward is the **in-process** route
  (open a Mini CSP container + `SignCades` via `npcades → cpcspi`), which needs a
  real **GOST token** to validate end-to-end.
- **If `cpconfig-view-type-elsewhere.txt` is empty/fails** while the in-folder one
  works → cpconfig (and likely capi20) resolves `config.ini` relative to **cwd**,
  not its own module path. That would be a concrete, fixable lever (ensure the
  Mini CSP dir is the working directory / config path for `nmcades.exe`).

## Commit

```powershell
git add docs/minicsp-snapshots/windows-reverse-enum tools/windows/reverse-enum-test.ps1
git commit -m "docs: windows reverse enumeration test (config vs registry view)"
git push origin HEAD:main
```

Then report the three views (cpconfig / certutil / advapi32) back. That settles
whether enumeration is architecturally impossible for an unregistered Mini CSP on
Windows (pivot to the in-process sign test) or whether it's a config-path issue
we can fix in the launcher.

## Next decisive experiment (likely)

If the split is confirmed: get a **Rutoken with a GOST container** and, on the
`ADDMINICSP=1` machine in plain Chrome, run the standard demo-page sign flow
(`internal-csp` page). If `SignCades` succeeds via the in-process Mini CSP, the
internal-CSP activation is effectively working and the only fix left is to stop
gating our diagnostics/UX on the (architecturally empty) `About.CSPName`
enumeration. If it fails, capture the exact error for the next step.
