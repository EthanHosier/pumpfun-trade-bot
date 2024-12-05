package main

import (
	"flag"
	"time"

	"github.com/ethanhosier/pumpfun-trade-bot/config"
	"github.com/ethanhosier/pumpfun-trade-bot/pumpSnipeBot"
	"github.com/joho/godotenv"
)

func main() {
	// Add botFinder flag
	botFinderEnabled := flag.Bool("botFinder", false, "Enable bot finder functionality")
	flag.Parse()

	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	config := config.MustNewDefaultConfig()

	// Only run bot finder if flag is set
	if *botFinderEnabled {
		go func() { config.KingOfTheHillClient.Start(5*time.Second, 999999) }()
		panic(config.BotFinder.CoinTradeTrackerLoop())
	}

	// Buy Bot
	pumpSnipeBot := pumpSnipeBot.NewPumpSnipeBot(config.Notifier, config.BlockchainClient, config.CoinInfoClient, config.PumpFunClient)
	wallets := []string{"J4bzyKJKZKKz2HUGFiq3DMRaxEaw6MxKf8rjGTvpkqaU"}
	panic(pumpSnipeBot.Start(wallets))
}
