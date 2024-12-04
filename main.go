package main

import (
	"github.com/ethanhosier/pumpfun-trade-bot/config"
	"github.com/ethanhosier/pumpfun-trade-bot/pumpSnipeBot"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	config := config.MustNewDefaultConfig()
	bot := pumpSnipeBot.NewPumpSnipeBot(config.Notifier, config.BlockchainClient, config.CoinInfoClient, config.PumpFunClient)

	wallets := []string{"8UEQEW1XmyyEoP1S9MfJDPyVCWViZLq2A71BQXf8yGqx"}
	panic(bot.Start(wallets))

}
