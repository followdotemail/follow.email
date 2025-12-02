package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

type LabelColor struct {
	BackgroundColor string `json:"background_color,omitempty"`
	TextColor       string `json:"text_color,omitempty"`
}

// Scan implements sql.Scanner interface for reading JSONB from database
func (lc *LabelColor) Scan(value interface{}) error {
	if value == nil {
		*lc = LabelColor{}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to scan LabelColor: invalid type")
	}

	if len(bytes) == 0 {
		*lc = LabelColor{}
		return nil
	}

	return json.Unmarshal(bytes, lc)
}

// Value implements driver.Valuer interface for writing JSONB to database
func (lc LabelColor) Value() (driver.Value, error) {
	if lc.BackgroundColor == "" && lc.TextColor == "" {
		return nil, nil
	}
	return json.Marshal(lc)
}

type UserLabel struct {
	ID                    int        `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID                uuid.UUID  `json:"user_id" gorm:"not null;index"`
	GmailLabelID          string     `json:"gmail_label_id" gorm:"not null;index"`
	LabelName             string     `json:"label_name" gorm:"not null"`
	Color                 LabelColor `json:"color" gorm:"type:jsonb"`
	MessageListVisibility string     `json:"message_list_visibility" gorm:"not null"`
	LabelListVisibility   string     `json:"label_list_visibility" gorm:"not null"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

func (UserLabel) TableName() string {
	return "user_labels"
}

func (l *UserLabel) HasColor() bool {
	return l.Color.BackgroundColor != "" || l.Color.TextColor != ""
}

func (l *UserLabel) GetColor() *LabelColor {
	if l.Color.BackgroundColor == "" && l.Color.TextColor == "" {
		return nil
	}

	return &l.Color
}
