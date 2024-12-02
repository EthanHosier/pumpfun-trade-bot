package blockchain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

func (c *BlockchainClient) getTransactionData(signature string) (*Transaction, error) {
	requestBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      uuid.New().String(),
		"method":  "getTransactionWithCompressionInfo",
		"params": []interface{}{
			signature,
		},
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	// Make the POST request
	resp, err := http.Post(
		restEndpoint+c.apiKey,
		"application/json",
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	var response struct {
		Result struct {
			Transaction Transaction `json:"transaction"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	// Check for error response
	var errorResp struct {
		Error struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &errorResp); err == nil && errorResp.Error.Code != 0 {
		return nil, fmt.Errorf("Error getting transaction data: %s", errorResp.Error.Message)
	}

	return &response.Result.Transaction, nil
}

// //////////////////////////////////////////////////////////////////////////////////////////////////////
// Helper function to subscribe to a single wallet's transactions
func (b *BlockchainClient) subscribeToWalletTransactions(conn *websocket.Conn, wallet string) (int, error) {
	subscriptionMessage := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "logsSubscribe",
		"params": []interface{}{
			map[string]interface{}{
				"mentions": []string{wallet},
			},
			map[string]interface{}{
				"commitment": commitment,
			},
		},
	}

	if err := conn.WriteJSON(subscriptionMessage); err != nil {
		return 0, fmt.Errorf("failed to send subscription message for wallet %s: %v", wallet, err)
	}

	// Create a channel to receive the ReadJSON result
	readDone := make(chan error, 1)
	var subscriptionResponse struct {
		Result int `json:"result"`
	}

	// Start reading in a goroutine
	go func() {
		readDone <- conn.ReadJSON(&subscriptionResponse)
	}()

	// Wait for either timeout or successful read
	select {
	case err := <-readDone:
		if err != nil {
			return 0, fmt.Errorf("failed to read subscription response for wallet %s: %v", wallet, err)
		}
	case <-time.After(subscribeReadTimeout):
		return 0, fmt.Errorf("timeout waiting for subscription response for wallet %s", wallet)
	}

	log.Printf("Subscription response for wallet %s: %v", wallet, subscriptionResponse.Result)
	return subscriptionResponse.Result, nil
}

func (b *BlockchainClient) transactionSignaturesLoop(conn *websocket.Conn, done <-chan interface{}, walletTransactionSignaturesCh chan<- WalletTransactionSignature, errCh chan<- error, subscriptionToWallet map[int]string) {
	defer conn.Close()

	msgCh := make(chan []byte, channelBufferSize) // maybe this can be unbuffered??
	msgChDone := make(chan struct{})

	// TODO: this is a bit tecky, make this cleaner so done ch makes this exit immediately
	go func(doneCh <-chan struct{}) {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				errCh <- fmt.Errorf("websocket read error: %v", err)
				return
			}
			msgCh <- message
		}
	}(msgChDone)

	for {
		// Check for done signal first
		select {
		case <-done:
			log.Println("Done signal received, exiting transaction signatures loop")
			return
		case message := <-msgCh:
			var response LogResponse
			if err := json.Unmarshal(message, &response); err != nil {
				errCh <- fmt.Errorf("websocket read error: %v", err)
				return
			}

			if response.Method == "logsNotification" {
				walletTransactionSignaturesCh <- WalletTransactionSignature{
					Signature: response.Params.Result.Value.Signature,
					Wallet:    subscriptionToWallet[response.Params.Subscription],
				}
			}
		}
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////////
