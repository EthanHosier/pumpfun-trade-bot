package blockchain

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/ethanhosier/pumpfun-trade-bot/coinInfo"
	"github.com/ethanhosier/pumpfun-trade-bot/utils"
	"github.com/gagliardetto/solana-go"
	computebudget "github.com/gagliardetto/solana-go/programs/compute-budget"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gorilla/websocket"
)

const (
	lamportsPerSol   = 1_000_000_000
	computeUnitLimit = 60816

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

func (b *BlockchainClient) BuyTokenWithSol(
	tokenMint string,
	bondingCurveAddress string,
	associatedBondingCurveAddress string,
	solAmount float64,
	slippage float64,
	privateKey string,
) (*BuyTokenResult, error) {

	blockhashTask := utils.DoAsync(func() (*rpc.GetLatestBlockhashResult, error) {
		return b.client.GetLatestBlockhash(context.Background(), rpc.CommitmentFinalized)
	})

	// Convert unique data to required formats
	mintPubKey, bondingCurvePubKey, associatedBondingCurvePubKey, err := pubKeysFrom(tokenMint, bondingCurveAddress, associatedBondingCurveAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pub keys: %w", err)
	}
	signer, err := solana.PrivateKeyFromBase58(privateKey)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}
	payerPubKey := signer.PublicKey()

	ata, ataCreateInstruction, err := b.getOrCreateTokenAccountInstruction(mintPubKey, signer)
	if err != nil {
		return nil, fmt.Errorf("failed to get or create associated token account: %w", err)
	}

	ataPubKey, err := solana.PublicKeyFromBase58(ata)
	if err != nil {
		return nil, fmt.Errorf("invalid associated token account address: %w", err)
	}

	price, err := b.coinInfoClient.PriceInSolFromBondingCurveAddress(bondingCurveAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get price in SOL from bonding curve address: %w", err)
	}

	// Replace the calculation with function call
	amountInLamports, maxAmountLamports, err := buyTokenAmountsFrom(solAmount, price, slippage)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate token amounts: %w", err)
	}

	buyInstruction := solana.NewInstruction(
		PUMP_PROGRAM, // PUMP_PROGRAM
		buyAccountsFrom(mintPubKey, bondingCurvePubKey, associatedBondingCurvePubKey, ataPubKey, payerPubKey), // AccountMetaSlice
		buyDataFrom(amountInLamports, maxAmountLamports),                                                      // Instruction data
	)

	// Send the transaction
	blockhash, err := utils.GetAsync(blockhashTask)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch recent blockhash: %w", err)
	}

	tx, err := solana.NewTransaction(
		buyInstructionsFrom(computeUnitLimit, ataCreateInstruction, buyInstruction),
		blockhash.Value.Blockhash,
		solana.TransactionPayer(payerPubKey),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create buy transaction: %w", err)
	}

	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if signer.PublicKey().Equals(key) {
			return &signer
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}

	txID, err := b.client.SendTransaction(context.Background(), tx)
	if err != nil {
		return nil, fmt.Errorf("failed to send buy transaction: %w", err) //TODO: handle case where blockhash invalid
	}

	fmt.Printf("Transaction successful. TXID: %s\n", txID)
	return &BuyTokenResult{TxID: txID.String(), AmountInLampts: amountInLamports, MaxAmountLampts: maxAmountLamports, AssociatedTokenAccountAddress: ata}, nil
}

func (b *BlockchainClient) SellTokenWithSol(
	tokenMint string,
	bondingCurveAddress string,
	associatedBondingCurveAddress string,
	associatedTokenAccountAddress string,
	slippage float64,
	privateKey string,
) (string, error) {
	// Get latest blockhash asynchronously
	blockhashTask := utils.DoAsync(func() (*rpc.GetLatestBlockhashResult, error) {
		return b.client.GetLatestBlockhash(context.Background(), rpc.CommitmentFinalized)
	})

	// Convert addresses to public keys
	mintPubKey, bondingCurvePubKey, associatedBondingCurvePubKey, err := pubKeysFrom(tokenMint, bondingCurveAddress, associatedBondingCurveAddress)
	if err != nil {
		return "", fmt.Errorf("failed to parse pub keys: %w", err)
	}

	// Parse private key
	signer, err := solana.PrivateKeyFromBase58(privateKey)
	if err != nil {
		return "", fmt.Errorf("invalid private key: %w", err)
	}

	// Get ATA public key
	ataPubKey, err := solana.PublicKeyFromBase58(associatedTokenAccountAddress)
	if err != nil {
		return "", fmt.Errorf("invalid associated token account address: %w", err)
	}

	// Get token balance
	tokenBalance, err := b.client.GetTokenAccountBalance(
		context.Background(),
		ataPubKey,
		rpc.CommitmentFinalized,
	)
	if err != nil {
		return "", fmt.Errorf("failed to get token balance: %w", err)
	}

	if tokenBalance.Value.Amount == "0" {
		return "", fmt.Errorf("no tokens to sell")
	}

	// Get price from bonding curve
	price, err := b.coinInfoClient.PriceInSolFromBondingCurveAddress(bondingCurveAddress)
	if err != nil {
		return "", fmt.Errorf("failed to get price from bonding curve: %w", err)
	}

	// Calculate minimum SOL output with slippage
	amount, err := strconv.ParseUint(tokenBalance.Value.Amount, 10, 64)
	if err != nil {
		return "", fmt.Errorf("failed to parse token amount: %w", err)
	}

	minSolOutput := uint64(float64(amount) * price * (1 - slippage))

	// Create sell instruction
	sellInstruction := solana.NewInstruction(
		PUMP_PROGRAM,
		sellAccountsFrom(mintPubKey, bondingCurvePubKey, associatedBondingCurvePubKey, ataPubKey, signer.PublicKey()),
		sellDataFrom(amount, minSolOutput),
	)

	// Get blockhash result
	blockhash, err := utils.GetAsync(blockhashTask)
	if err != nil {
		return "", fmt.Errorf("failed to get recent blockhash: %w", err)
	}

	// Create and sign transaction
	tx, err := solana.NewTransaction(
		[]solana.Instruction{
			computebudget.NewSetComputeUnitLimitInstruction(computeUnitLimit).Build(),
			sellInstruction,
		},
		blockhash.Value.Blockhash,
		solana.TransactionPayer(signer.PublicKey()),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create sell transaction: %w", err)
	}

	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if signer.PublicKey().Equals(key) {
			return &signer
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Send transaction
	sig, err := b.client.SendTransaction(context.Background(), tx)
	if err != nil {
		return "", fmt.Errorf("failed to send sell transaction: %w", err)
	}

	return sig.String(), nil
}
