package blockchain

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/associated-token-account"
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
	client *rpc.Client
}

func NewBlockchainClient(apiKey string) *BlockchainClient {
	return &BlockchainClient{apiKey, rpc.New(fmt.Sprintf("%s%s", restEndpoint, apiKey))}
}

func (b *BlockchainClient) GetTransactionDataWithRetries(signature string, maxRetries int) (*Transaction, error) {
	for i := 0; i < maxRetries; i++ {
		tx, err := b.getTransactionData(signature)
		if err == nil {
			return tx, nil
		}
		time.Sleep(500 * time.Millisecond)
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

func (b *BlockchainClient) SendSolanaToWallet(amountInSol float64, senderPrivateKey string, receiverPublicKey string) (string, error) {
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

	// Create transfer instruction
	transferIx := system.NewTransferInstruction(
		uint64(amountInSol*lamportsPerSol),
		privateKey.PublicKey(),
		receiver,
	).Build()

	// Get latest blockhash (replacing GetRecentBlockhash)
	recent, err := b.client.GetLatestBlockhash(context.Background(), rpc.CommitmentFinalized)
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
	sig, err := b.client.SendTransaction(context.Background(), tx)
	if err != nil {
		return "", fmt.Errorf("failed to send transaction: %v", err)
	}

	return sig.String(), nil
}

// func (b *BlockchainClient) BuyToken(tokenMint string, amountInSol float64, senderPrivateKey string, receiverPublicKey string) (string, error) {
// 	privateKey, err := solana.PrivateKeyFromBase58(senderPrivateKey)
// 	if err != nil {
// 		return "", fmt.Errorf("failed to decode private key: %v", err)
// 	}
// 	from := privateKey.PublicKey()

// 	tm, err := solana.PublicKeyFromBase58(tokenMint)
// 	if err != nil {
// 		return "", fmt.Errorf("invalid token mint: %v", err)
// 	}
// }

// GetOrCreateTokenAccount returns the associated token account address and (OPTIONAL) the transaction signature
func (b *BlockchainClient) GetOrCreateTokenAccount(tokenMint string, ownerPrivateKey string) (string, string, error) {
	// Parse private key and get public key
	privateKey, err := solana.PrivateKeyFromBase58(ownerPrivateKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to decode private key: %v", err)
	}
	owner := privateKey.PublicKey()

	// Parse token mint address
	mint, err := solana.PublicKeyFromBase58(tokenMint)
	if err != nil {
		return "", "", fmt.Errorf("invalid token mint address: %v", err)
	}

	// Find the associated token account address
	ata, _, err := solana.FindAssociatedTokenAddress(owner, mint)
	if err != nil {
		return "", "", fmt.Errorf("failed to find associated token address: %v", err)
	}

	// Check if the account already exists
	account, err := b.client.GetAccountInfo(context.Background(), ata)
	if err == nil && account != nil {
		log.Printf("Account already exists: %s", ata.String())
		return ata.String(), "", nil // Account already exists, so no transaction signature
	}

	// Create the instruction to create the associated token account
	createATAIx := associatedtokenaccount.NewCreateInstruction(
		owner, // payer
		owner, // wallet owner
		mint,  // token mint
	).Build()

	// Get latest blockhash
	recent, err := b.client.GetLatestBlockhash(context.Background(), rpc.CommitmentFinalized)
	if err != nil {
		return "", "", fmt.Errorf("failed to get latest blockhash: %v", err)
	}

	// Build transaction
	tx, err := solana.NewTransaction(
		[]solana.Instruction{createATAIx},
		recent.Value.Blockhash,
		solana.TransactionPayer(owner),
	)
	if err != nil {
		return "", "", fmt.Errorf("failed to create transaction: %v", err)
	}

	// Sign transaction
	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if owner.Equals(key) {
			return &privateKey
		}
		return nil
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to sign transaction: %v", err)
	}

	// Send transaction
	sig, err := b.client.SendTransaction(context.Background(), tx)
	if err != nil {
		return "", "", fmt.Errorf("failed to send transaction: %v", err)
	}

	log.Printf("Transaction sent: %s", sig.String())

	return ata.String(), sig.String(), nil
}
