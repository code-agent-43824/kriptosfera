package bootstrap

type EmbeddedCryptoProPluginInfo struct {
	Available bool
	Size      int64
	SHA256    string
}

func embeddedCryptoProPluginInfo() EmbeddedCryptoProPluginInfo {
	if len(embeddedCryptoProPlugin) == 0 {
		return EmbeddedCryptoProPluginInfo{}
	}
	return EmbeddedCryptoProPluginInfo{
		Available: true,
		Size:      int64(len(embeddedCryptoProPlugin)),
		SHA256:    checksumBytes(embeddedCryptoProPlugin),
	}
}
