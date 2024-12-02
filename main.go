package main

import (
	"fmt"
	"os"

	"github.com/ethanhosier/pumpfun-trade-bot/blockchain"
	"github.com/joho/godotenv"
)

// func main() {
// 	if err := godotenv.Load(); err != nil {
// 		panic(err)
// 	}

// 	bc := blockchain.NewBlockchainClient(os.Getenv("HELIUS_API_KEY"))

// 	doneCh := make(chan interface{})
// 	walletTransactionSignaturesCh, errCh, err := bc.SubscribeToWalletsTransactionSignatures([]string{"orcACRJYTFjTeo2pV8TfYRTpmqfoYgbVi9GeANXTCc8", "12BRrNxzJYMx7cRhuBdhA71AchuxWRcvGydNnDoZpump"}, doneCh)
// 	if err != nil {
// 		panic(err)
// 	}

// 	for {
// 		select {
// 		case err := <-errCh:
// 			panic(err)
// 		case walletTransactionSignature := <-walletTransactionSignaturesCh:
// 			fmt.Printf("%+v\n", walletTransactionSignature)
// 		}
// 	}
// }

func main() {
	if err := godotenv.Load(); err != nil {
		panic(err)
	}

	fmt.Println("Hello, World!")

	// willWallet := "BGfHXtqWiXP1goEu66eAeDnHQoLuohspdV5ui51qi56f"

	bc := blockchain.NewBlockchainClient(os.Getenv("HELIUS_API_KEY"))

	address, sig, err := bc.GetOrCreateTokenAccount("8u4QzAEwvxY1PZQF5EmZu9NtsMufinqG45SVnWivpump", os.Getenv("WALLET_PRIVATE_KEY"))
	if err != nil {
		panic(err)
	}

	if sig != "" {
		transaction, err := bc.GetTransactionDataWithRetries(sig, 6)
		if err != nil {
			panic(err)
		}

		fmt.Printf("%+v\n", transaction)
	}

	fmt.Println(address)
}
