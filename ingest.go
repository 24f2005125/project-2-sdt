package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type TaskRequest struct {
	Email  string `json:"email"`
	Secret string `json:"secret"`
	Url    string `json:"url"`
}

func Ingest(req TaskRequest) error {
	var ingest Ingests
	now := time.Now()

	reqJSON, _ := json.Marshal(req)

	ingest.Email = req.Email
	ingest.Secret = req.Secret
	ingest.URL = req.Url
	ingest.Status = IngestStatusPending
	ingest.CreatedAt = now
	ingest.Raw = string(reqJSON)
	ingest.Deadline = now.Add(3 * time.Minute)

	if err := DB.Create(&ingest).Error; err != nil {
		return err
	}

	return nil
}

func ListIngests() ([]Ingests, error) {
	var ingests []Ingests

	if err := DB.Find(&ingests).Error; err != nil {
		return nil, err
	}

	return ingests, nil
}

func AcceptIngest(id uint, password string) error {
	if password != os.Getenv("INGEST_ACCEPT_PASSWORD") {
		return fmt.Errorf("invalid password")
	}

	var ingest Ingests

	if err := DB.First(&ingest, id).Error; err != nil {
		return err
	}

	ingest.Status = IngestStatusNotified

	if err := DB.Save(&ingest).Error; err != nil {
		return err
	}

	return nil
}
