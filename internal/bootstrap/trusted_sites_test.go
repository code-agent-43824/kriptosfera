package bootstrap

import (
	"reflect"
	"testing"

	"github.com/code-agent-43824/kriptosfera/internal/config"
)

func TestPrepareCryptoProTrustedSitesWritesAndReuses(t *testing.T) {
	appDir := t.TempDir()
	logger := testLogger(t)

	var writes int
	var lastKey, lastValue string
	var lastSites []string
	original := writeCryptoProTrustedSites
	writeCryptoProTrustedSites = func(key, value string, sites []string) error {
		writes++
		lastKey, lastValue, lastSites = key, value, sites
		return nil
	}
	t.Cleanup(func() { writeCryptoProTrustedSites = original })

	sites := []string{"https://cryptopro.ru", "https://mescheryakov.pro"}
	res, err := PrepareCryptoProTrustedSites(appDir, sites, logger)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Written || res.Reused || res.Skipped {
		t.Fatalf("expected Written on first run, got %+v", res)
	}
	if writes != 1 {
		t.Fatalf("expected 1 write, got %d", writes)
	}
	if lastKey != cryptoProTrustedSitesKeyPath || lastValue != cryptoProTrustedSitesValueName {
		t.Fatalf("unexpected key/value: %s / %s", lastKey, lastValue)
	}
	if !reflect.DeepEqual(lastSites, sites) {
		t.Fatalf("unexpected sites written: %v", lastSites)
	}
	if res.RegistryKey != `HKCU\Software\Crypto Pro\CAdESplugin` {
		t.Fatalf("unexpected registry key: %s", res.RegistryKey)
	}

	// Second run with the same list must reuse (no re-write).
	res2, err := PrepareCryptoProTrustedSites(appDir, sites, logger)
	if err != nil {
		t.Fatal(err)
	}
	if !res2.Reused || res2.Written {
		t.Fatalf("expected Reused on second run, got %+v", res2)
	}
	if writes != 1 {
		t.Fatalf("expected no additional write, got %d", writes)
	}

	// Changing the list must trigger a re-write.
	if _, err := PrepareCryptoProTrustedSites(appDir, append(sites, "https://*.cryptopro.ru"), logger); err != nil {
		t.Fatal(err)
	}
	if writes != 2 {
		t.Fatalf("expected re-write after list change, got %d", writes)
	}
}

func TestPrepareCryptoProTrustedSitesSkipsWhenEmpty(t *testing.T) {
	appDir := t.TempDir()
	logger := testLogger(t)

	called := false
	original := writeCryptoProTrustedSites
	writeCryptoProTrustedSites = func(string, string, []string) error { called = true; return nil }
	t.Cleanup(func() { writeCryptoProTrustedSites = original })

	res, err := PrepareCryptoProTrustedSites(appDir, nil, logger)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Skipped || res.Written || res.Reused {
		t.Fatalf("expected Skipped for empty list, got %+v", res)
	}
	if called {
		t.Fatal("registry writer must not be called for an empty list")
	}
}

func TestNormalizeTrustedSitesTrimsAndDedupes(t *testing.T) {
	got := normalizeTrustedSites([]string{" https://a ", "https://a", "", "https://b", "  "})
	want := []string{"https://a", "https://b"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeTrustedSites = %v, want %v", got, want)
	}
}

func TestValidateAppConfigTrustedSites(t *testing.T) {
	base := config.AppConfig{
		StartURL:       "https://mescheryakov.pro/x",
		AllowedOrigins: []string{"https://mescheryakov.pro"},
		ProfileName:    "demo",
	}

	valid := base
	valid.TrustedSites = []string{"https://cryptopro.ru", "https://*.cryptopro.ru", "http://www.*.com"}
	if err := validateAppConfig(valid); err != nil {
		t.Fatalf("expected valid trusted sites to pass, got %v", err)
	}

	for _, bad := range []string{"cryptopro.ru", "https://", "://host", ""} {
		cfg := base
		cfg.TrustedSites = []string{bad}
		if err := validateAppConfig(cfg); err == nil {
			t.Fatalf("expected error for invalid trusted site %q", bad)
		}
	}
}
