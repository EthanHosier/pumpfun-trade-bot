package main

import (
	"fmt"
	"os"

	"github.com/ethanhosier/pumpfun-trade-bot/blockchain"
	"github.com/ethanhosier/pumpfun-trade-bot/coinInfo"
	"github.com/ethanhosier/pumpfun-trade-bot/pumpfun"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		panic(err)
	}

	fmt.Println("Hello, World!")

	// willWallet := "BGfHXtqWiXP1goEu66eAeDnHQoLuohspdV5ui51qi56f"
	coinInfoClient := coinInfo.NewCoinInfoClient(pumpfun.NewPumpFunClient(os.Getenv("PUMPFUN_API_KEY")))

	coinMint := "9DcRGV5bDNFg6sxhR15vka4ZUduQbr9HbdchuSrkKXDn"
	coinData, _, err := coinInfoClient.CoinDataFor(coinMint, false)
	if err != nil {
		panic(err)
	}

	bc := blockchain.NewBlockchainClient(os.Getenv("HELIUS_API_KEY"), coinInfoClient)

	// buyTokenResult, err := bc.BuyTokenWithSol(coinMint, coinData.BondingCurve, coinData.AssociatedBondingCurve, 0.01, 10, os.Getenv("WALLET_PRIVATE_KEY"))
	// if err != nil {
	// 	panic(err)
	// }

	// fmt.Printf("Success. Result: %+v\n", *buyTokenResult)

	txID, err := bc.SellTokenWithSol(
		coinMint,
		coinData.BondingCurve,
		coinData.AssociatedBondingCurve,
		"FmtAkNNiaeov2X2cXrfhwTZMH3fGXfZvgthjvXig9kJY",
		0.3, // 10% slippage
		os.Getenv("WALLET_PRIVATE_KEY"),
	)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Success. Transaction ID: %s\n", txID)
}
