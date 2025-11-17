package email

import (
	"encoding/base64"
	"strings"
	"testing"
)

// Test data for different email content scenarios
var (
	// Plain text email content
	plainTextContent = "Hello World!\nThis is a plain text email.\nBest regards,\nJohn"
	
	// HTML email content
	htmlContent = "<html><body><h1>Hello World!</h1><p>This is an <strong>HTML</strong> email.</p><p>Best regards,<br>John</p></body></html>"
	
	// Mixed content with HTML entities
	mixedContent = "<p>Hello &amp; welcome!</p><p>Price: &lt;$100&gt;</p>"
	
	// Empty content
	emptyContent = ""
)

// Helper function to create base64url encoded test data
func createBase64URLTestData(content string) string {
	encoded := base64.StdEncoding.EncodeToString([]byte(content))
	// Convert to base64url format
	encoded = strings.ReplaceAll(encoded, "+", "-")
	encoded = strings.ReplaceAll(encoded, "/", "_")
	encoded = strings.TrimRight(encoded, "=")
	return encoded
}

func TestFromBinary(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		hasError bool
	}{
		{
			name:     "Plain text content",
			input:    createBase64URLTestData(plainTextContent),
			expected: plainTextContent,
			hasError: false,
		},
		{
			name:     "HTML content",
			input:    createBase64URLTestData(htmlContent),
			expected: htmlContent,
			hasError: false,
		},
		{
			name:     "Empty content",
			input:    "",
			expected: "",
			hasError: false,
		},
		{
			name:     "Invalid base64",
			input:    "invalid-base64-data!!!",
			expected: "",
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FromBinary(tt.input)
			
			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("Expected %q, got %q", tt.expected, result)
				}
			}
		})
	}
}

func TestDecodeHTMLEntities(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "HTML entities",
			input:    "Hello &amp; welcome! Price: &lt;$100&gt;",
			expected: "Hello & welcome! Price: <$100>",
		},
		{
			name:     "No entities",
			input:    "Plain text without entities",
			expected: "Plain text without entities",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Multiple entities",
			input:    "&quot;Hello&quot; &amp; &apos;World&apos;",
			expected: "\"Hello\" & 'World'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DecodeHTMLEntities(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestStripHTMLTags(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "HTML with tags",
			input:    "<p>Hello <strong>World</strong>!</p>",
			expected: "Hello World!",
		},
		{
			name:     "Plain text",
			input:    "Hello World!",
			expected: "Hello World!",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Complex HTML",
			input:    "<html><body><h1>Title</h1><p>Content</p></body></html>",
			expected: "TitleContent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StripHTMLTags(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestIsPlainText(t *testing.T) {
	tests := []struct {
		name         string
		original     string
		stripped     string
		expectedBool bool
	}{
		{
			name:         "Plain text content",
			original:     "Hello World!",
			stripped:     "Hello World!",
			expectedBool: true,
		},
		{
			name:         "HTML content",
			original:     "<p>Hello World!</p>",
			stripped:     "Hello World!",
			expectedBool: false,
		},
		{
			name:         "Empty content",
			original:     "",
			stripped:     "",
			expectedBool: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsPlainText(tt.original, tt.stripped)
			if result != tt.expectedBool {
				t.Errorf("Expected %v, got %v", tt.expectedBool, result)
			}
		})
	}
}

func TestConvertNewlinesToBR(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Text with newlines",
			input:    "Line 1\nLine 2\nLine 3",
			expected: "Line 1<br>Line 2<br>Line 3",
		},
		{
			name:     "Text without newlines",
			input:    "Single line text",
			expected: "Single line text",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertNewlinesToBR(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestProcessEmailBody(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		hasError bool
	}{
		{
			name:     "Plain text email",
			input:    createBase64URLTestData(plainTextContent),
			expected: "Hello World!<br>This is a plain text email.<br>Best regards,<br>John",
			hasError: false,
		},
		{
			name:     "HTML email",
			input:    createBase64URLTestData(htmlContent),
			expected: htmlContent,
			hasError: false,
		},
		{
			name:     "Mixed content with entities",
			input:    createBase64URLTestData(mixedContent),
			expected: "<p>Hello & welcome!</p><p>Price: <$100></p>",
			hasError: false,
		},
		{
			name:     "Empty content",
			input:    "",
			expected: "",
			hasError: false,
		},
		{
			name:     "Invalid base64",
			input:    "invalid-base64-data!!!",
			expected: "",
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ProcessEmailBody(tt.input)
			
			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("Expected %q, got %q", tt.expected, result)
				}
			}
		})
	}
}

func TestProcessEmailBodySafe(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Valid plain text",
			input:    createBase64URLTestData(plainTextContent),
			expected: "Hello World!<br>This is a plain text email.<br>Best regards,<br>John",
		},
		{
			name:     "Invalid base64 - should return empty",
			input:    "invalid-base64-data!!!",
			expected: "",
		},
		{
			name:     "Empty input",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ProcessEmailBodySafe(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestProcessEmailBodyWithFallback(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Valid content",
			input:    createBase64URLTestData(plainTextContent),
			expected: "Hello World!<br>This is a plain text email.<br>Best regards,<br>John",
		},
		{
			name:     "Invalid base64 - should return original",
			input:    "invalid-base64-data!!!",
			expected: "invalid-base64-data!!!",
		},
		{
			name:     "Empty input",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ProcessEmailBodyWithFallback(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestValidateEmailBodyData(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		hasError bool
	}{
		{
			name:     "Valid base64url data",
			input:    createBase64URLTestData(plainTextContent),
			hasError: false,
		},
		{
			name:     "Empty data",
			input:    "",
			hasError: false,
		},
		{
			name:     "Too short data",
			input:    "abc",
			hasError: true,
		},
		{
			name:     "Invalid characters",
			input:    "invalid@#$%characters",
			hasError: true,
		},
		{
			name:     "Valid characters with padding",
			input:    "SGVsbG8gV29ybGQ=",
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmailBodyData(tt.input)
			
			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// Benchmark tests for performance
func BenchmarkProcessEmailBody(b *testing.B) {
	testData := createBase64URLTestData(htmlContent)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ProcessEmailBody(testData)
	}
}

func BenchmarkOptimizedProcessEmailBody(b *testing.B) {
	testData := createBase64URLTestData(htmlContent)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		OptimizedProcessEmailBody(testData)
	}
}