package botFinder

import (
	"os"
	"testing"

	"github.com/ethanhosier/pumpfun-trade-bot/coinInfo"
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
	botFinder := NewBotFinder(openaiClient, pumpFunClient, nil, nil, nil)

	trades, err := pumpFunClient.AllTradesForMint("4cab2KDe448uFKgz21FitpiDM7JWiPzYWdTLTuj7pump")
	if err != nil {
		t.Fatal(err)
	}

	userCodes, err := botFinder.findBotCandidatesInTradesWithChatgpt(trades)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(userCodes)
}

func TestBotFinder_FindBotCandidatesForMint(t *testing.T) {
	openaiClient := openai.NewOpenAiClient(os.Getenv("OPENAI_API_KEY"))
	pumpFunClient := pumpfun.NewPumpFunClient(os.Getenv("PUMPFUN_API_KEY"), "")
	coinInfoClient := coinInfo.NewCoinInfoClient(pumpFunClient)
	botFinder := NewBotFinder(openaiClient, pumpFunClient, coinInfoClient, nil, nil)

	userCodes, err := botFinder.findBotCandidatesForMint("HDf22yGrBpjYS2vKKhpREBEAUXs5CyYQDUB35FURCg8p")
	if err != nil {
		t.Fatal(err)
	}

	t.Log(userCodes)
}

// https://solscan.io/account/5P9YDk3fQ5ZXtMzrUyKoCEJL2q78s3Dbp9zoRWXj7bgp

// big daddy: https://solscan.io/account/JfPSxPNURkH6nYWjHdrT5krKjWeYW2STgaZDbK7rU9m - 169.06 sol
// prev big big daddy: https://solscan.io/account/GdA5DpW7xMFRZ7idPnAULcAp7CvyG79ezMkkqwiEQ5qf
