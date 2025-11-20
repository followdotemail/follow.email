package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/upstash/qstash-go"
)

// MessageType represents the type of message being sent
type MessageType string

const (
	EmailSyncType      MessageType = "email_sync"
	EmailAnalysisType  MessageType = "email_analysis"
	FollowUpType       MessageType = "follow_up"
	ScheduledTaskType  MessageType = "scheduled_task"
)

// Message structures for different queue operations
type EmailSyncMessage struct {
	UserID    string `json:"user_id"`
	MessageID string `json:"message_id,omitempty"`
}

type EmailAnalysisMessage struct {
	UserID    string `json:"user_id"`
	EmailID   string `json:"email_id"`
	BodyS3Key string `json:"body_s3_key"`
}

type FollowUpMessage struct {
	UserID  string `json:"user_id"`
	EmailID string `json:"email_id"`
	Content string `json:"content"`
}

type ScheduledTaskMessage struct {
	TaskType string                 `json:"task_type"`
	UserID   string                 `json:"user_id"`
	Payload  map[string]interface{} `json:"payload"`
}

// QStashService handles message queuing using QStash
type QStashService struct {
	client  *qstash.Client
	baseURL string
}

// NewQStashService creates a new QStash service instance
func NewQStashService(token, baseURL string) *QStashService {
	client := qstash.NewClient(token)
	return &QStashService{
		client:  client,
		baseURL: baseURL,
	}
}

// PublishEmailSync publishes an email sync message
func (q *QStashService) PublishEmailSync(ctx context.Context, msg EmailSyncMessage) error {
	return q.publishMessage(ctx, EmailSyncType, msg, nil)
}

// PublishEmailAnalysis publishes an email analysis message
func (q *QStashService) PublishEmailAnalysis(ctx context.Context, msg EmailAnalysisMessage) error {
	return q.publishMessage(ctx, EmailAnalysisType, msg, nil)
}

// PublishFollowUp publishes a follow-up message
func (q *QStashService) PublishFollowUp(ctx context.Context, msg FollowUpMessage) error {
	return q.publishMessage(ctx, FollowUpType, msg, nil)
}

// PublishScheduledTask publishes a scheduled task message
func (q *QStashService) PublishScheduledTask(ctx context.Context, msg ScheduledTaskMessage, delay *time.Duration) error {
	return q.publishMessage(ctx, ScheduledTaskType, msg, delay)
}

// publishMessage is a helper function to publish messages to QStash
func (q *QStashService) publishMessage(ctx context.Context, msgType MessageType, payload interface{}, delay *time.Duration) error {
	// Convert payload to map[string]any for QStash
	var bodyMap map[string]any
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal message payload: %w", err)
	}
	
	err = json.Unmarshal(jsonBytes, &bodyMap)
	if err != nil {
		return fmt.Errorf("failed to convert payload to map: %w", err)
	}

	// Construct the webhook URL based on message type
	// Convert message type to match webhook route format (underscore to hyphen)
	routeName := string(msgType)
	switch msgType {
		case EmailSyncType:
			routeName = "email-sync"
		case EmailAnalysisType:
			routeName = "email-analysis"
		case FollowUpType:
			routeName = "follow-up"
		case ScheduledTaskType:
			routeName = "scheduled-task"
		default:
			return fmt.Errorf("invalid message type: %s", msgType)
	}
	webhookURL := fmt.Sprintf("%s/%s", q.baseURL, routeName)

	// Create the publish options
	options := qstash.PublishJSONOptions{
		Url:     webhookURL,
		Body:    bodyMap,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}

	// Add delay if specified
	if delay != nil {
		delayStr := delay.String()
		options.Delay = delayStr
	}

	// Debug logging
	log.Printf("Publishing %s message to URL: %s", msgType, webhookURL)
	log.Printf("QStash client configured with baseURL: %s", q.baseURL)
	
	// Publish the message
	_, err = q.client.PublishJSON(options)
	if err != nil {
		log.Printf("QStash publish error: %v", err)
		return fmt.Errorf("failed to publish %s message: %w", msgType, err)
	}

	log.Printf("Successfully published %s message to QStash", msgType)
	return nil
}

// ScheduleRecurringTask schedules a recurring task using QStash's cron functionality
func (q *QStashService) ScheduleRecurringTask(ctx context.Context, cron string, msg ScheduledTaskMessage) error {
	// Convert message to map[string]any for QStash
	var bodyMap map[string]any
	jsonBytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal scheduled task payload: %w", err)
	}
	
	err = json.Unmarshal(jsonBytes, &bodyMap)
	if err != nil {
		return fmt.Errorf("failed to convert payload to map: %w", err)
	}

	// Construct the webhook URL
	webhookURL := fmt.Sprintf("%s/scheduled-task", q.baseURL)

	// Create the schedule options
	options := qstash.ScheduleJSONOptions{
		Destination: webhookURL,
		Body:        bodyMap,
		Cron:        cron,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}

	// Create the schedule
	_, err = q.client.Schedules().CreateJSON(options)
	if err != nil {
		return fmt.Errorf("failed to schedule recurring task: %w", err)
	}

	log.Printf("Successfully scheduled recurring task with cron: %s", cron)
	return nil
}

// Close gracefully shuts down the QStash service
func (q *QStashService) Close() error {
	// QStash client doesn't require explicit closing
	log.Println("QStash service closed")
	return nil
}