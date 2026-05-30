// Package logging provides the launcher's minimal append-only file logger.
//
// Each [Logger] writes RFC 3339 UTC-timestamped lines to a single file and is
// safe to use for the launcher's sequential bootstrap steps. It intentionally
// has no log levels or rotation: the launcher is a short-lived bootstrapper and
// the log is meant for first-run diagnostics, not long-term operation.
package logging
