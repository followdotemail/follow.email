package debug

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"follow-email-backend/config"
)

var (
	fileLogger     *log.Logger
	logFile        *os.File
	logMutex       sync.Mutex
	currentLogDate string
	logsDir        string
)

// initFileLogger initializes or rotates the log file based on the current date
func initFileLogger() error {
	logMutex.Lock()
	defer logMutex.Unlock()

	today := time.Now().Format("2006-01-02")

	// If already initialized for today, skip
	if fileLogger != nil && currentLogDate == today {
		return nil
	}

	// Find the logs directory relative to the executable or working directory
	if logsDir == "" {
		// Try to find the logs directory
		possiblePaths := []string{
			"logs",
			"apps/hermes/logs",
			"../logs",
			"../../logs",
		}

		for _, path := range possiblePaths {
			absPath, _ := filepath.Abs(path)
			if _, err := os.Stat(filepath.Dir(absPath)); err == nil {
				logsDir = absPath
				break
			}
		}

		// Default to "logs" in current directory if not found
		if logsDir == "" {
			logsDir = "logs"
		}
	}

	// Create logs directory if it doesn't exist
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return fmt.Errorf("failed to create logs directory: %w", err)
	}

	// Close existing file if open
	if logFile != nil {
		logFile.Close()
	}

	// Create new log file with date-based name
	logFileName := filepath.Join(logsDir, fmt.Sprintf("debug_%s.log", today))
	var err error
	logFile, err = os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	// Create logger that writes to file only (console is handled separately)
	fileLogger = log.New(logFile, "", log.LstdFlags)
	currentLogDate = today

	return nil
}

// writeToFile writes a log message to the file
func writeToFile(level, text string) {
	if err := initFileLogger(); err != nil {
		// Silently fail - don't break the application if logging fails
		return
	}

	// Write without ANSI codes to file
	fileLogger.Printf("[%s] %s", level, text)
}

// isDebugEnabled checks if debug logging is enabled
func isDebugEnabled() bool {
	env := config.Load().Environment
	return env == "development" || env == "staging"
}

func DebugTextPrint(text string) {
	if !isDebugEnabled() {
		return
	}
	// Console output
	log.Printf("[DEBUG] %s", text)
	// File output
	writeToFile("DEBUG", text)
}

func DebugErrorTextPrint(text string) {
	if !isDebugEnabled() {
		return
	}
	// Console output with red color
	log.Printf("\033[31m[ERROR] %s\033[0m", text)
	// File output without color codes
	writeToFile("ERROR", text)
}

func DebugSuccessTextPrint(text string) {
	if !isDebugEnabled() {
		return
	}
	// Console output with green color
	log.Printf("\033[32m[SUCCESS] %s\033[0m", text)
	// File output without color codes
	writeToFile("SUCCESS", text)
}

func DebugWarningTextPrint(text string) {
	if !isDebugEnabled() {
		return
	}
	// Console output with yellow color
	log.Printf("\033[33m[WARNING] %s\033[0m", text)
	// File output without color codes
	writeToFile("WARNING", text)
}

// SetLogsDirectory allows setting a custom logs directory
func SetLogsDirectory(dir string) {
	logMutex.Lock()
	defer logMutex.Unlock()
	logsDir = dir
	// Reset to force re-initialization
	currentLogDate = ""
}

// CloseLogFile closes the current log file (call on application shutdown)
func CloseLogFile() {
	logMutex.Lock()
	defer logMutex.Unlock()
	if logFile != nil {
		logFile.Close()
		logFile = nil
		fileLogger = nil
	}
}

// Ensure io import is used
var _ = io.Discard
