package bootstrap

import (
	"bytes"
	"context"

	"github.com/code-agent-43824/kriptosfera/internal/config"
)

type EmbeddedPayloadSource struct {
	version string
	payload []byte
	sha256  string
}

func NewEmbeddedPayloadSource(cfg config.RuntimeConfig, payload []byte) *EmbeddedPayloadSource {
	return &EmbeddedPayloadSource{
		version: cfg.Payload.Version,
		payload: payload,
		sha256:  checksumBytes(payload),
	}
}

func (s *EmbeddedPayloadSource) Mode() string { return config.PayloadModeEmbedded }

func (s *EmbeddedPayloadSource) Version() string { return s.version }

func (s *EmbeddedPayloadSource) ExpectedSHA256() string { return s.sha256 }

func (s *EmbeddedPayloadSource) Open(context.Context) (PayloadArchive, error) {
	reader := bytes.NewReader(s.payload)
	return PayloadArchive{
		ReaderAt: reader,
		Size:   int64(len(s.payload)),
		Close:  func() error { return nil },
	}, nil
}
