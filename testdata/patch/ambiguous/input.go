package handler

// This file uses -32002 in a generic error handler.
// The patcher must not touch this: no MCP read method in scope.

const legacyErrorCode = -32002

func genericErrorHandler(code int) string {
	switch code {
	case -32002:
		return "legacy error"
	default:
		return "unknown"
	}
}
