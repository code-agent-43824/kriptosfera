# Repository review — Kriptosfera (June 2026)

Reviewer pass: clarity of goal, code quality, documentation, achieved result.
Scope: a focused, not-exhaustive review meant to guide the next (Opus) development
pass. Snapshot facts: ~3.0k Go LOC (non-test) + ~1.4k test LOC, 52 tests, 41 Go
files, 13 doc files; `gofmt`/`go vet` clean; Windows + remote builds green.

## 1. Goal clarity — strong

The "what and why" is unambiguous. `CLAUDE.md` + `README.md` + `docs/project-summary.md`
state the product crisply: a single Windows `.exe` Go launcher that prepares a pinned
Chromium + CryptoPro CAdES plug-in (Mini CSP) payload and opens an isolated browser at
a target page, with the end goal of a Rutoken test signature **without a system CryptoPro
CSP**. The two payload modes (embedded vs remote) and the MV2/Chrome-138 compatibility
profile are well motivated. A newcomer can understand the project in ~15 minutes.

Minor: the headline goal ("works on a clean machine, no MSI") is currently **blocked by
a vendor bug** (see `docs/cryptopro-portable-plugin-findings.md`). The README still reads
as if stage 6 is an integration task; it should state plainly at the top that the
clean-machine path is **vendor-blocked** and that the MSI-installed path works. Right now
that nuance lives only in deeper docs.

## 2. Code quality — good, idiomatic, well-tested

Strengths:
- Clean layering: `cmd/` thin entry → `internal/bootstrap` (ordered pipeline) →
  `internal/config` (two config layers) → `internal/logging`. Easy to follow.
- Consistent, well-chosen idioms: `*_windows.go` / `*_other.go` build-tag splits;
  injectable function vars for tests (`registerNativeMessagingHost`,
  `writeCryptoProTrustedSites`); state-file gating + atomic rename + ready markers for
  every prepared component. The new trusted-sites feature followed these patterns
  exactly — that consistency is a real asset.
- Solid invariants in `validateAppConfig` (HTTPS diagnostics, safe `profileName`,
  origin allow-list, trusted-site shape) and zip-slip / MSI-pseudo-path guards in unzip.
- Good test ratio and meaningful tests (reuse/skip/state, validation, Windows-portable
  path edge cases). `go vet`/`gofmt` clean.

Watch items for the next pass:
- **`bootstrap.go` is becoming a god-file** (the `Run` pipeline + validation + arg
  building + dry-run all live there). Consider extracting the CryptoPro setup block
  (plugin → native messaging → trusted sites → diagnostics) into a small `setupCryptoPro`
  function so `Run` reads as a clean sequence of named steps.
- **Error-handling policy is inconsistent across registrations**: native-messaging
  failure is fatal, MV2 policy and trusted-sites failures are non-fatal-with-log. That's
  defensible, but the rationale should be documented in one place (a short comment block
  or doc) so future edits stay consistent.
- **`reg.exe` shelling**: functional and admin-free, but three near-identical reg writers
  exist now (native host, MV2 policy, trusted sites). A tiny internal `hkcuSet(key, name,
  type, data)` helper would cut duplication and centralize quoting/escaping (the
  REG_MULTI_SZ `\0` handling especially).
- **`go vet` only** is wired; consider adding `staticcheck` to CI for a cheap quality bump.
- The pinned **byte-patched DLL investigation artifacts** were correctly kept out of Git;
  keep it that way (no CryptoPro binaries / patches committed).

## 3. Documentation — excellent for analysis, slightly sprawling for onboarding

Strengths: genuinely high-quality engineering docs. The Mini-CSP root-cause writeups
(`cryptopro-portable-plugin-findings.md`, `cryptopro-csp-lite-plan.md`) are precise,
evidence-based, and would let a vendor or new engineer pick up the hardest problem cold.
The multi-agent `worklog.md` discipline (Planned/Done/Next) is paying off.

Gaps:
- **`worklog.md` is 770 lines** and mixes durable conclusions with day-by-day debugging.
  It's now more archive than handoff. Recommend: keep the last ~5 entries live, move the
  resolved CryptoPro-path saga into the findings doc (already mostly there), and archive
  the rest under `docs/worklog-archive.md`.
- **No architecture diagram of the runtime data/state flow** beyond prose. One small
  diagram (payload prepare → plugin extract → native messaging → trusted sites → launch)
  would shortcut onboarding. `docs/architecture.md` exists — extend it.
- README does not surface the current **blocked** status prominently (see §1).

## 4. Achieved result — substantial, with one external blocker

Delivered and working:
- End-to-end launcher (embedded + remote), payload integrity (SHA-256/size lock files),
  per-user HKCU native-messaging registration, MV2/Chrome-138 compatibility profile,
  slim embedded archive, shortened AppData layout, and now config-driven CAdES
  trusted-sites (confirmation dialog suppression — verified working).
- **Confirmed working on an MSI-installed machine**: extension + plugin + provider load,
  version/CSP correct, trusted-sites dialog suppression verified.

Not yet achieved (external):
- The flagship **clean-machine, no-MSI** signature. Root-caused to a CryptoPro
  `GetModuleFileName(0x10000000)` bug; byte-patching proven a dead end; **awaiting a
  vendor fix**. This is well documented and correctly escalated — not a code deficiency.
- The end-to-end **Rutoken `SignCades`** smoke test remains pending behind the above.

## 5. Top recommendations for the next pass (priority order)

1. **Surface the blocked status** at the top of README and `cryptopro-csp-lite-plan.md`
   (one paragraph): MSI-path works; clean-machine path is vendor-blocked.
2. **Refactor `bootstrap.Run`**: extract `setupCryptoPro(...)`; add the `hkcuSet` reg
   helper; document the fatal-vs-non-fatal registration policy in one place.
3. **Trim/anchor docs**: cap live `worklog.md`, archive the rest; extend
   `architecture.md` with one runtime-flow diagram.
4. **When the vendor build lands**: re-pin `build/cryptopro-plugin-lock.json`, run the
   clean-machine E2E (provider load + cert enumeration + `SignCades` with Rutoken), then
   start the MV3 / latest-Chromium migration tracked in the plan.
5. **Cheap quality**: add `staticcheck` to CI; consider a small integration test that
   asserts the full `Run` ordering on the dry-run (non-Windows) path.

Overall: a focused, well-engineered MVP scaffold with above-average documentation
discipline. The single biggest risk to "done" is external (the CryptoPro bug), and it is
handled responsibly. The main internal debt is mild and concentrated in `bootstrap.go`
and the growing worklog — both cheap to address.
