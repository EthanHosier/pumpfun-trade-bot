package botFinder

import (
	"github.com/ethanhosier/pumpfun-trade-bot/coinInfo"
	"github.com/ethanhosier/pumpfun-trade-bot/openai"
	"github.com/ethanhosier/pumpfun-trade-bot/pumpfun"
)

type BotFinder struct {
	openaiClient   *openai.OpenAiClient
	pumpFunClient  *pumpfun.PumpFunClient
	coinInfoClient *coinInfo.CoinInfoClient
}

func NewBotFinder(openaiClient *openai.OpenAiClient, pumpFunClient *pumpfun.PumpFunClient, coinInfoClient *coinInfo.CoinInfoClient) *BotFinder {
	return &BotFinder{openaiClient: openaiClient, pumpFunClient: pumpFunClient, coinInfoClient: coinInfoClient}
}
