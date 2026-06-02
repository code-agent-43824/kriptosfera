# Worklog — handoff log between agents

Several agents work on this repo, sometimes in parallel. Keep a short entry per
chunk of work: **Planned / Done / Next**, newest on top. Document first, then code.
For deeper context see `docs/cryptopro-csp-lite-plan.md` and `CHANGELOG.md`.

---

## 2026-06-02 — Re-pin the remote payload to the internal-csp build

**Context:** the previous chunk changed `app-config.json` `startUrl` to the
internal-csp page and pushed it (`23a3b8c`). CI went green: `build-windows`
(run 26815886534) and `build-payload` (run 26815886446) both succeeded. The
**embedded** launcher bakes the config from the commit, so it already opens the
internal-csp page. The **remote** launcher, however, downloads the payload pinned
by `build/payload-lock.json`, which still pointed at the *old* payload
(`9575882…`, official-demo startUrl). So a remote launcher would still open the
old page.

**Planned:** re-pin `build/payload-lock.json` to the payload that `build-payload`
just published from `23a3b8c`
(SHA `9b2a00bb8ba09f59f973691c9a26cdb0bd757795f75533ff3bb971cb83501c48`,
size `173037165`).

**Done:** verified the new payload is live on the server (HTTP 200, content-length
and `payload.json` SHA/size match the build log), then updated `payload-lock.json`
`sha256`, `size`, `url`, and `metadataUrl` to the new artifact. Updated CHANGELOG.
Embedded and remote launchers now both open the internal-csp page.

**Next:** same as below — E2E on a clean Windows machine with a Rutoken. The
remote launcher can now be used for that check too (not just embedded).

---

## 2026-06-02 — Point launcher startUrl at the internal-csp test page

**Context:** the legacy MV2 stack is integrated and CI-green (plug-in `2.0.15000`
+ MV2 extension `1.2.13` + Chrome 138 + `ExtensionManifestV2Availability=2` policy;
verified the 2.0.15000 bundle on the server matches the lock SHA/size and contains
all required files). The launcher's `startUrl` still pointed at the official
CryptoPro demo page, which does **not** set `EnableInternalCSP`, so a run there
would not exercise the bundled Mini CSP. The owner confirmed the normal
`internal-csp` page works.

**Planned:** set `payload-template/config/app-config.json` `startUrl` to
`https://mescheryakov.pro/kriptosfera/cryptopro-cades-test/internal-csp/demopage/cades_bes_sample.html`
and set `allowedOrigins` to `https://mescheryakov.pro` (startUrl must be inside
allowedOrigins per `validateAppConfig`).

**Done:** updated `app-config.json` (startUrl + allowedOrigins); kept
`windowMode: app` (standalone app window) and diagnostics off. Updated the plan
doc and CHANGELOG.

**Next:**
- E2E on a clean Windows machine: download the embedded launcher from the latest
  `build-windows` run, insert a Rutoken with a CryptoPro-format GOST cert, and
  confirm certificate enumeration + `SignCades` on the internal-csp page.
- If it works, this closes MVP stage 6 (clean-machine signing). Then consider
  branding/customer startUrl as a config, and track the future MV3/latest-Chromium
  migration (see `docs/cryptopro-csp-lite-plan.md` → Future goals).
- Blocked/needs owner: nothing for this step.
