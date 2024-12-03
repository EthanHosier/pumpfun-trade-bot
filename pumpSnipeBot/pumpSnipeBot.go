package pumpSnipeBot

import (
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/ethanhosier/pumpfun-trade-bot/blockchain"
	"github.com/ethanhosier/pumpfun-trade-bot/coinInfo"
	"github.com/ethanhosier/pumpfun-trade-bot/notifications"
	"github.com/ethanhosier/pumpfun-trade-bot/pumpfun"
)

const (
	buyAmountSol = 0.001
	buySlippage  = 0.5

	ethanPhoneNumber = "+447476133726"
	maxHoldTime      = 20 * time.Second
	kohPollTime      = 500 * time.Millisecond

	sellSlippage       = 0.9
	proxyRepeats       = 2
	maxConcurrentHolds = 1
)

type PumpSnipeBot struct {
	notifier         notifications.Notifier
	blockchainClient *blockchain.BlockchainClient
	coinInfoClient   *coinInfo.CoinInfoClient
	pumpfunClient    *pumpfun.PumpFunClient

	seenCoins   map[string]bool
	seenCoinsMu sync.Mutex

	coinsHeld   int
	coinsHeldMu sync.Mutex
}

func NewPumpSnipeBot(notifier notifications.Notifier, blockchainClient *blockchain.BlockchainClient, coinInfoClient *coinInfo.CoinInfoClient, pumpfunClient *pumpfun.PumpFunClient) *PumpSnipeBot {
	return &PumpSnipeBot{
		notifier:         notifier,
		blockchainClient: blockchainClient,
		coinInfoClient:   coinInfoClient,
		pumpfunClient:    pumpfunClient,
		seenCoins:        make(map[string]bool),
		seenCoinsMu:      sync.Mutex{},
		coinsHeld:        0,
		coinsHeldMu:      sync.Mutex{},
	}
}

func (p *PumpSnipeBot) Start(wallets []string) error {
	slog.Info("Starting pump snipe bot for wallets", "wallets", wallets)

	doneCh := make(chan interface{})
	wtsCh, wtsErrsCh, err := p.blockchainClient.SubscribeToWalletsTransactionSignatures(wallets, doneCh)
	if err != nil {
		return err
	}

	transactionErrsCh := make(chan *BotError)

	for {
		// Check highest priority first - wallet transaction errors
		select {
		case err := <-wtsErrsCh:
			return err
		default:
		}

		// Check transaction errors next
		select {
		case err := <-transactionErrsCh:
			if err.forceQuit {
				p.notifier.SendSMS(fmt.Sprintf("Critical error: %.20s, entering 10 min standby mode", err.error.Error()), ethanPhoneNumber)
				time.Sleep(10 * time.Minute)
				return err.error
			} else {
				slog.Error("Non-critical error", "error", err.error)
			}
		default:
		}

		// Finally check for new transactions or done signal
		select {
		case wts := <-wtsCh:
			p.handleTransaction(&wts, transactionErrsCh)
		case <-doneCh:
			return nil
		default:
			// Add a small sleep to prevent tight loop
			time.Sleep(10 * time.Millisecond)
		}
	}

}

func (p *PumpSnipeBot) handleTransaction(tx *blockchain.WalletTransactionSignature, errsCh chan<- *BotError) {
	transaction, err := p.blockchainClient.GetTransactionDataWithRetries(tx.Signature, 3)
	if err != nil {
		errsCh <- &BotError{error: err, forceQuit: false}
		return
	}

	if !isPumpfunBuy(transaction) {
		return
	}

	mint, err := pumpfunMint(transaction)
	if err != nil {
		errsCh <- &BotError{error: err, forceQuit: false}
		return
	}

	if p.seenCoins[mint] {
		return
	}

	p.seenCoins[mint] = true

	p.coinsHeldMu.Lock()
	defer p.coinsHeldMu.Unlock()

	go p.tryExecuteTrade(mint, errsCh)
	return
}

func (p *PumpSnipeBot) tryExecuteTrade(mint string, errsCh chan<- *BotError) {
	p.coinsHeldMu.Lock()
	if p.coinsHeld >= maxConcurrentHolds {
		slog.Info("Max concurrent holds reached", "max", maxConcurrentHolds, "current", p.coinsHeld)
		p.coinsHeldMu.Unlock()
		return
	}
	slog.Info("Incrementing coins held", "current", p.coinsHeld)
	p.coinsHeld++
	p.coinsHeldMu.Unlock()

	p.handleBuyAndSell(mint, errsCh)

	p.coinsHeldMu.Lock()
	p.coinsHeld--
	p.coinsHeldMu.Unlock()
}

func (p *PumpSnipeBot) handleBuyAndSell(mint string, errsCh chan<- *BotError) {
	coinData, _, err := p.coinInfoClient.CoinDataFor(mint, false)
	if err != nil {
		errsCh <- &BotError{error: err, forceQuit: false}
		return
	}

	slog.Info("Buying token", "mint", mint, "symbol", coinData.Symbol)
	btr, err := p.blockchainClient.BuyTokenWithSol(mint, coinData.BondingCurve, coinData.AssociatedBondingCurve, buyAmountSol, buySlippage, os.Getenv("WALLET_PRIVATE_KEY"))
	if err != nil {
		errsCh <- &BotError{error: err, forceQuit: false}
		return
	}

	go p.handleNotifyBuy(mint, btr.TokenAmount, coinData.Symbol)
	go p.handleHoldUntilSell(coinData, btr, errsCh)
}

func (p *PumpSnipeBot) handleHoldUntilSell(coinData *pumpfun.CoinData, btr *blockchain.BuyTokenResult, errsCh chan<- *BotError) {
	kohCh := make(chan interface{})
	ticker := time.NewTicker(maxHoldTime)

	endTime := time.Now().Add(maxHoldTime)

	for i := 0; i < proxyRepeats; i++ {
		go func() {
			errCount := 0
			for {
				if time.Now().After(endTime) { // Exit if max hold time reached
					return
				}
				if errCount > 6 {
					errsCh <- &BotError{error: fmt.Errorf("failed to get coin data after %d retries", errCount), forceQuit: true}
					return
				}
				c, _, err := p.pumpfunClient.CoinDataFor(coinData.Mint, false, true)
				if err != nil {
					errCount++
					continue
				}
				errCount = 0
				slog.Info(c.Mint, "koh", c.KingOfTheHillTimestamp > 0)
				if c.KingOfTheHillTimestamp > 0 {
					kohCh <- true
					return
				}
				time.Sleep(kohPollTime)
			}
		}()
	}

	for {
		select {
		case <-ticker.C:
			go p.handleSell(coinData.Symbol, coinData, btr, errsCh, "max hold time reached")
		case <-kohCh:
			go p.handleSell(coinData.Symbol, coinData, btr, errsCh, "koh reached")
		}
	}
}

func (p *PumpSnipeBot) handleSell(symbol string, coinData *pumpfun.CoinData, btr *blockchain.BuyTokenResult, errsCh chan<- *BotError, reason string) {
	slog.Info("Selling token", "mint", coinData.Mint, "symbol", symbol, "reason", reason)
	txID, err := p.blockchainClient.SellToken(coinData.Mint, coinData.BondingCurve, coinData.AssociatedBondingCurve, btr.AssociatedTokenAccountAddress, sellSlippage, os.Getenv("WALLET_PRIVATE_KEY"))
	if err != nil {
		// retry once
		txID, err = p.blockchainClient.SellToken(coinData.Mint, coinData.BondingCurve, coinData.AssociatedBondingCurve, btr.AssociatedTokenAccountAddress, sellSlippage, os.Getenv("WALLET_PRIVATE_KEY"))
		if err != nil {
			errsCh <- &BotError{error: err, forceQuit: true}
			p.notifier.SendSMS(fmt.Sprintf("ERROR SELLING: %s failed: %v", pumpfunUrl(coinData.Mint), reason), ethanPhoneNumber)
			return
		}
	}

	slog.Info("Sold token", "mint", coinData.Mint, "txId", txID, "reason", reason)
	p.handleNotifySell(coinData.Mint, symbol, reason)
}
