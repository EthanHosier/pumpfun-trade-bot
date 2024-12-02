package coinInfo

import (
	"fmt"

	"github.com/ethanhosier/pumpfun-trade-bot/pumpfun"
)

type CoinInfoClient struct {
	pumpfunClient *pumpfun.PumpFunClient
}

func NewCoinInfoClient(pumpfunClient *pumpfun.PumpFunClient) *CoinInfoClient {
	return &CoinInfoClient{pumpfunClient}
}

func (c *CoinInfoClient) SolPrice() (float64, error) {
	return c.pumpfunClient.SolPrice()
}

func (c *CoinInfoClient) CoinDataFor(mint string, getHolders bool) (*pumpfun.CoinData, []pumpfun.CoinHolder, error) {
	return c.pumpfunClient.CoinDataFor(mint, getHolders)
}

func (c *CoinInfoClient) PriceInSolFromBondingCurveAddress(bondingCurveAddress string) (float64, error) {
	data, err := fetchCurveData(bondingCurveAddress)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch curve data: %w", err)
	}
	return calculatePrice(data)
}
