package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type InitialSubmissionRequest struct {
	Email  string `json:"email"`
	Secret string `json:"secret"`
	URL    string `json:"url"`
	Answer string `json:"answer"`
}

type InitialSubmissionResponse struct {
	Correct bool   `json:"correct"`
	Reason  string `json:"reason"`
	URL     string `json:"url"`
	Delay   *int   `json:"delay"`
}

func generateBeepPayload() []byte {
	data := make([]byte, 160)

	for i := 0; i < len(data); i++ {
		if i%20 < 10 {
			data[i] = 0xFF
		} else {
			data[i] = 0x7F
		}
	}

	return data
}

func SubmitInitialRequest(email, secret, answer string) (*InitialSubmissionResponse, error) {
	req := InitialSubmissionRequest{
		Email:  email,
		Secret: secret,
		URL:    "https://tds-llm-analysis.s-anand.net/project2",
		Answer: answer,
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	resp, err := http.Post(
		"https://tds-llm-analysis.s-anand.net/submit",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("error making POST request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response InitialSubmissionResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	return &response, nil
}

// MakeInitialSubmission makes the initial submission using environment variables
// This can be called to initiate the first request programmatically
func MakeInitialSubmission(answer string) (*InitialSubmissionResponse, error) {
	email := os.Getenv("SUBMISSION_EMAIL")
	secret := os.Getenv("SECRET")

	if email == "" {
		return nil, fmt.Errorf("SUBMISSION_EMAIL environment variable not set")
	}
	if secret == "" {
		return nil, fmt.Errorf("SECRET environment variable not set")
	}

	return ProcessInitialSubmissionFlow(email, secret, answer)
}

// ExtractURLAndID extracts the next URL and ID from the initial submission response
func ExtractURLAndID(response *InitialSubmissionResponse) (nextURL string, id string) {
	if response == nil {
		return "", ""
	}

	// The URL format should be something like:
	// https://tds-llm-analysis.s-anand.net/project2-uv?email=XXX%40ds.study.iitm.ac.in&id=40960
	// We can extract the full URL and parse the ID parameter if needed

	return response.URL, ""
}

// ProcessInitialSubmissionFlow handles the complete flow from initial submission to creating ingest and quiz session
func ProcessInitialSubmissionFlow(email, secret, answer string) (*InitialSubmissionResponse, error) {
	// Make the initial submission
	response, err := SubmitInitialRequest(email, secret, answer)
	if err != nil {
		return nil, fmt.Errorf("initial submission failed: %v", err)
	}

	// If successful and we got a URL, create the ingest record and quiz session
	if response.Correct && response.URL != "" {
		// Create ingest record
		taskReq := TaskRequest{
			Email:  email,
			Secret: secret,
			Url:    response.URL, // Use the returned URL from the initial submission
		}

		// Create ingest record first
		err = Ingest(taskReq)
		if err != nil {
			return response, fmt.Errorf("failed to create ingest: %v", err)
		}

		// Get the created ingest to link with quiz session
		var ingest Ingests
		if err := DB.Where("email = ? AND url = ?", email, response.URL).First(&ingest).Error; err != nil {
			return response, fmt.Errorf("failed to find created ingest: %v", err)
		}

		// Start quiz session with ingest link
		err = StartQuizSessionWithIngest(taskReq, ingest.ID)
		if err != nil {
			return response, fmt.Errorf("failed to start quiz session: %v", err)
		}
	}

	return response, nil
}
