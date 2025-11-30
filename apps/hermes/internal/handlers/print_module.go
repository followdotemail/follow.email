package handlers

import "follow-email-backend/pkg/debug"

// These functions delegate to the debug package for consistency
// This allows handlers package to use local function names while
// other packages import from pkg/debug

func DebugTextPrint(text string) {
	debug.DebugTextPrint(text)
}

func DebugErrorTextPrint(text string) {
	debug.DebugErrorTextPrint(text)
}

func DebugSuccessTextPrint(text string) {
	debug.DebugSuccessTextPrint(text)
}

func DebugWarningTextPrint(text string) {
	debug.DebugWarningTextPrint(text)
}
