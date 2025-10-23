package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"follow-email-backend/pkg/oauth" // Stub implementation for compilation
	amqp "github.com/rabbitmq/amqp091-go"
)

// QueueService handles RabbitMQ operations
type QueueService struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	queues  map[string]amqp.Queue
}

// Message types for different queue operations
type MessageType string

const (
	MessageTypeEmailSync     MessageType = "email_sync"
	MessageTypeEmailAnalysis MessageType = "email_analysis"
	MessageTypeFollowUp      MessageType = "follow_up"
	MessageTypeScheduled     MessageType = "scheduled_task"
)

// Queue names
const (
	QueueEmailSync     = "email_sync"
	QueueEmailAnalysis = "email_analysis"
	QueueFollowUp      = "follow_up"
	QueueScheduled     = "scheduled_tasks"
	QueueDeadLetter    = "dead_letter"
)

// EmailSyncMessage represents a message for email synchronization
type EmailSyncMessage struct {
	UserID       int                `json:"user_id"`
	Provider     oauth.Provider     `json:"provider"`
	TokenInfo    *oauth.TokenInfo   `json:"token_info"`
	LastSyncTime *time.Time         `json:"last_sync_time,omitempty"`
	HistoryID    string             `json:"history_id,omitempty"`
	DeltaToken   string             `json:"delta_token,omitempty"`
	RetryCount   int                `json:"retry_count"`
	ScheduledAt  time.Time          `json:"scheduled_at"`
}

// EmailAnalysisMessage represents a message for AI email analysis
type EmailAnalysisMessage struct {
	EmailID     int       `json:"email_id"`
	UserID      int       `json:"user_id"`
	MessageID   string    `json:"message_id"`
	Subject     string    `json:"subject"`
	FromEmail   string    `json:"from_email"`
	BodyS3Key   string    `json:"body_s3_key"`
	RetryCount  int       `json:"retry_count"`
	ScheduledAt time.Time `json:"scheduled_at"`
}

// FollowUpMessage represents a message for follow-up email generation
type FollowUpMessage struct {
	OriginalEmailID int       `json:"original_email_id"`
	UserID          int       `json:"user_id"`
	TemplateID      int       `json:"template_id,omitempty"`
	ScheduledFor    time.Time `json:"scheduled_for"`
	RetryCount      int       `json:"retry_count"`
	CreatedAt       time.Time `json:"created_at"`
}

// ScheduledTaskMessage represents a generic scheduled task
type ScheduledTaskMessage struct {
	TaskID      string                 `json:"task_id"`
	TaskType    string                 `json:"task_type"`
	UserID      int                    `json:"user_id"`
	Payload     map[string]interface{} `json:"payload"`
	ScheduledAt time.Time              `json:"scheduled_at"`
	RetryCount  int                    `json:"retry_count"`
}

// NewQueueService creates a new RabbitMQ service
func NewQueueService(rabbitmqURL string) (*QueueService, error) {
	conn, err := amqp.Dial(rabbitmqURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	qs := &QueueService{
		conn:    conn,
		channel: channel,
		queues:  make(map[string]amqp.Queue),
	}

	// Initialize queues
	if err := qs.initializeQueues(); err != nil {
		qs.Close()
		return nil, fmt.Errorf("failed to initialize queues: %w", err)
	}

	return qs, nil
}

// initializeQueues sets up all required queues with proper configuration
func (qs *QueueService) initializeQueues() error {
	// Dead letter exchange for failed messages
	err := qs.channel.ExchangeDeclare(
		"dlx",      // name
		"direct",   // type
		false,      // durable
		false,      // auto-deleted
		false,      // internal
		false,      // no-wait
		nil,        // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare dead letter exchange: %w", err)
	}

	// Dead letter queue
	dlq, err := qs.channel.QueueDeclare(
		QueueDeadLetter, // name
		true,            // durable
		false,           // delete when unused
		false,           // exclusive
		false,           // no-wait
		nil,             // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare dead letter queue: %w", err)
	}
	qs.queues[QueueDeadLetter] = dlq

	// Bind dead letter queue to exchange
	err = qs.channel.QueueBind(
		QueueDeadLetter, // queue name
		"failed",        // routing key
		"dlx",           // exchange
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to bind dead letter queue: %w", err)
	}

	// Queue arguments for dead letter routing
	queueArgs := amqp.Table{
		"x-dead-letter-exchange":    "dlx",
		"x-dead-letter-routing-key": "failed",
		"x-message-ttl":             300000, // 5 minutes TTL
	}

	// Email sync queue
	qs.queues[QueueEmailSync], err = qs.channel.QueueDeclare(
		QueueEmailSync, // name
		true,           // durable
		false,          // delete when unused
		false,          // exclusive
		false,          // no-wait
		queueArgs,      // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare email sync queue: %w", err)
	}

	// Email analysis queue
	qs.queues[QueueEmailAnalysis], err = qs.channel.QueueDeclare(
		QueueEmailAnalysis, // name
		true,               // durable
		false,              // delete when unused
		false,              // exclusive
		false,              // no-wait
		queueArgs,          // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare email analysis queue: %w", err)
	}

	// Follow-up queue
	qs.queues[QueueFollowUp], err = qs.channel.QueueDeclare(
		QueueFollowUp, // name
		true,          // durable
		false,         // delete when unused
		false,         // exclusive
		false,         // no-wait
		queueArgs,     // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare follow-up queue: %w", err)
	}

	// Scheduled tasks queue
	qs.queues[QueueScheduled], err = qs.channel.QueueDeclare(
		QueueScheduled, // name
		true,           // durable
		false,          // delete when unused
		false,          // exclusive
		false,          // no-wait
		queueArgs,      // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare scheduled tasks queue: %w", err)
	}

	// Set QoS to limit concurrent processing
	err = qs.channel.Qos(
		10,    // prefetch count
		0,     // prefetch size
		false, // global
	)
	if err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	log.Println("RabbitMQ queues initialized successfully")
	return nil
}

// PublishEmailSync publishes an email sync message to the queue
func (qs *QueueService) PublishEmailSync(ctx context.Context, msg *EmailSyncMessage) error {
	return qs.publishMessage(ctx, QueueEmailSync, MessageTypeEmailSync, msg)
}

// PublishEmailAnalysis publishes an email analysis message to the queue
func (qs *QueueService) PublishEmailAnalysis(ctx context.Context, msg *EmailAnalysisMessage) error {
	return qs.publishMessage(ctx, QueueEmailAnalysis, MessageTypeEmailAnalysis, msg)
}

// PublishFollowUp publishes a follow-up message to the queue
func (qs *QueueService) PublishFollowUp(ctx context.Context, msg *FollowUpMessage) error {
	return qs.publishMessage(ctx, QueueFollowUp, MessageTypeFollowUp, msg)
}

// PublishScheduledTask publishes a scheduled task message to the queue
func (qs *QueueService) PublishScheduledTask(ctx context.Context, msg *ScheduledTaskMessage) error {
	return qs.publishMessage(ctx, QueueScheduled, MessageTypeScheduled, msg)
}

// publishMessage is a generic method to publish messages to queues
func (qs *QueueService) publishMessage(ctx context.Context, queueName string, msgType MessageType, message interface{}) error {
	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	err = qs.channel.PublishWithContext(
		ctx,
		"",        // exchange
		queueName, // routing key
		false,     // mandatory
		false,     // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent, // Make message persistent
			Timestamp:    time.Now(),
			Type:         string(msgType),
			Headers: amqp.Table{
				"message_type": string(msgType),
				"published_at": time.Now().Unix(),
			},
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish message to %s: %w", queueName, err)
	}

	log.Printf("Published %s message to %s queue", msgType, queueName)
	return nil
}

// ConsumeEmailSync starts consuming email sync messages
func (qs *QueueService) ConsumeEmailSync(ctx context.Context, handler func(*EmailSyncMessage) error) error {
	return qs.consumeMessages(ctx, QueueEmailSync, func(body []byte) error {
		var msg EmailSyncMessage
		if err := json.Unmarshal(body, &msg); err != nil {
			return fmt.Errorf("failed to unmarshal email sync message: %w", err)
		}
		return handler(&msg)
	})
}

// ConsumeEmailAnalysis starts consuming email analysis messages
func (qs *QueueService) ConsumeEmailAnalysis(ctx context.Context, handler func(*EmailAnalysisMessage) error) error {
	return qs.consumeMessages(ctx, QueueEmailAnalysis, func(body []byte) error {
		var msg EmailAnalysisMessage
		if err := json.Unmarshal(body, &msg); err != nil {
			return fmt.Errorf("failed to unmarshal email analysis message: %w", err)
		}
		return handler(&msg)
	})
}

// ConsumeFollowUp starts consuming follow-up messages
func (qs *QueueService) ConsumeFollowUp(ctx context.Context, handler func(*FollowUpMessage) error) error {
	return qs.consumeMessages(ctx, QueueFollowUp, func(body []byte) error {
		var msg FollowUpMessage
		if err := json.Unmarshal(body, &msg); err != nil {
			return fmt.Errorf("failed to unmarshal follow-up message: %w", err)
		}
		return handler(&msg)
	})
}

// ConsumeScheduledTasks starts consuming scheduled task messages
func (qs *QueueService) ConsumeScheduledTasks(ctx context.Context, handler func(*ScheduledTaskMessage) error) error {
	return qs.consumeMessages(ctx, QueueScheduled, func(body []byte) error {
		var msg ScheduledTaskMessage
		if err := json.Unmarshal(body, &msg); err != nil {
			return fmt.Errorf("failed to unmarshal scheduled task message: %w", err)
		}
		return handler(&msg)
	})
}

// consumeMessages is a generic method to consume messages from queues
func (qs *QueueService) consumeMessages(ctx context.Context, queueName string, handler func([]byte) error) error {
	msgs, err := qs.channel.Consume(
		queueName, // queue
		"",        // consumer
		false,     // auto-ack
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer for %s: %w", queueName, err)
	}

	log.Printf("Started consuming messages from %s queue", queueName)

	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Printf("Stopping consumer for %s queue", queueName)
				return
			case msg, ok := <-msgs:
				if !ok {
					log.Printf("Message channel closed for %s queue", queueName)
					return
				}

				// Process message with retry logic
				if err := qs.processMessageWithRetry(msg, handler); err != nil {
					log.Printf("Failed to process message from %s: %v", queueName, err)
					msg.Nack(false, false) // Send to dead letter queue
				} else {
					msg.Ack(false)
				}
			}
		}
	}()

	return nil
}

// processMessageWithRetry processes a message with exponential backoff retry
func (qs *QueueService) processMessageWithRetry(msg amqp.Delivery, handler func([]byte) error) error {
	maxRetries := 3
	baseDelay := time.Second

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if err := handler(msg.Body); err != nil {
			if attempt == maxRetries {
				return fmt.Errorf("max retries exceeded: %w", err)
			}

			// Exponential backoff
			delay := baseDelay * time.Duration(1<<attempt)
			log.Printf("Message processing failed (attempt %d/%d), retrying in %v: %v", attempt+1, maxRetries+1, delay, err)
			time.Sleep(delay)
			continue
		}
		return nil
	}

	return nil
}

// GetQueueStats returns statistics for all queues
func (qs *QueueService) GetQueueStats() (map[string]amqp.Queue, error) {
	stats := make(map[string]amqp.Queue)

	for name := range qs.queues {
		queue, err := qs.channel.QueueInspect(name)
		if err != nil {
			return nil, fmt.Errorf("failed to inspect queue %s: %w", name, err)
		}
		stats[name] = queue
	}

	return stats, nil
}

// Close closes the RabbitMQ connection and channel
func (qs *QueueService) Close() error {
	if qs.channel != nil {
		qs.channel.Close()
	}
	if qs.conn != nil {
		return qs.conn.Close()
	}
	return nil
}

// IsConnected checks if the RabbitMQ connection is still active
func (qs *QueueService) IsConnected() bool {
	return qs.conn != nil && !qs.conn.IsClosed()
}