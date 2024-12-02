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

	willWallet := "BGfHXtqWiXP1goEu66eAeDnHQoLuohspdV5ui51qi56f"

	bc := blockchain.NewBlockchainClient(os.Getenv("HELIUS_API_KEY"))

	tx, err := bc.SendSolana(0.01, os.Getenv("WALLET_PRIVATE_KEY"), willWallet)
	if err != nil {
		panic(err)
	}

	fmt.Println(tx)
}
