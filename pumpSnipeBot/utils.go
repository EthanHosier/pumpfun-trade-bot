package pumpSnipeBot

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/ethanhosier/pumpfun-trade-bot/blockchain"
)

func (p *PumpSnipeBot) handleNotifyBuy(mint string, tokenAmount float64, symbol string) {
	solPrice, err := p.coinInfoClient.SolPrice()
	if err != nil {
		slog.Error("Error getting SOL price", "error", err)
		return
	}

	amountInPounds := solPrice * buyAmountSol

	err = p.notifier.SendSMS(fmt.Sprintf("BUY: %s £%v -> %v %s", pumpfunUrl(mint), amountInPounds, tokenAmount, symbol), ethanPhoneNumber)
	if err != nil {
		slog.Error("Error sending SMS", "error", err)
	}
}

func (p *PumpSnipeBot) handleNotifySell(mint string, symbol string) {
	err := p.notifier.SendSMS(fmt.Sprintf("SELL: %s -> %s", pumpfunUrl(mint), symbol), ethanPhoneNumber)
	if err != nil {
		slog.Error("Error sending SMS", "error", err)
	}
}

func isPumpfunBuy(tx *blockchain.Transaction) bool {
	buy := false
	pumpfun := false

	for _, log := range tx.Meta.LogMessages {
		if strings.Contains(log, "Instruction: Buy") {
			buy = true
		}
		if strings.Contains(log, "Program 6EF8rrecthR5Dkzon8Nwu78hRvfCKubJ14M5uBEwF6P invoke") {
			pumpfun = true
		}

		if buy && pumpfun {
			return true
		}
	}

	return false
}

func pumpfunMint(tx *blockchain.Transaction) (string, error) {
	if len(tx.Meta.PostTokenBalances) == 0 {
		return "", fmt.Errorf("no post token balances")
	}
	return tx.Meta.PostTokenBalances[0].Mint, nil
}

func pumpfunUrl(mint string) string {
	return fmt.Sprintf("https://pump.fun/coin/%s", mint)
}
