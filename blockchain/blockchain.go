package blockchain

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"time"

	"github.com/ethanhosier/pumpfun-trade-bot/coinInfo"
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

var (
	PUMP_PROGRAM                            = solana.MustPublicKeyFromBase58("6EF8rrecthR5Dkzon8Nwu78hRvfCKubJ14M5uBEwF6P")
	PUMP_GLOBAL                             = solana.MustPublicKeyFromBase58("4wTV1YmiEkRvAtNtsSGPtUrqRYQMe5SKy2uB4Jjaxnjf")
	PUMP_EVENT_AUTHORITY                    = solana.MustPublicKeyFromBase58("Ce6TQqeHC9p8KetsN6JsjHK7UTZk7nasjjnr7XxXp9F1")
	PUMP_FEE                                = solana.MustPublicKeyFromBase58("CebN5WGQ4jvEPvsVU4EoHEpgzq1VV7AbicfhtW4xC9iM")
	SYSTEM_PROGRAM                          = solana.MustPublicKeyFromBase58("11111111111111111111111111111111")
	SYSTEM_TOKEN_PROGRAM                    = solana.MustPublicKeyFromBase58("TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA")
	SYSTEM_ASSOCIATED_TOKEN_ACCOUNT_PROGRAM = solana.MustPublicKeyFromBase58("ATokenGPvbdGVxr1b2hvZbsiqW5xWH25efTNsLJA8knL")
	SYSTEM_RENT                             = solana.MustPublicKeyFromBase58("SysvarRent111111111111111111111111111111111")
	SOL                                     = solana.MustPublicKeyFromBase58("So11111111111111111111111111111111111111112")
	LAMPORTS_PER_SOL                        = uint64(1_000_000_000)
)

type BlockchainClient struct {
	apiKey         string
	client         *rpc.Client
	coinInfoClient *coinInfo.CoinInfoClient
}

func NewBlockchainClient(apiKey string, coinInfoClient *coinInfo.CoinInfoClient) *BlockchainClient {
	return &BlockchainClient{apiKey, rpc.New(fmt.Sprintf("%s%s", restEndpoint, apiKey)), coinInfoClient}
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

// MAYBE CAN SKIP THIS??
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

func (b *BlockchainClient) BuyToken(
	mintAddress string,
	bondingCurveAddress string,
	associatedBondingCurveAddress string,
	solAmount float64,
	slippage float64,
	privateKey string,
) error {

	// Convert unique data to required formats
	mintPubKey := solana.MustPublicKeyFromBase58(mintAddress)
	bondingCurvePubKey := solana.MustPublicKeyFromBase58(bondingCurveAddress)
	associatedBondingCurvePubKey := solana.MustPublicKeyFromBase58(associatedBondingCurveAddress)
	signer, err := solana.PrivateKeyFromBase58(privateKey)
	if err != nil {
		return fmt.Errorf("invalid private key: %w", err)
	}
	payerPubKey := signer.PublicKey()

	// Create associated token account if it doesn't exist
	ata, sig, err := b.GetOrCreateTokenAccount(mintAddress, privateKey)
	if err != nil {
		return fmt.Errorf("failed to get or create associated token account: %w", err)
	}

	if sig != "" {
		_, err = b.GetTransactionDataWithRetries(sig, 12)
		if err != nil {
			return fmt.Errorf("failed to get transaction data: %w", err)
		}
	}

	ataPubKey, err := solana.PublicKeyFromBase58(ata)
	if err != nil {
		return fmt.Errorf("invalid associated token account address: %w", err)
	}

	price, err := b.coinInfoClient.PriceInSolFromBondingCurveAddress(bondingCurveAddress)
	if err != nil {
		return fmt.Errorf("failed to get price in SOL from bonding curve address: %w", err)
	}

	// Prepare buy instruction
	tokenAmount := solAmount / price
	amountInLamports := uint64(tokenAmount * lamportsPerSol)

	maxAmountLamports := uint64(float64(amountInLamports) * (1 + slippage))

	discriminator := make([]byte, 8)
	binary.LittleEndian.PutUint64(discriminator, 16927863322537952870)

	amountData := make([]byte, 8)
	binary.LittleEndian.PutUint64(amountData, amountInLamports)

	maxAmountData := make([]byte, 8)
	binary.LittleEndian.PutUint64(maxAmountData, maxAmountLamports)

	data := append(discriminator, append(amountData, maxAmountData...)...)

	accounts := []*solana.AccountMeta{
		solana.NewAccountMeta(PUMP_GLOBAL, false, false),                 // PUMP_GLOBAL
		solana.NewAccountMeta(PUMP_FEE, true, false),                     // PUMP_FEE
		solana.NewAccountMeta(mintPubKey, false, false),                  // Mint
		solana.NewAccountMeta(bondingCurvePubKey, true, false),           // Bonding Curve
		solana.NewAccountMeta(associatedBondingCurvePubKey, true, false), // Associated Bonding Curve
		solana.NewAccountMeta(ataPubKey, true, false),                    // Associated Token Account
		solana.NewAccountMeta(payerPubKey, true, true),                   // Payer
		solana.NewAccountMeta(SYSTEM_PROGRAM, false, false),              // SYSTEM_PROGRAM
		solana.NewAccountMeta(SYSTEM_TOKEN_PROGRAM, false, false),        // SYSTEM_TOKEN_PROGRAM
		solana.NewAccountMeta(SYSTEM_RENT, false, false),                 // SYSTEM_RENT
		solana.NewAccountMeta(PUMP_EVENT_AUTHORITY, false, false),        // PUMP_EVENT_AUTHORITY
		solana.NewAccountMeta(PUMP_PROGRAM, false, false),                // PUMP_PROGRAM
	}

	buyInstruction := solana.NewInstruction(
		solana.MustPublicKeyFromBase58("6EF8rrecthR5Dkzon8Nwu78hRvfCKubJ14M5uBEwF6P"), // PUMP_PROGRAM
		accounts, // AccountMetaSlice
		data,     // Instruction data
	)

	// Send the transaction
	blockhash, err := b.client.GetLatestBlockhash(context.Background(), rpc.CommitmentFinalized)
	if err != nil {
		return fmt.Errorf("failed to fetch recent blockhash: %w", err)
	}

	tx, err := solana.NewTransaction(
		[]solana.Instruction{buyInstruction},
		blockhash.Value.Blockhash,
		solana.TransactionPayer(payerPubKey),
	)
	if err != nil {
		return fmt.Errorf("failed to create buy transaction: %w", err)
	}

	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if signer.PublicKey().Equals(key) {
			return &signer
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to sign transaction: %w", err)
	}

	txID, err := b.client.SendTransaction(context.Background(), tx)
	if err != nil {
		return fmt.Errorf("failed to send buy transaction: %w", err)
	}

	fmt.Printf("Transaction successful. TXID: %s\n", txID)
	return nil
}
