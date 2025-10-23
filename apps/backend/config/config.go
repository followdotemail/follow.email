package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	// Server configuration
	Port string

	// Database configuration
	DatabaseURL string

	// Clerk configuration
	ClerkSecretKey      string
	ClerkPublishableKey string

	// Google OAuth configuration
	GoogleClientID     string
	GoogleClientSecret string
	BaseURL            string

	// JWT configuration (still needed for custom tokens)
	JWTSecret string

	// AI API configuration
	GeminiAPIKey string

	// Encryption configuration
	EncryptionKey string

	// AWS S3 configuration
	AWSRegion          string
	AWSAccessKeyID     string
	AWSSecretAccessKey string
	S3BucketName       string

	// RabbitMQ configuration
	RabbitMQURL string

	// QStash configuration
	QStashToken             string
	QStashCurrentSigningKey string
	QStashNextSigningKey    string

	// Application settings
	Environment string
	LogLevel    string
}

func Load() *Config {
	// Check environment first to determine if we should load .env file
	env := os.Getenv("ENVIRONMENT")

	// Only load .env file if not in staging or production
	if env != "staging" && env != "production" {
		if err := godotenv.Load(); err != nil {
			log.Println("No .env file found, using environment variables")
		}
	} else {
		log.Printf("Environment is %s, skipping .env file loading", env)
	}

	return &Config{
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://localhost/followemail?sslmode=disable"),

		ClerkSecretKey:      getEnv("CLERK_SECRET_KEY", ""),
		ClerkPublishableKey: getEnv("CLERK_PUBLISHABLE_KEY", ""),

		GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		BaseURL:            getEnv("BASE_URL", "http://localhost:8080"),

		JWTSecret: getEnv("JWT_SECRET", "your-secret-key"),

		GeminiAPIKey: getEnv("GEMINI_API_KEY", ""),

		EncryptionKey: getEnv("ENCRYPTION_KEY", ""),

		AWSRegion:          getEnv("AWS_REGION", "us-east-1"),
		AWSAccessKeyID:     getEnv("AWS_ACCESS_KEY_ID", ""),
		AWSSecretAccessKey: getEnv("AWS_SECRET_ACCESS_KEY", ""),
		S3BucketName:       getEnv("S3_BUCKET_NAME", "followemail-storage"),

		RabbitMQURL: getEnv("RABBITMQ_URL", "amqp://localhost:5672"),

		QStashToken:             getEnv("QSTASH_TOKEN", ""),
		QStashCurrentSigningKey: getEnv("QSTASH_CURRENT_SIGNING_KEY", ""),
		QStashNextSigningKey:    getEnv("QSTASH_NEXT_SIGNING_KEY", ""),

		Environment: getEnv("ENVIRONMENT", "development"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}
