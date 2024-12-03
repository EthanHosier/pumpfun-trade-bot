package pumpSnipeBot

import (
	"testing"
	"time"

	"github.com/ethanhosier/pumpfun-trade-bot/config"
	"github.com/joho/godotenv"
)

func init() {
	err := godotenv.Load("../.env")
	if err != nil {
		panic(err)
	}
}

func TestPumpSnipeBot(t *testing.T) {
	ticker := time.NewTicker(2 * time.Minute)

	config := config.MustNewDefaultConfig()
	bot := NewPumpSnipeBot(config.Notifier, config.BlockchainClient, config.CoinInfoClient, config.PumpFunClient)

	go bot.handleBuyAndSell("G791oHKLamcQmik9bxkW6M1XpFrJsavF1MfujjUdpump", nil)

	<-ticker.C
}
