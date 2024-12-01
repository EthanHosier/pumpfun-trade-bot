package notifications

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	senderID        = "TRADES"
	clicksendSmsUrl = "https://rest.clicksend.com/v3/sms/send"
)

type Notifier interface {
	SendSMS(body string, to string) error
}

type ClicksendClient struct {
	username string
	apiKey   string
}

func NewClicksendClient(username, apiKey string) *ClicksendClient {
	return &ClicksendClient{
		username: username,
		apiKey:   apiKey,
	}
}

func (c *ClicksendClient) SendSMS(body string, to string) error {
	message := map[string]interface{}{
		"messages": []map[string]string{
			{
				"source": senderID,
				"to":     to,
				"body":   body,
			},
		},
	}

	jsonBody, err := json.Marshal(message)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", clicksendSmsUrl, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}

	auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", c.username, c.apiKey)))

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", auth))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send SMS: %s", resp.Status)
	}

	return nil
}
