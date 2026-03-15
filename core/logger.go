package core

import (
	"log"
	"os"
)

// Log prints a debug message when SKIPPER_DEBUG is set.
func Log(msg string) {
	if isDebugEnabled() {
		log.Println("[skipper]", msg)
	}
}

// Logf prints a formatted debug message when SKIPPER_DEBUG is set.
func Logf(format string, args ...any) {
	if isDebugEnabled() {
		log.Printf("[skipper] "+format, args...)
	}
}

// Warn prints a warning message when SKIPPER_DEBUG is set.
func Warn(msg string) {
	if isDebugEnabled() {
		log.Println("[skipper] WARN:", msg)
	}
}

func isDebugEnabled() bool {
	return os.Getenv("SKIPPER_DEBUG") != ""
}
