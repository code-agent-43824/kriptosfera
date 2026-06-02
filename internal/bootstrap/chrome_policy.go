package bootstrap

import (
	"github.com/code-agent-43824/kriptosfera/internal/logging"
)

const (
	chromePolicyExtensionManifestV2Availability = "ExtensionManifestV2Availability"
	chromePolicyLegacyMV2Enabled                = 2
	chromeLegacyMV2EnableFeature                = "AllowLegacyMV2Extensions"
	chromeLegacyMV2DisabledFeature              = "ExtensionManifestV2Disabled"
	chromeLegacyMV2UnsupportedFeature           = "ExtensionManifestV2Unsupported"
)

var writeChromePolicyDWORD = setChromePolicyDWORD

func ApplyChromeCompatibilityPolicies(extensions []ExtensionSpec, logger *logging.Logger) []string {
	if !requiresExtensionManifestV2Policy(extensions) {
		logger.Info("chrome compatibility policies skipped: no manifest v2 extensions")
		return nil
	}

	if err := writeChromePolicyDWORD(chromePolicyExtensionManifestV2Availability, chromePolicyLegacyMV2Enabled); err != nil {
		fallbackArgs := chromeLegacyMV2FallbackArgs()
		logger.Info("chrome compatibility policy write failed name=%s fallback_args=true error=%s", chromePolicyExtensionManifestV2Availability, err)
		return fallbackArgs
	}
	logger.Info("chrome compatibility policy applied name=%s value=%d", chromePolicyExtensionManifestV2Availability, chromePolicyLegacyMV2Enabled)
	return nil
}

func chromeLegacyMV2FallbackArgs() []string {
	return []string{
		"--enable-features=" + chromeLegacyMV2EnableFeature,
		"--disable-features=" + chromeLegacyMV2UnsupportedFeature + "," + chromeLegacyMV2DisabledFeature,
	}
}

func requiresExtensionManifestV2Policy(extensions []ExtensionSpec) bool {
	for _, ext := range extensions {
		if ext.ManifestError == "" && ext.ManifestVersion == 2 {
			return true
		}
	}
	return false
}
