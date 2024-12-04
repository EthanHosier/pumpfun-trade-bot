package botFinder

import (
	"os"
	"testing"

	"github.com/ethanhosier/pumpfun-trade-bot/openai"
	"github.com/ethanhosier/pumpfun-trade-bot/pumpfun"
	"github.com/joho/godotenv"
)

func init() {
	err := godotenv.Load("../.env")
	if err != nil {
		panic(err)
	}
}

func TestBotFinder_FindBotsInTradesWithChatgpt(t *testing.T) {
	openaiClient := openai.NewOpenAiClient(os.Getenv("OPENAI_API_KEY"))
	pumpFunClient := pumpfun.NewPumpFunClient(os.Getenv("PUMPFUN_API_KEY"), "")
	botFinder := NewBotFinder(openaiClient, pumpFunClient)

	trades, err := pumpFunClient.AllTradesForMint("4cab2KDe448uFKgz21FitpiDM7JWiPzYWdTLTuj7pump")
	if err != nil {
		t.Fatal(err)
	}

	userCodes, err := botFinder.findBotsInTradesWithChatgpt(trades)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(userCodes)
}
