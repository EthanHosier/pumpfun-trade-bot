package config

import (
	"os"

	"github.com/ethanhosier/pumpfun-trade-bot/blockchain"
	"github.com/ethanhosier/pumpfun-trade-bot/botFinder"
	"github.com/ethanhosier/pumpfun-trade-bot/coinInfo"
	"github.com/ethanhosier/pumpfun-trade-bot/kingOfTheHill"
	"github.com/ethanhosier/pumpfun-trade-bot/notifications"
	"github.com/ethanhosier/pumpfun-trade-bot/openai"
	"github.com/ethanhosier/pumpfun-trade-bot/pumpfun"
	"github.com/ethanhosier/pumpfun-trade-bot/storage"
	"github.com/ethanhosier/pumpfun-trade-bot/utils"
)

type Config struct {
	HeliusApiKey        string
	BlockchainClient    *blockchain.BlockchainClient
	CoinInfoClient      *coinInfo.CoinInfoClient
	Notifier            notifications.Notifier
	KingOfTheHillClient *kingOfTheHill.KingOfTheHillClient
	PumpFunClient       *pumpfun.PumpFunClient
	Storage             storage.Storage
	BotFinder           *botFinder.BotFinder
}

func MustNewDefaultConfig() *Config {
	heliusApiKey := utils.Required(os.Getenv("HELIUS_API_KEY"), "HELIUS_API_KEY")

	storage := storage.NewSupabaseStorage(utils.Required(os.Getenv("SUPABASE_URL"), "SUPABASE_URL"), utils.Required(os.Getenv("SUPABASE_SERVICE_KEY"), "SUPABASE_SERVICE_KEY"))
	pumpfunClient := pumpfun.NewPumpFunClient(utils.Required(os.Getenv("PUMPFUN_API_KEY"), "PUMPFUN_API_KEY"), utils.Required(os.Getenv("DATA_IMPULSE_PROXY_URL"), "DATA_IMPULSE_PROXY_URL"))
	kingOfTheHillClient := kingOfTheHill.NewKingOfTheHillClient(pumpfunClient)
	coinInfoClient := coinInfo.NewCoinInfoClient(pumpfunClient)
	blockchainClient := blockchain.NewBlockchainClient(heliusApiKey, coinInfoClient)
	clicksendClient := notifications.NewClicksendClient(utils.Required(os.Getenv("CLICKSEND_USERNAME"), "CLICKSEND_USERNAME"), utils.Required(os.Getenv("CLICKSEND_API_KEY"), "CLICKSEND_API_KEY"))
	openaiClient := openai.NewOpenAiClient(utils.Required(os.Getenv("OPENAI_API_KEY"), "OPENAI_API_KEY"))
	botFinder := botFinder.NewBotFinder(openaiClient, pumpfunClient, coinInfoClient, storage, kingOfTheHillClient)

	return &Config{
		HeliusApiKey:        heliusApiKey,
		BlockchainClient:    blockchainClient,
		CoinInfoClient:      coinInfoClient,
		Notifier:            clicksendClient,
		KingOfTheHillClient: kingOfTheHillClient,
		PumpFunClient:       pumpfunClient,
		Storage:             storage,
		BotFinder:           botFinder,
	}
}
