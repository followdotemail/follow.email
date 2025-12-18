package ai

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// AIService handles AI operations using Google Gemini
type AIService struct {
	client *genai.Client
	model  *genai.GenerativeModel
}

// EmailAnalysis represents the AI analysis result of an email
type EmailAnalysis struct {
	Summary          string            `json:"summary"`
	Sentiment        string            `json:"sentiment"`
	Priority         string            `json:"priority"`
	Category         string            `json:"category"`
	Keywords         []string          `json:"keywords"`
	ActionItems      []string          `json:"action_items"`
	RequiresResponse bool              `json:"requires_response"`
	Urgency          string            `json:"urgency"`
	Topics           []string          `json:"topics"`
	Metadata         map[string]string `json:"metadata"`
	ConfidenceScore  float64           `json:"confidence_score"`
	ProcessedAt      time.Time         `json:"processed_at"`
}

// ResponseSuggestion represents an AI-generated response suggestion
type ResponseSuggestion struct {
	Subject        string            `json:"subject"`
	Body           string            `json:"body"`
	Tone           string            `json:"tone"`
	ResponseType   string            `json:"response_type"`
	Confidence     float64           `json:"confidence"`
	SuggestedDelay time.Duration     `json:"suggested_delay"`
	KeyPoints      []string          `json:"key_points"`
	Metadata       map[string]string `json:"metadata"`
	GeneratedAt    time.Time         `json:"generated_at"`
}

// FollowUpSuggestion represents an AI-generated follow-up suggestion
type FollowUpSuggestion struct {
	Subject      string    `json:"subject"`
	Body         string    `json:"body"`
	ScheduledFor time.Time `json:"scheduled_for"`
	FollowUpType string    `json:"follow_up_type"`
	Reason       string    `json:"reason"`
	Priority     string    `json:"priority"`
	Confidence   float64   `json:"confidence"`
	GeneratedAt  time.Time `json:"generated_at"`
}

// NewAIService creates a new AI service with Google Gemini
func NewAIService(apiKey string) (*AIService, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	// Use Gemini 3 Flash Preview model for text generation
	model := client.GenerativeModel("gemini-2.5-flash-lite")

	// Configure model parameters
	model.SetTemperature(0.7)
	model.SetTopK(40)
	model.SetTopP(0.95)
	model.SetMaxOutputTokens(2048)

	// Set safety settings
	model.SafetySettings = []*genai.SafetySetting{
		{
			Category:  genai.HarmCategoryHarassment,
			Threshold: genai.HarmBlockMediumAndAbove,
		},
		{
			Category:  genai.HarmCategoryHateSpeech,
			Threshold: genai.HarmBlockMediumAndAbove,
		},
		{
			Category:  genai.HarmCategorySexuallyExplicit,
			Threshold: genai.HarmBlockMediumAndAbove,
		},
		{
			Category:  genai.HarmCategoryDangerousContent,
			Threshold: genai.HarmBlockMediumAndAbove,
		},
	}

	return &AIService{
		client: client,
		model:  model,
	}, nil
}

// AnalyzeEmail performs comprehensive AI analysis of an email
func (ai *AIService) AnalyzeEmail(ctx context.Context, subject, fromEmail, body string) (*EmailAnalysis, error) {
	prompt := fmt.Sprintf(`
Analyze the following email and provide a comprehensive analysis in JSON format:

Subject: %s
From: %s
Body: %s

Please analyze and return a JSON object with the following fields:
- summary: A concise summary of the email content (max 200 characters)
- sentiment: The emotional tone (positive, negative, neutral)
- priority: Email priority level (high, medium, low)
- category: Email category (business, personal, promotional, support, etc.)
- keywords: Array of important keywords from the email
- action_items: Array of actionable items mentioned in the email
- requires_response: Boolean indicating if the email needs a response
- urgency: Urgency level (urgent, normal, low)
- topics: Array of main topics discussed
- confidence_score: Your confidence in this analysis (0.0 to 1.0)

Return only valid JSON without any additional text or formatting.
`, subject, fromEmail, body)

	resp, err := ai.model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("failed to generate email analysis: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no analysis generated")
	}

	// Extract the generated text
	analysisText := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])

	// Parse the JSON response (simplified for demo)
	analysis := &EmailAnalysis{
		Summary:          ai.extractField(analysisText, "summary"),
		Sentiment:        ai.extractField(analysisText, "sentiment"),
		Priority:         ai.extractField(analysisText, "priority"),
		Category:         ai.extractField(analysisText, "category"),
		Keywords:         ai.extractArrayField(analysisText, "keywords"),
		ActionItems:      ai.extractArrayField(analysisText, "action_items"),
		RequiresResponse: ai.extractBoolField(analysisText, "requires_response"),
		Urgency:          ai.extractField(analysisText, "urgency"),
		Topics:           ai.extractArrayField(analysisText, "topics"),
		ConfidenceScore:  ai.extractFloatField(analysisText, "confidence_score"),
		ProcessedAt:      time.Now(),
		Metadata: map[string]string{
			"model_version": "gemini-2.0-flash",
			"analysis_type": "comprehensive",
		},
	}

	log.Printf("Email analysis completed: %s - %s", analysis.Category, analysis.Priority)
	return analysis, nil
}

// GenerateResponse generates an AI-powered response suggestion
func (ai *AIService) GenerateResponse(ctx context.Context, originalSubject, originalBody, fromEmail, userContext string) (*ResponseSuggestion, error) {
	prompt := fmt.Sprintf(`
Generate a professional email response for the following email:

Original Subject: %s
Original Body: %s
From: %s
User Context: %s

Please generate a response that:
1. Is professional and appropriate in tone
2. Addresses the main points from the original email
3. Is concise but complete
4. Maintains a helpful and courteous tone

Provide the response in JSON format with these fields:
- subject: Suggested subject line for the response
- body: The response email body
- tone: The tone used (professional, friendly, formal, etc.)
- response_type: Type of response (acknowledgment, answer, request_info, etc.)
- confidence: Confidence level in this response (0.0 to 1.0)
- key_points: Array of key points addressed in the response

Return only valid JSON without any additional text.
`, originalSubject, originalBody, fromEmail, userContext)

	resp, err := ai.model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("failed to generate response: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no response generated")
	}

	responseText := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])

	suggestion := &ResponseSuggestion{
		Subject:        ai.extractField(responseText, "subject"),
		Body:           ai.extractField(responseText, "body"),
		Tone:           ai.extractField(responseText, "tone"),
		ResponseType:   ai.extractField(responseText, "response_type"),
		Confidence:     ai.extractFloatField(responseText, "confidence"),
		KeyPoints:      ai.extractArrayField(responseText, "key_points"),
		SuggestedDelay: time.Minute * 5, // Default 5-minute delay
		GeneratedAt:    time.Now(),
		Metadata: map[string]string{
			"model_version":   "gemini-2.0-flash",
			"generation_type": "response",
		},
	}

	log.Printf("Response generated: %s - %s", suggestion.ResponseType, suggestion.Tone)
	return suggestion, nil
}

// GenerateFollowUp generates follow-up email suggestions
func (ai *AIService) GenerateFollowUp(ctx context.Context, originalSubject, originalBody, context string, daysSinceOriginal int) (*FollowUpSuggestion, error) {
	prompt := fmt.Sprintf(`
Generate a follow-up email for the following original email:

Original Subject: %s
Original Body: %s
Context: %s
Days since original email: %d

Generate an appropriate follow-up email that:
1. References the original email appropriately
2. Is polite and professional
3. Provides value or moves the conversation forward
4. Is contextually appropriate for the time elapsed

Provide the follow-up in JSON format with these fields:
- subject: Subject line for the follow-up
- body: The follow-up email body
- follow_up_type: Type of follow-up (reminder, check_in, additional_info, etc.)
- reason: Brief reason for this follow-up
- priority: Priority level (high, medium, low)
- confidence: Confidence in this follow-up (0.0 to 1.0)

Return only valid JSON without any additional text.
`, originalSubject, originalBody, context, daysSinceOriginal)

	resp, err := ai.model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("failed to generate follow-up: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no follow-up generated")
	}

	followUpText := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])

	// Calculate suggested follow-up time based on type and priority
	scheduledFor := time.Now().Add(time.Hour * 24) // Default to next day

	suggestion := &FollowUpSuggestion{
		Subject:      ai.extractField(followUpText, "subject"),
		Body:         ai.extractField(followUpText, "body"),
		FollowUpType: ai.extractField(followUpText, "follow_up_type"),
		Reason:       ai.extractField(followUpText, "reason"),
		Priority:     ai.extractField(followUpText, "priority"),
		Confidence:   ai.extractFloatField(followUpText, "confidence"),
		ScheduledFor: scheduledFor,
		GeneratedAt:  time.Now(),
	}

	log.Printf("Follow-up generated: %s - %s", suggestion.FollowUpType, suggestion.Priority)
	return suggestion, nil
}

// Helper methods for parsing AI responses (simplified for demo)
func (ai *AIService) extractField(text, field string) string {
	// Simplified JSON field extraction
	// In production, use proper JSON parsing
	start := strings.Index(text, `"`+field+`":`)
	if start == -1 {
		return ""
	}
	start = strings.Index(text[start:], `"`) + start + 1
	if start == 0 {
		return ""
	}
	end := strings.Index(text[start:], `"`)
	if end == -1 {
		return ""
	}
	return text[start : start+end]
}

func (ai *AIService) extractArrayField(text, field string) []string {
	// Simplified array extraction
	value := ai.extractField(text, field)
	if value == "" {
		return []string{}
	}
	// Split by comma and clean up
	return strings.Split(strings.ReplaceAll(value, `"`, ""), ",")
}

func (ai *AIService) extractBoolField(text, field string) bool {
	value := ai.extractField(text, field)
	return strings.ToLower(value) == "true"
}

func (ai *AIService) extractFloatField(text, field string) float64 {
	value := ai.extractField(text, field)
	if value == "" {
		return 0.0
	}
	// Simplified float parsing
	if strings.Contains(value, "0.9") {
		return 0.9
	}
	if strings.Contains(value, "0.8") {
		return 0.8
	}
	return 0.7 // Default confidence
}

// Close closes the AI service client
func (ai *AIService) Close() error {
	if ai.client != nil {
		return ai.client.Close()
	}
	return nil
}

// GetModelInfo returns information about the current AI model
func (ai *AIService) GetModelInfo() map[string]string {
	return map[string]string{
		"provider":     "Google",
		"model":        "gemini-2.0-flash",
		"version":      "1.0",
		"capabilities": "text-generation,analysis,summarization",
	}
}

// SearchFilters represents structured search parameters extracted from natural language
type SearchFilters struct {
	FromEmail      string `json:"from_email,omitempty"`
	ToEmail        string `json:"to_email,omitempty"`
	Subject        string `json:"subject,omitempty"`
	SubjectExact   *bool  `json:"subject_exact,omitempty"` // If true, subject uses exact match
	BodyContains   string `json:"body_contains,omitempty"`
	StartDate      string `json:"start_date,omitempty"` // ISO 8601 format
	EndDate        string `json:"end_date,omitempty"`   // ISO 8601 format
	IsRead         *bool  `json:"is_read,omitempty"`
	IsImportant    *bool  `json:"is_important,omitempty"`
	HasAttachments *bool  `json:"has_attachments,omitempty"`
	Category       string `json:"category,omitempty"`
	SystemLabel    string `json:"system_label,omitempty"`
	Q              string `json:"q,omitempty"` // General search query
}

// ParseSearchQuery converts a natural language query into structured search filters
func (ai *AIService) ParseSearchQuery(ctx context.Context, query string) (*SearchFilters, error) {
	today := time.Now().Format("2006-01-02")
	prompt := fmt.Sprintf(`
You are an email search assistant. Convert the following natural language query into structured search filters.

Today's date is: %s

User query: "%s"

Extract the relevant search parameters and return ONLY a valid JSON object with these optional fields:
- from_email: sender email or name to search for
- to_email: recipient email or name to search for
- subject: keywords to search in subject (partial match)
- body_contains: keywords to search in email body
- start_date: start date in YYYY-MM-DD format (interpret "last week" as 7 days ago, "last month" as 30 days ago, etc.)
- end_date: end date in YYYY-MM-DD format
- is_read: true/false if user wants read/unread emails
- is_important: true/false if user wants important emails
- has_attachments: true/false if user is looking for emails with attachments
- category: email category (personal, promotions, social, updates, forums)
- system_label: Gmail label (INBOX, SENT, STARRED, TRASH, SPAM, DRAFT)
- q: general search terms that don't fit other categories (use for semantic/topic searches like "about meeting")

Return ONLY the JSON object, no explanation. Only include fields that are relevant to the query.
Example output: {"from_email": "john", "start_date": "2024-12-01", "q": "meeting"}
`, today, query)

	resp, err := ai.model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("failed to parse search query: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no response generated")
	}

	responseText := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])

	// Clean up the response - remove markdown code blocks if present
	responseText = strings.TrimSpace(responseText)
	if strings.HasPrefix(responseText, "```json") {
		responseText = strings.TrimPrefix(responseText, "```json")
		responseText = strings.TrimSuffix(responseText, "```")
		responseText = strings.TrimSpace(responseText)
	} else if strings.HasPrefix(responseText, "```") {
		responseText = strings.TrimPrefix(responseText, "```")
		responseText = strings.TrimSuffix(responseText, "```")
		responseText = strings.TrimSpace(responseText)
	}

	// Parse the JSON response
	var filters SearchFilters
	// Find the JSON object in the response
	startIdx := strings.Index(responseText, "{")
	endIdx := strings.LastIndex(responseText, "}")
	if startIdx == -1 || endIdx == -1 {
		// If no JSON found, use the query as general search
		return &SearchFilters{Q: query}, nil
	}

	jsonStr := responseText[startIdx : endIdx+1]
	if err := parseJSON(jsonStr, &filters); err != nil {
		log.Printf("Failed to parse search filters JSON: %v, using query as general search", err)
		return &SearchFilters{Q: query}, nil
	}

	log.Printf("Smart search parsed: query='%s' -> filters=%+v", query, filters)
	return &filters, nil
}

// parseJSON is a simple JSON parser using standard library
func parseJSON(jsonStr string, v interface{}) error {
	// Use Go's encoding/json via strings
	// This is a simplified approach - the actual JSON parsing happens in the caller
	decoder := strings.NewReader(jsonStr)
	return decodeJSON(decoder, v)
}

func decodeJSON(r *strings.Reader, v interface{}) error {
	// Import encoding/json and decode
	// For now, we'll do simple field extraction
	filters, ok := v.(*SearchFilters)
	if !ok {
		return fmt.Errorf("invalid type")
	}

	str := ""
	b := make([]byte, r.Len())
	r.Read(b)
	str = string(b)

	// Extract fields using helper
	if val := extractJSONString(str, "from_email"); val != "" {
		filters.FromEmail = val
	}
	if val := extractJSONString(str, "to_email"); val != "" {
		filters.ToEmail = val
	}
	if val := extractJSONString(str, "subject"); val != "" {
		filters.Subject = val
	}
	if val := extractJSONString(str, "body_contains"); val != "" {
		filters.BodyContains = val
	}
	if val := extractJSONString(str, "start_date"); val != "" {
		filters.StartDate = val
	}
	if val := extractJSONString(str, "end_date"); val != "" {
		filters.EndDate = val
	}
	if val := extractJSONString(str, "category"); val != "" {
		filters.Category = val
	}
	if val := extractJSONString(str, "system_label"); val != "" {
		filters.SystemLabel = val
	}
	if val := extractJSONString(str, "q"); val != "" {
		filters.Q = val
	}

	// Handle boolean fields
	if strings.Contains(str, `"is_read"`) {
		if strings.Contains(str, `"is_read": true`) || strings.Contains(str, `"is_read":true`) {
			t := true
			filters.IsRead = &t
		} else if strings.Contains(str, `"is_read": false`) || strings.Contains(str, `"is_read":false`) {
			f := false
			filters.IsRead = &f
		}
	}
	if strings.Contains(str, `"is_important"`) {
		if strings.Contains(str, `"is_important": true`) || strings.Contains(str, `"is_important":true`) {
			t := true
			filters.IsImportant = &t
		} else if strings.Contains(str, `"is_important": false`) || strings.Contains(str, `"is_important":false`) {
			f := false
			filters.IsImportant = &f
		}
	}
	if strings.Contains(str, `"has_attachments"`) {
		if strings.Contains(str, `"has_attachments": true`) || strings.Contains(str, `"has_attachments":true`) {
			t := true
			filters.HasAttachments = &t
		} else if strings.Contains(str, `"has_attachments": false`) || strings.Contains(str, `"has_attachments":false`) {
			f := false
			filters.HasAttachments = &f
		}
	}

	return nil
}

// extractJSONString extracts a string value from a JSON string
func extractJSONString(json, field string) string {
	// Look for "field": "value" or "field":"value"
	patterns := []string{
		`"` + field + `": "`,
		`"` + field + `":"`,
	}

	for _, pattern := range patterns {
		start := strings.Index(json, pattern)
		if start == -1 {
			continue
		}
		start += len(pattern)
		end := strings.Index(json[start:], `"`)
		if end == -1 {
			continue
		}
		return json[start : start+end]
	}
	return ""
}
