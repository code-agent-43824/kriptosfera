// Package config defines the two configuration layers used by the launcher.
//
// [RuntimeConfig] is baked into the launcher binary at build time (via
// runtime-config.json / app-version.txt) and selects the payload mode
// (embedded or remote) together with its version and, for remote mode, the
// download URL, expected SHA-256, and size.
//
// [AppConfig] ships inside the payload (config/app-config.json) and describes
// the product-facing behavior: start URL, allowed origins, profile name,
// window mode, diagnostics, and extra Chromium arguments.
package config
