package handlers

import "log"
import "follow-email-backend/config"


func DebugTextPrint(text string)  {
    if config.Load().Environment == "development" || config.Load().Environment == "staging" {
		log.Printf("[DEBUG] %s", text)
	}
}

func DebugErrorTextPrint(text string)  {
	if config.Load().Environment == "development" || config.Load().Environment == "staging" {
		// Print error text in red color using ANSI escape codes
		log.Printf("\033[31m[ERROR] %s\033[0m", text)
	}
}

func DebugSuccessTextPrint(text string)  {
	if config.Load().Environment == "development" || config.Load().Environment == "staging" {
		// Print success text in green color using ANSI escape codes
		log.Printf("\033[32m[SUCCESS] %s\033[0m", text)
	}
}

func DebugWarningTextPrint(text string)  {
	if config.Load().Environment == "development" || config.Load().Environment == "staging" {
		// Print warning text in yellow color using ANSI escape codes
		log.Printf("\033[33m[WARNING] %s\033[0m", text)
	}
}