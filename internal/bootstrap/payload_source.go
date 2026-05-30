package bootstrap

import (
	"context"
	"io"
)

// PayloadArchive is an opened payload zip ready for extraction. Close releases
// any backing resources (for remote sources it also removes the temp file).
type PayloadArchive struct {
	ReaderAt io.ReaderAt
	Size     int64
	Close    func() error
}

// PayloadSource abstracts where the launcher obtains its payload. Implementations
// expose the payload's identity (mode, version, expected SHA-256) and open the
// archive on demand. The two implementations are EmbeddedPayloadSource and
// RemotePayloadSource.
type PayloadSource interface {
	Mode() string
	Version() string
	ExpectedSHA256() string
	Open(ctx context.Context) (PayloadArchive, error)
}
