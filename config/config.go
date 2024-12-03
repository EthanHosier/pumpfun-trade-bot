package config

import (
	"os"

	"github.com/ethanhosier/pumpfun-trade-bot/blockchain"
	"github.com/ethanhosier/pumpfun-trade-bot/coinInfo"
	"github.com/ethanhosier/pumpfun-trade-bot/notifications"
	"github.com/ethanhosier/pumpfun-trade-bot/pumpfun"
	"github.com/joho/godotenv"
)

type Config struct {
	HeliusApiKey     string
	BlockchainClient *blockchain.BlockchainClient
	CoinInfoClient   *coinInfo.CoinInfoClient
	Notifier         notifications.Notifier
}

func MustNewDefaultConfig() *Config {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	heliusApiKey := os.Getenv("HELIUS_API_KEY")
	coinInfoClient := coinInfo.NewCoinInfoClient(pumpfun.NewPumpFunClient(os.Getenv("PUMPFUN_API_KEY")))
	blockchainClient := blockchain.NewBlockchainClient(heliusApiKey, coinInfoClient)
	clicksendClient := notifications.NewClicksendClient(os.Getenv("CLICKSEND_USERNAME"), os.Getenv("CLICKSEND_API_KEY"))

	return &Config{
		HeliusApiKey:     heliusApiKey,
		BlockchainClient: blockchainClient,
		CoinInfoClient:   coinInfoClient,
		Notifier:         clicksendClient,
	}
}
