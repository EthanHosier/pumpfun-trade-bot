package notifications

import (
	"github.com/joho/godotenv"
	"os"
	"testing"
)

func init() {
	// Load .env file if it exists
	if err := godotenv.Load("../.env"); err != nil {
		// It's okay if .env doesn't exist for CI/CD
		println("No .env file found")
	}
}

func TestSendSMS(t *testing.T) {
	username := os.Getenv("CLICKSEND_USERNAME")
	apiKey := os.Getenv("CLICKSEND_API_KEY")

	if username == "" || apiKey == "" {
		t.Skip("Skipping test: CLICKSEND_USERNAME and/or CLICKSEND_API_KEY environment variables not set")
	}

	client := NewClicksendClient(username, apiKey)

	err := client.SendSMS("Hello, world!", "+447476133726")
	if err != nil {
		t.Errorf("Error sending SMS: %v", err)
	}
}
