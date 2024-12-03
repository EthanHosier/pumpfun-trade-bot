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

	wallets := []string{"EJNqoDXGXoUW7xvs9J9wvSFjxe8JjCM2WVoQeqcAAkD8"}
	panic(bot.Start(wallets))

}
