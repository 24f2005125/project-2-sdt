package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

const NOTIFICATION_TEXT = "We have an ingest task for project-2-sdt !"

func NotifyJob() {
	var ingests []Ingests
	if err := DB.Where("status = ?", IngestStatusPending).Find(&ingests).Error; err != nil {
		return
	}

	for _, ingest := range ingests {
		notification := NOTIFICATION_TEXT + "\n"
		notification += "\nIngest ID: " + string(rune(ingest.ID))
		notification += "\nEmail: " + ingest.Email
		notification += "\nURL: " + ingest.URL

		err := SendNotification(notification)

		if err != nil {
			continue
		}
	}
}

func SendNotification(message string) error {
	_, err := http.Post(fmt.Sprintf("https://ntfy.sh/%s", os.Getenv("NTFY_TOPIC")),
		"text/plain",
		strings.NewReader(message),
	)

	log.Println("Notification sent:", message)

	return err
}
