package bootstrap

import (
	"fmt"

	"github.com/code-agent-43824/kriptosfera/internal/logging"
)

const (
	chromePolicyExtensionManifestV2Availability = "ExtensionManifestV2Availability"
	chromePolicyLegacyMV2Enabled                = 2
)

func ApplyChromeCompatibilityPolicies(extensions []ExtensionSpec, logger *logging.Logger) error {
	if !requiresExtensionManifestV2Policy(extensions) {
		logger.Info("chrome compatibility policies skipped: no manifest v2 extensions")
		return nil
	}

	if err := setChromePolicyDWORD(chromePolicyExtensionManifestV2Availability, chromePolicyLegacyMV2Enabled); err != nil {
		return fmt.Errorf("apply chrome manifest v2 policy: %w", err)
	}
	logger.Info("chrome compatibility policy applied name=%s value=%d", chromePolicyExtensionManifestV2Availability, chromePolicyLegacyMV2Enabled)
	return nil
}

func requiresExtensionManifestV2Policy(extensions []ExtensionSpec) bool {
	for _, ext := range extensions {
		if ext.ManifestError == "" && ext.ManifestVersion == 2 {
			return true
		}
	}
	return false
}
