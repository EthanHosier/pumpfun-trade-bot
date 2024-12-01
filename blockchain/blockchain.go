package blockchain

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	wsEndpoint   = "wss://mainnet.helius-rpc.com/?api-key=" // REMEMBER TO ADD THE %s BACK
	restEndpoint = "https://mainnet.helius-rpc.com/?api-key="

	channelBufferSize    = 10000
	commitment           = "processed"
	subscribeReadTimeout = 5 * time.Second
)

type BlockchainClient struct {
	apiKey string
}

func NewBlockchainClient(apiKey string) *BlockchainClient {
	return &BlockchainClient{apiKey}
}

func (b *BlockchainClient) GetTransactionDataWithRetries(signature string, maxRetries int) (*Transaction, error) {
	for i := 0; i < maxRetries; i++ {
		tx, err := b.getTransactionData(signature)
		if err == nil {
			return tx, nil
		}
	}

	return nil, fmt.Errorf("failed to get transaction data after %d retries", maxRetries)
}

func (b *BlockchainClient) SubscribeToWalletsTransactionSignatures(walletAddresses []string, done <-chan interface{}) (<-chan WalletTransactionSignature, <-chan error, error) {
	conn, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("%s%s", wsEndpoint, b.apiKey), nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to WebSocket: %v", err)
	}

	subscriptionToWallet := make(map[int]string)

	for _, wallet := range walletAddresses {
		subscriptionID, err := b.subscribeToWalletTransactions(conn, wallet)
		if err != nil {
			return nil, nil, err
		}
		subscriptionToWallet[subscriptionID] = wallet
	}

	walletTransactionSignaturesCh := make(chan WalletTransactionSignature, channelBufferSize)
	errCh := make(chan error)

	go b.transactionSignaturesLoop(conn, done, walletTransactionSignaturesCh, errCh, subscriptionToWallet)

	return walletTransactionSignaturesCh, errCh, nil
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
