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
	buyAmountSol = 0.01
	buySlippage  = 0.5

	ethanPhoneNumber = "+447476133726"
	maxHoldTime      = 1 * time.Minute
	kohPollTime      = 500 * time.Millisecond

	sellSlippage = 0.9
)

type PumpSnipeBot struct {
	notifier         notifications.Notifier
	blockchainClient *blockchain.BlockchainClient
	coinInfoClient   *coinInfo.CoinInfoClient

	seenCoins   map[string]bool
	seenCoinsMu sync.Mutex
}

func NewPumpSnipeBot(notifier notifications.Notifier, blockchainClient *blockchain.BlockchainClient, coinInfoClient *coinInfo.CoinInfoClient) *PumpSnipeBot {
	return &PumpSnipeBot{
		notifier:         notifier,
		blockchainClient: blockchainClient,
		coinInfoClient:   coinInfoClient,
		seenCoins:        make(map[string]bool),
		seenCoinsMu:      sync.Mutex{},
	}
}

func (p *PumpSnipeBot) Start(wallets []string) error {
	slog.Info("Starting pump snipe bot for wallets", "wallets", wallets)

	doneCh := make(chan interface{})
	wtsCh, wtsErrsCh, err := p.blockchainClient.SubscribeToWalletsTransactionSignatures(wallets, doneCh)
	if err != nil {
		return err
	}

	transactionErrsCh := make(chan error)

	// TODO: add max concurrent holds
	for {
		select {
		case err := <-wtsErrsCh:
			return err
		case err := <-transactionErrsCh:
			return err
		case wts := <-wtsCh:
			p.handleTransaction(&wts, transactionErrsCh)
		case <-doneCh:
			return nil
		}
	}
}

func (p *PumpSnipeBot) handleTransaction(tx *blockchain.WalletTransactionSignature, errsCh chan<- error) {
	transaction, err := p.blockchainClient.GetTransactionDataWithRetries(tx.Signature, 3)
	if err != nil {
		errsCh <- err
		return
	}

	if !isPumpfunBuy(transaction) {
		return
	}

	mint, err := pumpfunMint(transaction)
	if err != nil {
		errsCh <- err
		return
	}

	if p.seenCoins[mint] {
		return
	}

	p.seenCoins[mint] = true

	go p.handleBuyAndSell(mint, errsCh)
	return
}

func (p *PumpSnipeBot) handleBuyAndSell(mint string, errsCh chan<- error) {
	coinData, _, err := p.coinInfoClient.CoinDataFor(mint, false)
	if err != nil {
		errsCh <- err
		return
	}

	btr, err := p.blockchainClient.BuyTokenWithSol(mint, coinData.BondingCurve, coinData.AssociatedBondingCurve, buyAmountSol, buySlippage, os.Getenv("PRIVATE_KEY"))
	if err != nil {
		errsCh <- err
		return
	}

	go p.handleNotifyBuy(mint, btr.TokenAmount, coinData.Symbol)
	go p.handleHoldUntilSell(coinData, btr, errsCh)
}

func (p *PumpSnipeBot) handleHoldUntilSell(coinData *pumpfun.CoinData, btr *blockchain.BuyTokenResult, errsCh chan<- error) {
	kohCh := make(chan interface{})
	ticker := time.NewTicker(maxHoldTime)

	go func() {
		for {
			c, _, err := p.coinInfoClient.CoinDataFor(coinData.Mint, false)
			if err != nil {
				errsCh <- err
				return
			}
			if c.KingOfTheHillTimestamp > 0 {
				kohCh <- true
				return
			}
			time.Sleep(kohPollTime)
		}
	}()

	for {
		select {
		case <-ticker.C:
			go p.handleSell(coinData.Symbol, coinData, btr, errsCh, "max hold time reached")
		case <-kohCh:
			go p.handleSell(coinData.Symbol, coinData, btr, errsCh, "koh reached")
		}
	}
}

func (p *PumpSnipeBot) handleSell(symbol string, coinData *pumpfun.CoinData, btr *blockchain.BuyTokenResult, errsCh chan<- error, reason string) {
	txID, err := p.blockchainClient.SellToken(coinData.Mint, coinData.BondingCurve, coinData.AssociatedBondingCurve, btr.AssociatedTokenAccountAddress, sellSlippage, os.Getenv("PRIVATE_KEY"))
	if err != nil {
		errsCh <- err
		p.notifier.SendSMS(fmt.Sprintf("ERROR SELLING: %s failed: %v", pumpfunUrl(coinData.Mint), reason), ethanPhoneNumber)
		return
	}

	slog.Info("Sold token", "mint", coinData.Mint, "txId", txID)
	p.handleNotifySell(coinData.Mint, symbol)
}
