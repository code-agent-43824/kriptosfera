package bootstrap

import "fmt"

const (
	ErrPayloadDownloadFailed  = "PAYLOAD_DOWNLOAD_FAILED"
	ErrPayloadHashMismatch    = "PAYLOAD_HASH_MISMATCH"
	ErrPayloadExtractFailed   = "PAYLOAD_EXTRACT_FAILED"
	ErrPayloadManifestInvalid = "PAYLOAD_MANIFEST_INVALID"
	ErrPayloadNotFound        = "PAYLOAD_NOT_FOUND"
)

type LauncherError struct {
	Code    string
	Message string
	Err     error
}

func (e *LauncherError) Error() string {
	if e == nil {
		return ""
	}
	if e.Err == nil {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
}

func (e *LauncherError) Unwrap() error { return e.Err }

func wrapLauncherError(code, message string, err error) error {
	return &LauncherError{Code: code, Message: message, Err: err}
}
