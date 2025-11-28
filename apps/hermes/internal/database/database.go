package database

import (
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"follow-email-backend/config"
	"follow-email-backend/internal/models"
)

func Connect(cfg *config.Config) (*gorm.DB, error) {
	// Build GORM config
	var gormConfig *gorm.Config
	if cfg.Environment == "production" {
		gormConfig = &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)}
	} else {
		gormConfig = &gorm.Config{Logger: logger.Default.LogMode(logger.Warn)} // Reduced logging
	}

	// Open connection with GORM using DATABASE_URL from config
	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Neon database: %v", err)
	}

	// Get underlying sql.DB for connection pool configuration
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %v", err)
	}

	// Configure connection pool for stability
	sqlDB.SetMaxOpenConns(25)                 // Maximum open connections
	sqlDB.SetMaxIdleConns(5)                  // Maximum idle connections
	sqlDB.SetConnMaxLifetime(5 * time.Minute) // Maximum connection lifetime
	sqlDB.SetConnMaxIdleTime(1 * time.Minute) // Maximum idle time

	// Test the connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	return db, nil
}

// Migrate runs database migrations using GORM AutoMigrate
func Migrate(db *gorm.DB) error {
	// Auto-migrate the schema for all models
	return db.AutoMigrate(
		&models.User{},
		&models.OAuthToken{},
		&models.UserPrivacyMetadata{},
		&models.GmailConsent{},
	)
}

// Health checks database connectivity
func Health(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %v", err)
	}
	return sqlDB.Ping()
}

// Close closes the database connection
func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %v", err)
	}
	return sqlDB.Close()
}
