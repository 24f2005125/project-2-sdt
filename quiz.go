package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type QuizResponse struct {
	Correct bool   `json:"correct"`
	URL     string `json:"url,omitempty"`
	Reason  string `json:"reason,omitempty"`
}

type AnswerSubmission struct {
	Email  string      `json:"email"`
	Secret string      `json:"secret"`
	URL    string      `json:"url"`
	Answer interface{} `json:"answer"`
}

// StartQuizSession starts a new quiz session for manual solving
func StartQuizSession(req TaskRequest) error {
	// Create a new quiz session
	session := QuizSession{
		Email:      req.Email,
		Secret:     req.Secret,
		CurrentURL: req.Url,
		Status:     "waiting_for_answer",
	}

	if err := DB.Create(&session).Error; err != nil {
		return fmt.Errorf("failed to create quiz session: %v", err)
	}

	// Update the corresponding ingest status
	if err := DB.Model(&Ingests{}).Where("email = ? AND url = ?", req.Email, req.Url).
		Update("status", IngestStatusRunning).Error; err != nil {
		return fmt.Errorf("failed to update ingest status: %v", err)
	}

	// Create initial attempt record
	attempt := QuizAttempt{
		SessionID: session.ID,
		URL:       req.Url,
		Question:  "Visit the URL to see the question",
		Answer:    "", // Explicitly set empty string
		Deadline:  time.Now().Add(3 * time.Minute),
	}

	if err := DB.Create(&attempt).Error; err != nil {
		return fmt.Errorf("failed to create quiz attempt: %v", err)
	}

	return nil
}

// StartQuizSessionWithIngest starts a new quiz session for manual solving with ingest link
func StartQuizSessionWithIngest(req TaskRequest, ingestID uint) error {
	// Create a new quiz session
	session := QuizSession{
		IngestID:   ingestID,
		Email:      req.Email,
		Secret:     req.Secret,
		CurrentURL: req.Url,
		Status:     "waiting_for_answer",
	}

	if err := DB.Create(&session).Error; err != nil {
		return fmt.Errorf("failed to create quiz session: %v", err)
	}

	// Update the corresponding ingest status
	if err := DB.Model(&Ingests{}).Where("id = ?", ingestID).
		Update("status", IngestStatusRunning).Error; err != nil {
		return fmt.Errorf("failed to update ingest status: %v", err)
	}

	// Create initial attempt record
	attempt := QuizAttempt{
		SessionID: session.ID,
		URL:       req.Url,
		Question:  "Visit the URL to see the question",
		Answer:    "", // Explicitly set empty string
		Deadline:  time.Now().Add(3 * time.Minute),
	}

	if err := DB.Create(&attempt).Error; err != nil {
		return fmt.Errorf("failed to create quiz attempt: %v", err)
	}

	return nil
}

// SubmitManualAnswer submits a manually provided answer to a custom submit URL
func SubmitManualAnswer(sessionID uint, answerData interface{}, submitURL string) (*QuizResponse, error) {
	var session QuizSession
	if err := DB.First(&session, sessionID).Error; err != nil {
		return nil, fmt.Errorf("failed to find quiz session: %v", err)
	}

	// Find the latest pending attempt for this session that hasn't expired
	var attempt QuizAttempt
	if err := DB.Where("session_id = ? AND answer = '' AND (deadline > ? OR deadline IS NULL OR deadline = ?)", sessionID, time.Now(), time.Time{}).
		Order("created_at DESC").
		First(&attempt).Error; err != nil {
		return nil, fmt.Errorf("no pending attempt found or attempt expired: %v", err)
	}

	// Submit the answer directly as provided (not wrapped in submission structure)
	response, err := SubmitRawAnswer(submitURL, answerData)
	if err != nil {
		return nil, fmt.Errorf("failed to submit answer: %v", err)
	}

	// Update the attempt with the response
	answerJSON, _ := json.Marshal(answerData)
	attempt.Answer = string(answerJSON)
	attempt.SubmitURL = submitURL
	attempt.Correct = &response.Correct
	attempt.NextURL = response.URL
	attempt.Reason = response.Reason
	responseJSON, _ := json.Marshal(response)
	attempt.ResponseRaw = string(responseJSON)

	if err := DB.Save(&attempt).Error; err != nil {
		return nil, fmt.Errorf("failed to update quiz attempt: %v", err)
	}

	// If we have a next URL, create a new attempt and update session
	if response.URL != "" {
		session.CurrentURL = response.URL
		if err := DB.Save(&session).Error; err != nil {
			return nil, fmt.Errorf("failed to update session: %v", err)
		}

		// Create next attempt
		nextAttempt := QuizAttempt{
			SessionID: sessionID,
			URL:       response.URL,
			Question:  "Visit the URL to see the next question",
			Deadline:  time.Now().Add(3 * time.Minute),
		}

		if err := DB.Create(&nextAttempt).Error; err != nil {
			return nil, fmt.Errorf("failed to create next attempt: %v", err)
		}

		// Extend the ingest deadline to match the new attempt
		if err := DB.Model(&Ingests{}).Where("email = ?", session.Email).
			Update("deadline", nextAttempt.Deadline).Error; err != nil {
			return nil, fmt.Errorf("failed to update ingest deadline: %v", err)
		}
	} else if response.Correct {
		// Quiz completed successfully (correct answer and no next URL)
		session.Status = "completed"
		if err := DB.Save(&session).Error; err != nil {
			return nil, fmt.Errorf("failed to update session status: %v", err)
		}

		// Update ingest status
		DB.Model(&Ingests{}).Where("email = ?", session.Email).
			Update("status", IngestStatusCompleted)
	} else {
		// Answer is incorrect and no next URL provided - create a new attempt for retry
		// Retry attempts inherit the original attempt's deadline (no extension)
		retryAttempt := QuizAttempt{
			SessionID: sessionID,
			URL:       attempt.URL, // Keep the same URL for retry
			Question:  "Answer was incorrect. You can retry within the remaining time window.",
			Deadline:  attempt.Deadline, // Keep the same deadline as the original attempt
		}

		if err := DB.Create(&retryAttempt).Error; err != nil {
			return nil, fmt.Errorf("failed to create retry attempt: %v", err)
		}
		// Note: No deadline extension for retry attempts - users must retry within the original 3-minute window
	}

	return response, nil
}

// SubmitRawAnswer submits exactly the answer data provided without wrapping
func SubmitRawAnswer(submitURL string, answer any) (*QuizResponse, error) {
	// Submit EXACTLY the answer JSON provided by the user
	jsonData, err := json.Marshal(answer)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal answer: %v", err)
	}

	resp, err := http.Post(submitURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to submit answer: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	var quizResp QuizResponse
	if err := json.Unmarshal(body, &quizResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	return &quizResp, nil
}

// GetCurrentAttempt gets the current pending attempt for a session that hasn't expired
func GetCurrentAttempt(sessionID uint) (*QuizAttempt, error) {
	var attempt QuizAttempt
	err := DB.Where("session_id = ? AND (answer = '' OR answer IS NULL) AND (deadline > ? OR deadline IS NULL OR deadline = ?)", sessionID, time.Now(), time.Time{}).
		Order("created_at DESC").
		First(&attempt).Error

	if err != nil {
		return nil, err
	}

	return &attempt, nil
}

// GetPendingAttempts returns attempts that are waiting for answers and haven't expired
func GetPendingAttempts() ([]QuizAttempt, error) {
	var attempts []QuizAttempt
	err := DB.Where("(answer = '' OR answer IS NULL) AND (deadline > ? OR deadline IS NULL OR deadline = ?)", time.Now(), time.Time{}).
		Order("created_at ASC").
		Find(&attempts).Error
	return attempts, err
}

// GetQuizSessions returns all quiz sessions
func GetQuizSessions() ([]QuizSession, error) {
	var sessions []QuizSession
	err := DB.Order("created_at DESC").Find(&sessions).Error
	return sessions, err
}

// GetQuizAttempts returns all attempts for a session
func GetQuizAttempts(sessionID uint) ([]QuizAttempt, error) {
	var attempts []QuizAttempt
	err := DB.Where("session_id = ?", sessionID).
		Order("created_at ASC").
		Find(&attempts).Error
	return attempts, err
}

// ExtendSessionDeadline extends the deadline for all ingests matching the email
func ExtendSessionDeadline(email string, extension time.Duration) error {
	return DB.Model(&Ingests{}).
		Where("email = ? AND status IN (?, ?)", email, IngestStatusRunning, IngestStatusNotified).
		Update("deadline", DB.Raw("datetime(deadline, '+' || ? || ' seconds')", int(extension.Seconds()))).Error
}
