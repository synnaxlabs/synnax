package logutil

import "go.uber.org/zap"

// DebugError returns a zap field that can be used to log an error whose presence
// is not exceptional i.e. it does not deserve a stack trace. zap.Error has no way
// to disable stack traces in debug logging, so we use this instead. DebugError should
// only be used in debug logging, and NOT for production errors that are exceptional.
func DebugError(err error) zap.Field {
	if err == nil {
		return zap.Skip()
	}
	return zap.String("error", err.Error())
}
