package main

import (
	"fmt"
	"os"

	"github.com/ethanhosier/pumpfun-trade-bot/blockchain"
	"github.com/ethanhosier/pumpfun-trade-bot/coinInfo"
	"github.com/ethanhosier/pumpfun-trade-bot/pumpfun"
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

	coinInfoClient := coinInfo.NewCoinInfoClient(pumpfun.NewPumpFunClient(os.Getenv("PUMPFUN_API_KEY")))

	bc := blockchain.NewBlockchainClient(os.Getenv("HELIUS_API_KEY"), coinInfoClient)

	err := bc.BuyToken("2ownHic2xgfAkZX79HQY5QEMaDAEXDXq7BdwAPpQJSCr", "3yDrKYwVa5ezQUvBW8hFHW1TYEdXZ6QziYjze9FvWG67", "24ZiJfVrmAk9GQAtibTz2f2HoHgytWu83Lf5TY9FiUcz", 0.0001, 10, os.Getenv("WALLET_PRIVATE_KEY"))
	if err != nil {
		panic(err)
	}

	fmt.Println("Success")
}
