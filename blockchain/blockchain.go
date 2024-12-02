package blockchain

import (
	"context"
	"fmt"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gorilla/websocket"
)

const (
	lamportsPerSol = 1_000_000_000

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

func (b *BlockchainClient) SendSolana(amountInSol float64, senderPrivateKey string, receiverPublicKey string) (string, error) {
	// Decode private key from base58
	privateKey, err := solana.PrivateKeyFromBase58(senderPrivateKey)
	if err != nil {
		return "", fmt.Errorf("failed to decode private key: %v", err)
	}

	// Parse receiver's public key
	receiver, err := solana.PublicKeyFromBase58(receiverPublicKey)
	if err != nil {
		return "", fmt.Errorf("invalid receiver public key: %v", err)
	}

	// Create RPC client
	client := rpc.New(fmt.Sprintf("%s%s", restEndpoint, b.apiKey))

	// Create transfer instruction
	transferIx := system.NewTransferInstruction(
		uint64(amountInSol*lamportsPerSol),
		privateKey.PublicKey(),
		receiver,
	).Build()

	// Get latest blockhash (replacing GetRecentBlockhash)
	recent, err := client.GetLatestBlockhash(context.Background(), rpc.CommitmentFinalized)
	if err != nil {
		return "", fmt.Errorf("failed to get latest blockhash: %v", err)
	}

	// Build transaction (updated to use new blockhash response)
	tx, err := solana.NewTransaction(
		[]solana.Instruction{transferIx},
		recent.Value.Blockhash,
		solana.TransactionPayer(privateKey.PublicKey()),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create transaction: %v", err)
	}

	// Sign transaction
	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if privateKey.PublicKey().Equals(key) {
			return &privateKey
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("failed to sign transaction: %v", err)
	}

	// Send transaction
	sig, err := client.SendTransaction(context.Background(), tx)
	if err != nil {
		return "", fmt.Errorf("failed to send transaction: %v", err)
	}

	return sig.String(), nil
}
