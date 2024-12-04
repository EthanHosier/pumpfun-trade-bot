package botFinder

import (
	"github.com/ethanhosier/pumpfun-trade-bot/openai"
	"github.com/ethanhosier/pumpfun-trade-bot/pumpfun"
)

type BotFinder struct {
	openaiClient  *openai.OpenAiClient
	pumpFunClient *pumpfun.PumpFunClient
}

func NewBotFinder(openaiClient *openai.OpenAiClient, pumpFunClient *pumpfun.PumpFunClient) *BotFinder {
	return &BotFinder{openaiClient: openaiClient, pumpFunClient: pumpFunClient}
}
