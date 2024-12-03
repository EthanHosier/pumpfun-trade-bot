package config

import (
	"os"

	"github.com/ethanhosier/pumpfun-trade-bot/blockchain"
	"github.com/ethanhosier/pumpfun-trade-bot/coinInfo"
	"github.com/ethanhosier/pumpfun-trade-bot/kingOfTheHill"
	"github.com/ethanhosier/pumpfun-trade-bot/notifications"
	"github.com/ethanhosier/pumpfun-trade-bot/pumpfun"
)

type Config struct {
	HeliusApiKey        string
	BlockchainClient    *blockchain.BlockchainClient
	CoinInfoClient      *coinInfo.CoinInfoClient
	Notifier            notifications.Notifier
	KingOfTheHillClient *kingOfTheHill.KingOfTheHillClient
	PumpFunClient       *pumpfun.PumpFunClient
}

func MustNewDefaultConfig() *Config {
	heliusApiKey := os.Getenv("HELIUS_API_KEY")
	if heliusApiKey == "" {
		panic("HELIUS_API_KEY is not set")
	}

	dataImpulseProxyUrl := os.Getenv("DATA_IMPULSE_PROXY_URL")
	if dataImpulseProxyUrl == "" {
		panic("DATA_IMPULSE_PROXY_URL is not set")
	}

	pumpfunClient := pumpfun.NewPumpFunClient(os.Getenv("PUMPFUN_API_KEY"), dataImpulseProxyUrl)
	kingOfTheHillClient := kingOfTheHill.NewKingOfTheHillClient(pumpfunClient)
	coinInfoClient := coinInfo.NewCoinInfoClient(pumpfunClient)
	blockchainClient := blockchain.NewBlockchainClient(heliusApiKey, coinInfoClient)
	clicksendClient := notifications.NewClicksendClient(os.Getenv("CLICKSEND_USERNAME"), os.Getenv("CLICKSEND_API_KEY"))

	return &Config{
		HeliusApiKey:        heliusApiKey,
		BlockchainClient:    blockchainClient,
		CoinInfoClient:      coinInfoClient,
		Notifier:            clicksendClient,
		KingOfTheHillClient: kingOfTheHillClient,
		PumpFunClient:       pumpfunClient,
	}
}
