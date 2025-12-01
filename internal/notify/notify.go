package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

var webhookURL string

func Init(url string) {
	webhookURL = url
}

func Send(msg string) {
	if webhookURL == "" {
		// Try to load from env if not set (fallback)
		webhookURL = os.Getenv("DISCORD_WEBHOOK_URL")
		if webhookURL == "" {
			return
		}
	}

	// Rate limit protection (simple)
	// We don't want to spam Discord if we are in a loop
	// But for debugging "3K requests", maybe we WANT to see the spam?
	// Let's just send it.

	payload := map[string]string{
		"content": fmt.Sprintf("🔍 **Debug**: %s", msg),
	}
	
	jsonBody, _ := json.Marshal(payload)
	
	// Run in goroutine to avoid blocking main flow
	go func() {
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Post(webhookURL, "application/json", bytes.NewBuffer(jsonBody))
		if err != nil {
			log.Printf("Failed to send Discord log: %v", err)
			return
		}
		defer resp.Body.Close()
	}()
}
