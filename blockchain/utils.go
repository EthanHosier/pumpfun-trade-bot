package blockchain

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gagliardetto/solana-go"
	associatedtokenaccount "github.com/gagliardetto/solana-go/programs/associated-token-account"
	computebudget "github.com/gagliardetto/solana-go/programs/compute-budget"
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

func (b *BlockchainClient) getOrCreateTokenAccountInstruction(tokenMintPubKey solana.PublicKey, ownerPrivateKey solana.PrivateKey) (string, *associatedtokenaccount.Instruction, error) {
	owner := ownerPrivateKey.PublicKey()

	// Find the associated token account address
	ata, _, err := solana.FindAssociatedTokenAddress(owner, tokenMintPubKey)
	if err != nil {
		return "", nil, fmt.Errorf("failed to find associated token address: %v", err)
	}

	// Check if the account already exists
	account, err := b.client.GetAccountInfo(context.TODO(), ata)
	if err == nil && account != nil {
		log.Printf("Associated token account already exists: %s", ata.String())
		return ata.String(), nil, nil // Account already exists, so no transaction signature
	}

	// Create the instruction to create the associated token account
	createATAIx := associatedtokenaccount.NewCreateInstruction(
		owner,           // payer
		owner,           // wallet owner
		tokenMintPubKey, // token mint
	).Build()

	return ata.String(), createATAIx, nil
}

func buyDataFrom(amountInLamports uint64, maxAmountLamports uint64) []byte {
	discriminator := make([]byte, 8)
	binary.LittleEndian.PutUint64(discriminator, 16927863322537952870)

	amountData := make([]byte, 8)
	binary.LittleEndian.PutUint64(amountData, amountInLamports)

	maxAmountData := make([]byte, 8)
	binary.LittleEndian.PutUint64(maxAmountData, maxAmountLamports)

	return append(discriminator, append(amountData, maxAmountData...)...)
}

func buyAccountsFrom(
	mintPubKey solana.PublicKey,
	bondingCurvePubKey solana.PublicKey,
	associatedBondingCurvePubKey solana.PublicKey,
	ataPubKey solana.PublicKey,
	payerPubKey solana.PublicKey,
) []*solana.AccountMeta {
	return []*solana.AccountMeta{
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
}

func buyInstructionsFrom(computeUnitLimit uint32, ataCreateInstruction *associatedtokenaccount.Instruction, buyInstruction *solana.GenericInstruction) []solana.Instruction {
	computeUnitLimitInstruction := computebudget.NewSetComputeUnitLimitInstruction(
		computeUnitLimit,
	).Build()

	instructions := []solana.Instruction{computeUnitLimitInstruction}
	if ataCreateInstruction != nil {
		fmt.Printf("Adding ATA create instruction\n")
		instructions = append(instructions, ataCreateInstruction)
	} else {
		fmt.Printf("No ATA create instruction\n")
	}
	instructions = append(instructions, buyInstruction)
	return instructions
}

func pubKeysFrom(tokenMint string, bondingCurveAddress string, associatedBondingCurveAddress string) (solana.PublicKey, solana.PublicKey, solana.PublicKey, error) {
	mintPubKey, err := solana.PublicKeyFromBase58(tokenMint)
	if err != nil {
		return solana.PublicKey{}, solana.PublicKey{}, solana.PublicKey{}, fmt.Errorf("invalid token mint: %v", err)
	}
	bondingCurvePubKey, err := solana.PublicKeyFromBase58(bondingCurveAddress)
	if err != nil {
		return solana.PublicKey{}, solana.PublicKey{}, solana.PublicKey{}, fmt.Errorf("invalid bonding curve address: %v", err)
	}
	associatedBondingCurvePubKey, err := solana.PublicKeyFromBase58(associatedBondingCurveAddress)
	if err != nil {
		return solana.PublicKey{}, solana.PublicKey{}, solana.PublicKey{}, fmt.Errorf("invalid associated bonding curve address: %v", err)
	}
	return mintPubKey, bondingCurvePubKey, associatedBondingCurvePubKey, nil
}

func buyTokenAmountsFrom(solAmount, price, slippage float64) (uint64, uint64, float64, error) {
	if price == 0 {
		return 0, 0, 0, fmt.Errorf("price is 0")
	}

	tokenAmount := solAmount / price
	amountInLamports := uint64(tokenAmount * lamportsPerSol)
	maxAmountLamports := uint64(float64(amountInLamports) * (1 + slippage))
	return amountInLamports, maxAmountLamports, tokenAmount, nil
}

func sellDataFrom(amount uint64, minSolOutput uint64) []byte {
	discriminator := make([]byte, 8)
	binary.LittleEndian.PutUint64(discriminator, 12502976635542562355)

	amountData := make([]byte, 8)
	binary.LittleEndian.PutUint64(amountData, amount)

	minSolOutputData := make([]byte, 8)
	binary.LittleEndian.PutUint64(minSolOutputData, minSolOutput)

	return append(discriminator, append(amountData, minSolOutputData...)...)
}

func sellAccountsFrom(
	mintPubKey solana.PublicKey,
	bondingCurvePubKey solana.PublicKey,
	associatedBondingCurvePubKey solana.PublicKey,
	ataPubKey solana.PublicKey,
	payerPubKey solana.PublicKey,
) []*solana.AccountMeta {
	return []*solana.AccountMeta{
		solana.NewAccountMeta(PUMP_GLOBAL, false, false),                             // PUMP_GLOBAL
		solana.NewAccountMeta(PUMP_FEE, true, false),                                 // PUMP_FEE
		solana.NewAccountMeta(mintPubKey, false, false),                              // Mint
		solana.NewAccountMeta(bondingCurvePubKey, true, false),                       // Bonding Curve
		solana.NewAccountMeta(associatedBondingCurvePubKey, true, false),             // Associated Bonding Curve
		solana.NewAccountMeta(ataPubKey, true, false),                                // Associated Token Account
		solana.NewAccountMeta(payerPubKey, true, true),                               // Payer
		solana.NewAccountMeta(SYSTEM_PROGRAM, false, false),                          // SYSTEM_PROGRAM
		solana.NewAccountMeta(SYSTEM_ASSOCIATED_TOKEN_ACCOUNT_PROGRAM, false, false), // SYSTEM_ASSOCIATED_TOKEN_ACCOUNT_PROGRAM
		solana.NewAccountMeta(SYSTEM_TOKEN_PROGRAM, false, false),                    // SYSTEM_TOKEN_PROGRAM
		solana.NewAccountMeta(PUMP_EVENT_AUTHORITY, false, false),                    // PUMP_EVENT_AUTHORITY
		solana.NewAccountMeta(PUMP_PROGRAM, false, false),                            // PUMP_PROGRAM
	}
}
