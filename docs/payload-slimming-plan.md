# Payload slimming plan

## Scope

This note records the approved cleanup direction before changing payload contents.
It intentionally does **not** update `build/payload-lock.json` or publish a new
payload. Rebuild, lock update, GitHub Actions review, and Windows smoke/E2E checks
are deferred to the next chunk.

## Current payload snapshot

Pinned remote payload:

```text
url: https://mescheryakov.pro/kriptosfera/payloads/win64/demo/0.5.0/9b2a00bb8ba09f59f973691c9a26cdb0bd757795f75533ff3bb971cb83501c48/payload.zip
sha256: 9b2a00bb8ba09f59f973691c9a26cdb0bd757795f75533ff3bb971cb83501c48
size: 173037165
```

The public payload re-download verified against the lock file on 2026-06-04.

The archive is almost entirely Chromium:

```text
389.7 MB raw / 172.9 MB zip  chromium/
0.12 MB raw / 0.07 MB zip    extensions/
0.03 MB raw / 0.01 MB zip    diagnostics/
```

The CryptoPro Browser Plug-in is **not** inside `payload.zip`. It is a separate
archive pinned by `build/cryptopro-plugin-lock.json` and embedded into launcher
variants by the Windows build.

Largest Chromium payload areas:

```text
255.2 MB raw / 116.1 MB zip  chrome.dll
40.3 MB raw / 10.3 MB zip    locales/
25.8 MB raw / 10.2 MB zip    dxcompiler.dll
11.7 MB raw / 10.3 MB zip    resources.pak
10.5 MB raw / 4.6 MB zip     icudtl.dat
7.9 MB raw / 3.2 MB zip      libGLESv2.dll
5.3 MB raw / 2.1 MB zip      vk_swiftshader.dll
4.9 MB raw / 2.1 MB zip      D3DCompiler_47.dll
4.9 MB raw / 2.0 MB zip      setup.exe
3.5 MB raw / 1.7 MB zip      elevated_tracing_service.exe
3.1 MB raw / 1.5 MB zip      chrome.exe
2.2 MB raw / 1.1 MB zip      elevation_service.exe
1.8 MB raw / 0.7 MB zip      chrome_pwa_launcher.exe
1.8 MB raw / 1.1 MB zip      hyphen-data/
1.7 MB raw / 0.8 MB zip      notification_helper.exe
```

## Safe first pass

Add a dedicated PowerShell slimming step after `prepare-chromium.ps1`, disabled or
easy to revert by commit history, that removes only low-risk Chromium extras:

- keep only required locales: `ru`, `en-US`, and optionally `en-GB`;
- remove most `hyphen-data/`, keeping `hyph-ru.hyb`, `hyph-en-us.hyb`, and
  `manifest.json` if Chromium expects the directory to exist;
- remove `setup.exe`;
- consider removing `chrome_pwa_launcher.exe`, `elevated_tracing_service.exe`,
  `elevation_service.exe`, and `notification_helper.exe` only after a local
  Chrome startup smoke test shows app-mode launch still works.

Expected first-pass win is mostly from locales plus a few helpers: about
`12-16 MB` compressed, depending on which helper binaries survive smoke testing.

## Do not touch in the first pass

Do not remove these before a Windows startup smoke test and a separate rollback
commit are ready:

- `chrome.dll`, `chrome.exe`, `chrome_*.pak`, `resources.pak`, `icudtl.dat`;
- GPU/graphics stack: `libEGL.dll`, `libGLESv2.dll`, `D3DCompiler_47.dll`,
  `dxcompiler.dll`, `dxil.dll`, `vulkan-1.dll`, `vk_swiftshader.dll`,
  `vk_swiftshader_icd.json`;
- `v8_context_snapshot.bin`;
- extension files or diagnostics files.

The GPU/SwiftShader/Vulkan files look tempting, but removing them can produce a
non-obvious Chrome startup or rendering failure. Treat them as a second pass only.

## Verification required before lock update

The next chunk that actually changes payload contents must:

- rebuild payload on Windows or in the same CI environment used for publishing;
- compare raw size, compressed size, file count, and payload manifest;
- launch Chromium in app mode at the current `internal-csp` start URL;
- verify the MV2 CryptoPro extension still loads;
- publish a new immutable payload only after the smoke test;
- update `build/payload-lock.json` and record the new URL/size/SHA in
  `docs/worklog.md`;
- push and inspect GitHub Actions logs for `build-payload` and `build-windows`.
