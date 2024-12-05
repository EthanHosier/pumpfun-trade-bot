package botFinder

import (
	"log"

	"github.com/ethanhosier/pumpfun-trade-bot/coinInfo"
	"github.com/ethanhosier/pumpfun-trade-bot/kingOfTheHill"
	"github.com/ethanhosier/pumpfun-trade-bot/openai"
	"github.com/ethanhosier/pumpfun-trade-bot/pumpfun"
	"github.com/ethanhosier/pumpfun-trade-bot/storage"
)

const (
	kohId              = "BotFinder"
	botSecondsToIgnore = 15
)

type BotFinder struct {
	openaiClient        *openai.OpenAiClient
	pumpFunClient       *pumpfun.PumpFunClient
	coinInfoClient      *coinInfo.CoinInfoClient
	storage             *storage.SupabaseStorage
	kingOfTheHillClient *kingOfTheHill.KingOfTheHillClient
}

func NewBotFinder(openaiClient *openai.OpenAiClient, pumpFunClient *pumpfun.PumpFunClient, coinInfoClient *coinInfo.CoinInfoClient, storage *storage.SupabaseStorage, kingOfTheHillClient *kingOfTheHill.KingOfTheHillClient) *BotFinder {
	return &BotFinder{openaiClient: openaiClient, pumpFunClient: pumpFunClient, coinInfoClient: coinInfoClient, storage: storage, kingOfTheHillClient: kingOfTheHillClient}
}

func (b *BotFinder) CoinTradeTrackerLoop() error {
	ch, err := b.kingOfTheHillClient.Subscribe("BotFinder")
	if err != nil {
		panic(err)
	}

	errChan := make(chan error)
	go func() {
		for coinData := range ch {
			go func() {
				err := b.handleNewKohCoin(coinData)
				if err != nil {
					errChan <- err
				}
			}()
		}
	}()

	return <-errChan
}

func (b *BotFinder) handleNewKohCoin(coinData *pumpfun.CoinData) error {
	if coin, _ := b.storage.Get(storage.DbCoinsTable, coinData.Mint); coin != nil {
		return nil
	}

	trades, err := b.pumpFunClient.AllTradesForMint(coinData.Mint)
	if err != nil {
		panic(err)
	}

	filteredTrades := filterTrades(trades, coinData)

	_, err = b.storage.Store(storage.DbCoinsTable, coinData)
	if err != nil {
		panic(err)
	}

	interfaceSlice := make([]interface{}, len(filteredTrades))
	for i, v := range filteredTrades {
		interfaceSlice[i] = v.ToStorableTrade()
	}
	_, err = b.storage.StoreAll(storage.DbTradesTable, interfaceSlice)
	if err != nil {
		return err
	}

	log.Printf("Stored coin data and %d trades for %s %s", len(filteredTrades), coinData.Symbol, coinData.Mint)
	return nil
}

func filterTrades(trades []pumpfun.Trade, coinData *pumpfun.CoinData) []pumpfun.Trade {
	// Create a map to track user trading patterns
	userTrades := make(map[string][]pumpfun.Trade)

	// Get the timestamp of the first trade (last index since trades are in reverse chronological order)
	firstTradeTime := trades[len(trades)-1].Timestamp

	// Group trades by user
	for _, trade := range trades {
		// ignore coins which come after the Koh timestamp
		if trade.Timestamp > coinData.KingOfTheHillTimestamp {
			continue
		}

		userTrades[trade.User] = append(userTrades[trade.User], trade)
	}

	// Filter out sniper bot trades
	var filteredTrades []pumpfun.Trade
	for _, trade := range trades {
		userTradeList := userTrades[trade.User]

		// Check if user is a sniper bot:
		// 1. Has exactly 2 trades
		// 2. First trade (index 1) is buy, second trade (index 0) is sell
		// 3. Both trades within 30 seconds of first trade
		isSniper := len(userTradeList) == 2 &&
			userTradeList[1].IsBuy && !userTradeList[0].IsBuy &&
			userTradeList[0].Timestamp-firstTradeTime <= botSecondsToIgnore &&
			userTradeList[1].Timestamp-firstTradeTime <= botSecondsToIgnore

		if !isSniper {
			filteredTrades = append(filteredTrades, trade)
		}
	}

	return filteredTrades
}
