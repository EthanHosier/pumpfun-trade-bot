package coinInfo

import "github.com/ethanhosier/pumpfun-trade-bot/pumpfun"

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
