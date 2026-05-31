# Mini CSP install-state snapshots

Factual before/after snapshots of the Windows registry and
`Program Files (x86)\Crypto Pro` across three CryptoPro plugin install states,
captured by `tools/windows/snapshot-cryptopro-state.ps1`. Task and analysis
instructions: [`../handoff-windows-minicsp-snapshots.md`](../handoff-windows-minicsp-snapshots.md).

Phases (one subdirectory each, created when the snapshot runs):

- `clean/` — clean Windows, no plugin installed.
- `installed-noflags/` — plugin installed with default options.
- `installed-addminicsp/` — plugin reinstalled with `ADDMINICSP=1`.

Each phase contains `registry/`, `files/`, `summary.txt`, and an `ANALYSIS.md`.
A cross-phase `CONCLUSIONS.md` summarizes what the diffs prove about where
provider registration lives (registry vs `Mini CSP\config.ini`).
