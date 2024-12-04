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
	kohId             = "BotFinder"
	firstSecsToIgnore = 5
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

	// Filter trades in the first 5 seconds are ignored (to avoid sniper bots)
	cutoffTime := trades[len(trades)-1].Timestamp + firstSecsToIgnore
	filteredTrades := make([]pumpfun.Trade, 0)
	for _, trade := range trades {
		if trade.Timestamp > cutoffTime {
			filteredTrades = append(filteredTrades, trade)
		}
	}

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
