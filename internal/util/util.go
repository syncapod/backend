package util

import "log/slog"

// Err is a wrapper to quickly log an error
func Err(err error) slog.Attr {
	return slog.Any("error", err)
}
