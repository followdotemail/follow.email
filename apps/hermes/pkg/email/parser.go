package email

import (
	"encoding/base64"
	"fmt"
	"html"
	"regexp"
	"strings"
)

// EmailParsingError represents errors that occur during email parsing
type EmailParsingError struct {
	Operation string
	Err       error
}

func (e *EmailParsingError) Error() string {
	return fmt.Sprintf("email parsing error in %s: %v", e.Operation, e.Err)
}

// FromBinary converts Gmail's base64url encoding to readable text
// Equivalent to the TypeScript fromBinary function
func FromBinary(str string) (string, error) {
	if str == "" {
		return "", nil
	}
	
	// Convert base64url format (uses - and _) to standard base64 (+ and /)
	base64Str := strings.ReplaceAll(str, "-", "+")
	base64Str = strings.ReplaceAll(base64Str, "_", "/")
	
	// Add padding if necessary
	switch len(base64Str) % 4 {
	case 2:
		base64Str += "=="
	case 3:
		base64Str += "="
	}
	
	// Try multiple decoding strategies
	decodingStrategies := []func(string) ([]byte, error){
		base64.StdEncoding.DecodeString,
		base64.URLEncoding.DecodeString,
		base64.RawStdEncoding.DecodeString,
		base64.RawURLEncoding.DecodeString,
	}
	
	var decoded []byte
	var err error
	
	for i, strategy := range decodingStrategies {
		decoded, err = strategy(base64Str)
		if err == nil {
			break
		}
		// If it's the last strategy and still failing, return the error
		if i == len(decodingStrategies)-1 {
			return "", &EmailParsingError{
				Operation: "base64 decoding",
				Err:       err,
			}
		}
	}
	
	// Convert bytes to UTF-8 string
	return string(decoded), nil
}

// DecodeHTMLEntities decodes HTML entities to their actual characters
// Equivalent to he.decode() in TypeScript
func DecodeHTMLEntities(text string) string {
	if text == "" {
		return ""
	}
	return html.UnescapeString(text)
}

// StripHTMLTags removes all HTML tags from the text
// Equivalent to .replace(/<[^>]*>/g, '') in TypeScript
func StripHTMLTags(text string) string {
	if text == "" {
		return ""
	}
	re := regexp.MustCompile(`<[^>]*>`)
	return re.ReplaceAllString(text, "")
}

// IsPlainText determines if the content is plain text (no HTML tags)
func IsPlainText(originalDecoded, strippedHTML string) bool {
	return strings.TrimSpace(strippedHTML) == strings.TrimSpace(originalDecoded)
}

// ConvertNewlinesToBR converts newline characters to HTML <br> tags
func ConvertNewlinesToBR(text string) string {
	return strings.ReplaceAll(text, "\n", "<br>")
}

// ProcessEmailBody processes Gmail API email body data with content type detection
// Implements the core logic from the TypeScript version:
// const decodedBody = bodyData
//   ? he.decode(fromBinary(bodyData)).replace(/<[^>]*>/g, '').trim() === fromBinary(bodyData).trim()
//     ? he.decode(fromBinary(bodyData).replace(/\n/g, '<br>'))
//     : he.decode(fromBinary(bodyData))
//   : '';
func ProcessEmailBody(bodyData string) (string, error) {
	if bodyData == "" {
		return "", nil
	}
	
	// Step 1: Decode the base64url content
	decoded, err := FromBinary(bodyData)
	if err != nil {
		return "", err
	}
	
	// Step 2: Decode HTML entities and strip HTML tags for comparison
	decodedWithEntities := DecodeHTMLEntities(decoded)
	strippedHTML := StripHTMLTags(decodedWithEntities)
	
	// Step 3: Content type detection - compare stripped version with original
	if IsPlainText(decoded, strippedHTML) {
		// Step 4a: Plain text path - convert newlines to <br> tags
		textWithBR := ConvertNewlinesToBR(decoded)
		return DecodeHTMLEntities(textWithBR), nil
	} else {
		// Step 4b: HTML path - keep original structure
		return decodedWithEntities, nil
	}
}

// ProcessEmailBodySafe is a safe wrapper around ProcessEmailBody with error handling
func ProcessEmailBodySafe(bodyData string) string {
	result, err := ProcessEmailBody(bodyData)
	if err != nil {
		// Return empty string as fallback for any parsing errors
		return ""
	}
	return result
}

// ProcessEmailBodyWithFallback processes email body with multiple fallback strategies
func ProcessEmailBodyWithFallback(bodyData string) string {
	if bodyData == "" {
		return ""
	}
	
	// Try the main processing function first
	result, err := ProcessEmailBody(bodyData)
	if err == nil {
		return result
	}
	
	// Fallback 1: Try to decode as raw base64 without processing
	if decoded, err := FromBinary(bodyData); err == nil {
		return DecodeHTMLEntities(decoded)
	}
	
	// Fallback 2: Return the original data if all else fails
	return bodyData
}

// ValidateEmailBodyData performs basic validation on email body data
func ValidateEmailBodyData(bodyData string) error {
	if bodyData == "" {
		return nil // Empty is valid
	}
	
	// Check if it looks like base64 data
	if len(bodyData) < 4 {
		return &EmailParsingError{
			Operation: "validation",
			Err:       fmt.Errorf("body data too short to be valid base64"),
		}
	}
	
	// Check for valid base64url characters
	validChars := regexp.MustCompile(`^[A-Za-z0-9\-_=]*$`)
	if !validChars.MatchString(bodyData) {
		return &EmailParsingError{
			Operation: "validation",
			Err:       fmt.Errorf("body data contains invalid base64url characters"),
		}
	}
	
	return nil
}

// OptimizedProcessEmailBody is an optimized version that minimizes repeated decoding
func OptimizedProcessEmailBody(bodyData string) (string, error) {
	if bodyData == "" {
		return "", nil
	}
	
	// Decode once and reuse the result
	decoded, err := FromBinary(bodyData)
	if err != nil {
		return "", err
	}
	
	decodedWithEntities := DecodeHTMLEntities(decoded)
	strippedHTML := StripHTMLTags(decodedWithEntities)
	
	if IsPlainText(decoded, strippedHTML) {
		// Plain text: convert newlines to <br> tags
		textWithBR := ConvertNewlinesToBR(decoded)
		return DecodeHTMLEntities(textWithBR), nil
	} else {
		// HTML content: keep original structure
		return decodedWithEntities, nil
	}
}