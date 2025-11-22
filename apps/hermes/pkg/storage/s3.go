package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// S3Service handles S3 operations for email storage
type S3Service struct {
	client     *s3.Client
	bucketName string
	region     string
}

// StorageConfig holds S3 configuration
type StorageConfig struct {
	BucketName string
	Region     string
	AccessKey  string
	SecretKey  string
	Endpoint   string // For S3-compatible services
}

// EmailStorageMetadata contains metadata for stored emails
type EmailStorageMetadata struct {
	EmailID     int               `json:"email_id"`
	UserID      int               `json:"user_id"`
	MessageID   string            `json:"message_id"`
	S3Key       string            `json:"s3_key"`
	ContentType string            `json:"content_type"`
	Size        int64             `json:"size"`
	StoredAt    time.Time         `json:"stored_at"`
	Metadata    map[string]string `json:"metadata"`
}

// AttachmentStorageMetadata contains metadata for stored attachments
type AttachmentStorageMetadata struct {
	AttachmentID int               `json:"attachment_id"`
	EmailID      int               `json:"email_id"`
	UserID       int               `json:"user_id"`
	Filename     string            `json:"filename"`
	S3Key        string            `json:"s3_key"`
	ContentType  string            `json:"content_type"`
	Size         int64             `json:"size"`
	StoredAt     time.Time         `json:"stored_at"`
	Metadata     map[string]string `json:"metadata"`
}

// NewS3Service creates a new S3 service
func NewS3Service(ctx context.Context, cfg *StorageConfig) (*S3Service, error) {
	// Load AWS config
	awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(cfg.Region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client
	client := s3.NewFromConfig(awsCfg)

	// If custom endpoint is provided (for S3-compatible services)
	if cfg.Endpoint != "" {
		client = s3.NewFromConfig(awsCfg, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
			o.UsePathStyle = true
		})
	}

	s3Service := &S3Service{
		client:     client,
		bucketName: cfg.BucketName,
		region:     cfg.Region,
	}

	// Ensure bucket exists
	if err := s3Service.ensureBucketExists(ctx); err != nil {
		return nil, fmt.Errorf("failed to ensure bucket exists: %w", err)
	}

	return s3Service, nil
}

// ensureBucketExists checks if bucket exists and creates it if not
func (s *S3Service) ensureBucketExists(ctx context.Context) error {
	// Check if bucket exists
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(s.bucketName),
	})
	if err == nil {
		log.Printf("S3 bucket %s already exists", s.bucketName)
		return nil
	}

	// Create bucket if it doesn't exist
	createInput := &s3.CreateBucketInput{
		Bucket: aws.String(s.bucketName),
	}

	// Set location constraint for regions other than us-east-1
	if s.region != "us-east-1" {
		createInput.CreateBucketConfiguration = &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraint(s.region),
		}
	}

	_, err = s.client.CreateBucket(ctx, createInput)
	if err != nil {
		return fmt.Errorf("failed to create bucket: %w", err)
	}

	log.Printf("Created S3 bucket: %s", s.bucketName)
	return nil
}

// StoreEmailBody stores email body content in S3
func (s *S3Service) StoreEmailBody(ctx context.Context, userID, messageID, body string) (*EmailStorageMetadata, error) {
	// Generate S3 key for email body
	s3Key := s.generateEmailBodyKey(userID, messageID)

	// Convert body to bytes
	bodyBytes := []byte(body)

	// Upload to S3
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucketName),
		Key:         aws.String(s3Key),
		Body:        bytes.NewReader(bodyBytes),
		ContentType: aws.String("text/html"),
		Metadata: map[string]string{
			"user-id":    userID,
			"message-id": messageID,
			"type":       "email-body",
			"stored-at":  time.Now().Format(time.RFC3339),
		},
		ServerSideEncryption: types.ServerSideEncryptionAes256,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to store email body: %w", err)
	}

	metadata := &EmailStorageMetadata{
		UserID:      0, // String userID converted for legacy compatibility
		MessageID:   messageID,
		S3Key:       s3Key,
		ContentType: "text/html",
		Size:        int64(len(bodyBytes)),
		StoredAt:    time.Now(),
		Metadata: map[string]string{
			"type": "email-body",
		},
	}

	log.Printf("Stored email body for user %s, message %s at %s", userID, messageID, s3Key)
	return metadata, nil
}

// StoreAttachment stores email attachment in S3
func (s *S3Service) StoreAttachment(ctx context.Context, userID, messageID, filename string, content io.Reader, contentType string) (*AttachmentStorageMetadata, error) {
	// Generate S3 key for attachment
	s3Key := s.generateAttachmentKey(userID, messageID, filename)

	// Read content to get size
	contentBytes, err := io.ReadAll(content)
	if err != nil {
		return nil, fmt.Errorf("failed to read attachment content: %w", err)
	}

	// Upload to S3
	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucketName),
		Key:         aws.String(s3Key),
		Body:        bytes.NewReader(contentBytes),
		ContentType: aws.String(contentType),
		Metadata: map[string]string{
			"user-id":    userID,
			"message-id": messageID,
			"filename":   filename,
			"type":       "attachment",
			"stored-at":  time.Now().Format(time.RFC3339),
		},
		ServerSideEncryption: types.ServerSideEncryptionAes256,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to store attachment: %w", err)
	}

	metadata := &AttachmentStorageMetadata{
		EmailID:     0, // Will be set by caller
		UserID:      0, // String userID converted for legacy compatibility
		Filename:    filename,
		S3Key:       s3Key,
		ContentType: contentType,
		Size:        int64(len(contentBytes)),
		StoredAt:    time.Now(),
		Metadata: map[string]string{
			"type":       "attachment",
			"message_id": messageID,
		},
	}

	log.Printf("Stored attachment %s for user %s, message %s at %s", filename, userID, messageID, s3Key)
	return metadata, nil
}

// RetrieveEmailBody retrieves email body from S3
func (s *S3Service) RetrieveEmailBody(ctx context.Context, s3Key string) (string, error) {
	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(s3Key),
	})
	if err != nil {
		return "", fmt.Errorf("failed to retrieve email body: %w", err)
	}
	defer result.Body.Close()

	bodyBytes, err := io.ReadAll(result.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read email body: %w", err)
	}

	return string(bodyBytes), nil
}

// RetrieveAttachment retrieves attachment from S3
func (s *S3Service) RetrieveAttachment(ctx context.Context, s3Key string) (io.ReadCloser, string, error) {
	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(s3Key),
	})
	if err != nil {
		return nil, "", fmt.Errorf("failed to retrieve attachment: %w", err)
	}

	contentType := "application/octet-stream"
	if result.ContentType != nil {
		contentType = *result.ContentType
	}

	return result.Body, contentType, nil
}

// DeleteEmailBody deletes email body from S3
func (s *S3Service) DeleteEmailBody(ctx context.Context, s3Key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(s3Key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete email body: %w", err)
	}

	log.Printf("Deleted email body: %s", s3Key)
	return nil
}

// DeleteAttachment deletes attachment from S3
func (s *S3Service) DeleteAttachment(ctx context.Context, s3Key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(s3Key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete attachment: %w", err)
	}

	log.Printf("Deleted attachment: %s", s3Key)
	return nil
}

// GeneratePresignedURL generates a presigned URL for direct access
func (s *S3Service) GeneratePresignedURL(ctx context.Context, s3Key string, expiration time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(s.client)

	request, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(s3Key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = expiration
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return request.URL, nil
}

// AttachmentInfo holds information about an attachment
type AttachmentInfo struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
	URL         string `json:"url"`
	S3Key       string `json:"s3_key"`
}

// GetAttachmentInfo retrieves attachment metadata and generates a presigned URL
func (s *S3Service) GetAttachmentInfo(ctx context.Context, s3Key string, urlExpiration time.Duration) (*AttachmentInfo, error) {
	// Get object metadata
	headResult, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(s3Key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get attachment metadata: %w", err)
	}

	// Extract filename from metadata or S3 key
	filename := "unknown"
	if headResult.Metadata != nil {
		if fn, ok := headResult.Metadata["filename"]; ok {
			filename = fn
		}
	}
	// If filename not in metadata, extract from S3 key
	if filename == "unknown" {
		parts := strings.Split(s3Key, "/")
		if len(parts) > 0 {
			// Get last part and remove messageID prefix
			lastPart := parts[len(parts)-1]
			// Remove messageID_ prefix if present
			if idx := strings.Index(lastPart, "_"); idx != -1 && idx < len(lastPart)-1 {
				filename = lastPart[idx+1:]
			} else {
				filename = lastPart
			}
		}
	}

	// Get content type
	contentType := "application/octet-stream"
	if headResult.ContentType != nil {
		contentType = *headResult.ContentType
	}

	// Get size
	size := int64(0)
	if headResult.ContentLength != nil {
		size = *headResult.ContentLength
	}

	// Generate presigned URL
	url, err := s.GeneratePresignedURL(ctx, s3Key, urlExpiration)
	if err != nil {
		return nil, fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return &AttachmentInfo{
		Filename:    filename,
		ContentType: contentType,
		Size:        size,
		URL:         url,
		S3Key:       s3Key,
	}, nil
}

// ListUserEmails lists all stored emails for a user
func (s *S3Service) ListUserEmails(ctx context.Context, userID int) ([]string, error) {
	prefix := fmt.Sprintf("emails/user_%d/", userID)

	result, err := s.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucketName),
		Prefix: aws.String(prefix),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list user emails: %w", err)
	}

	var keys []string
	for _, obj := range result.Contents {
		if obj.Key != nil {
			keys = append(keys, *obj.Key)
		}
	}

	return keys, nil
}

// GetStorageStats returns storage statistics
func (s *S3Service) GetStorageStats(ctx context.Context) (map[string]interface{}, error) {
	// List all objects to calculate stats
	result, err := s.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucketName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get storage stats: %w", err)
	}

	stats := map[string]interface{}{
		"total_objects": len(result.Contents),
		"total_size":    int64(0),
		"email_bodies":  0,
		"attachments":   0,
	}

	for _, obj := range result.Contents {
		if obj.Size != nil {
			stats["total_size"] = stats["total_size"].(int64) + *obj.Size
		}
		if obj.Key != nil {
			if strings.Contains(*obj.Key, "/bodies/") {
				stats["email_bodies"] = stats["email_bodies"].(int) + 1
			} else if strings.Contains(*obj.Key, "/attachments/") {
				stats["attachments"] = stats["attachments"].(int) + 1
			}
		}
	}

	return stats, nil
}

// Helper methods for generating S3 keys
func (s *S3Service) generateEmailBodyKey(userID, messageID string) string {
	timestamp := time.Now().Format("2006/01/02")
	return fmt.Sprintf("emails/user_%s/bodies/%s/%s.html", userID, timestamp, messageID)
}

func (s *S3Service) generateAttachmentKey(userID, messageID, filename string) string {
	timestamp := time.Now().Format("2006/01/02")
	safeFilename := strings.ReplaceAll(filename, " ", "_")
	ext := filepath.Ext(safeFilename)
	name := strings.TrimSuffix(safeFilename, ext)
	return fmt.Sprintf("emails/user_%s/attachments/%s/%s_%s%s", userID, timestamp, messageID, name, ext)
}

// Close closes any resources (S3 client doesn't need explicit closing)
// DeleteUserData deletes all data for a specific user from S3
func (s *S3Service) DeleteUserData(ctx context.Context, userID int) error {
	// List all objects with user prefix
	userPrefix := fmt.Sprintf("users/%d/", userID)

	listInput := &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucketName),
		Prefix: aws.String(userPrefix),
	}

	var objectsToDelete []types.ObjectIdentifier

	// Paginate through all objects
	paginator := s3.NewListObjectsV2Paginator(s.client, listInput)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to list objects: %w", err)
		}

		for _, obj := range page.Contents {
			objectsToDelete = append(objectsToDelete, types.ObjectIdentifier{
				Key: obj.Key,
			})
		}
	}

	// Delete objects in batches (S3 allows max 1000 objects per delete request)
	if len(objectsToDelete) == 0 {
		log.Printf("No objects found for user %d", userID)
		return nil
	}

	for i := 0; i < len(objectsToDelete); i += 1000 {
		end := i + 1000
		if end > len(objectsToDelete) {
			end = len(objectsToDelete)
		}

		batch := objectsToDelete[i:end]
		deleteInput := &s3.DeleteObjectsInput{
			Bucket: aws.String(s.bucketName),
			Delete: &types.Delete{
				Objects: batch,
				Quiet:   aws.Bool(true),
			},
		}

		_, err := s.client.DeleteObjects(ctx, deleteInput)
		if err != nil {
			return fmt.Errorf("failed to delete objects batch: %w", err)
		}
	}

	log.Printf("Successfully deleted %d objects for user %d", len(objectsToDelete), userID)
	return nil
}

func (s *S3Service) Close() error {
	// S3 client doesn't need explicit closing
	return nil
}
