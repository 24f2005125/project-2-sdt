package main

import (
	"log"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	_ "github.com/glebarez/go-sqlite"
)

type IngestStatus string

const (
	IngestStatusPending   IngestStatus = "Pending"
	IngestStatusNotified  IngestStatus = "Notification Accepted"
	IngestStatusRunning   IngestStatus = "Running"
	IngestStatusCompleted IngestStatus = "Completed"
	IngestStatusFailed    IngestStatus = "Failed"
)

type Ingests struct {
	ID     uint   `json:"id" gorm:"primaryKey;autoIncrement"`
	Email  string `json:"email"`
	Secret string `json:"-"`
	URL    string `json:"url"`
	Raw    string `json:"raw"`

	Status    IngestStatus `json:"status"`
	CreatedAt time.Time    `json:"createdAt" gorm:"autoCreateTime"`
	Deadline  time.Time    `json:"deadline"`
}

type QuizSession struct {
	ID         uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	IngestID   uint      `json:"ingestId"`
	Email      string    `json:"email"`
	Secret     string    `json:"-"`
	CurrentURL string    `json:"currentUrl"`
	Status     string    `json:"status"` // "running", "completed", "failed"
	CreatedAt  time.Time `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt  time.Time `json:"updatedAt" gorm:"autoUpdateTime"`
}

type QuizAttempt struct {
	ID          uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	SessionID   uint      `json:"sessionId"`
	URL         string    `json:"url"`
	Question    string    `json:"question"`
	Answer      string    `json:"answer" gorm:"default:''"`
	SubmitURL   string    `json:"submitUrl"`
	Correct     *bool     `json:"correct" gorm:"default:null"`
	NextURL     string    `json:"nextUrl"`
	Reason      string    `json:"reason"`
	ResponseRaw string    `json:"responseRaw"`
	Deadline    time.Time `json:"deadline"` // 3 minutes from creation for attempts
	CreatedAt   time.Time `json:"createdAt" gorm:"autoCreateTime"`
}

var DB *gorm.DB

func InitDB() error {
	var err error

	DB, err = gorm.Open(sqlite.Open("data/app.db"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})

	if err != nil {
		log.Fatalf("failed to connect with gorm: %v", err)
		return err
	}

	if err := DB.AutoMigrate(&Ingests{}, &QuizSession{}, &QuizAttempt{}); err != nil {
		log.Fatalf("AutoMigrate failed: %v", err)
		return err
	}

	// Migration: Set deadline for existing attempts that don't have one
	if err := migrateExistingAttempts(); err != nil {
		log.Printf("Warning: Failed to migrate existing attempts: %v", err)
	}

	return nil
}

func migrateExistingAttempts() error {
	// Update attempts with empty/null deadlines to have a deadline 3 minutes from creation
	var count int64
	DB.Model(&QuizAttempt{}).Where("deadline IS NULL OR deadline = ?", time.Time{}).Count(&count)

	if count > 0 {
		log.Printf("Migrating %d existing quiz attempts with missing deadlines", count)
		return DB.Exec("UPDATE quiz_attempts SET deadline = datetime(created_at, '+3 minutes') WHERE deadline IS NULL OR deadline = ?", time.Time{}).Error
	}

	return nil
}
