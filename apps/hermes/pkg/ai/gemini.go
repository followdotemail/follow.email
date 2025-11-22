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
	Subject         string            `json:"subject"`
	Body            string            `json:"body"`
	Tone            string            `json:"tone"`
	ResponseType    string            `json:"response_type"`
	Confidence      float64           `json:"confidence"`
	SuggestedDelay  time.Duration     `json:"suggested_delay"`
	KeyPoints       []string          `json:"key_points"`
	Metadata        map[string]string `json:"metadata"`
	GeneratedAt     time.Time         `json:"generated_at"`
}

// FollowUpSuggestion represents an AI-generated follow-up suggestion
type FollowUpSuggestion struct {
	Subject         string        `json:"subject"`
	Body            string        `json:"body"`
	ScheduledFor    time.Time     `json:"scheduled_for"`
	FollowUpType    string        `json:"follow_up_type"`
	Reason          string        `json:"reason"`
	Priority        string        `json:"priority"`
	Confidence      float64       `json:"confidence"`
	GeneratedAt     time.Time     `json:"generated_at"`
}

// NewAIService creates a new AI service with Google Gemini
func NewAIService(apiKey string) (*AIService, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	// Use Gemini Pro model for text generation
	model := client.GenerativeModel("gemini-pro")
	
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
			"model_version": "gemini-pro",
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
			"model_version": "gemini-pro",
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
		"model":        "gemini-pro",
		"version":      "1.0",
		"capabilities": "text-generation,analysis,summarization",
	}
}