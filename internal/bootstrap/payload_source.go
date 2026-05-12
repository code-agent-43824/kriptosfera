package bootstrap

import (
	"context"
	"io"
)

type PayloadArchive struct {
	Reader io.ReadCloser
	Size   int64
	Close  func() error
}

type PayloadSource interface {
	Mode() string
	Version() string
	ExpectedSHA256() string
	Open(ctx context.Context) (PayloadArchive, error)
}
