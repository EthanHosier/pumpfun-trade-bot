package main

import (
	"github.com/ethanhosier/pumpfun-trade-bot/config"
	"github.com/ethanhosier/pumpfun-trade-bot/pumpSnipeBot"
)

func main() {
	config := config.MustNewDefaultConfig()
	bot := pumpSnipeBot.NewPumpSnipeBot(config.Notifier, config.BlockchainClient, config.CoinInfoClient)

	wallets := []string{"BGfHXtqWiXP1goEu66eAeDnHQoLuohspdV5ui51qi56f"}
	panic(bot.Start(wallets))
}
