package pumpfun

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ethanhosier/pumpfun-trade-bot/utils"
)

type PumpFunClient struct {
	apiKey string
}

func NewPumpFunClient(apiKey string) *PumpFunClient {
	return &PumpFunClient{apiKey}
}

func (p *PumpFunClient) SolPrice() (float64, error) {
	url := "https://frontend-api.pump.fun/sol-price"

	type Response struct {
		Price float64 `json:"solPrice"`
	}

	resp, err := http.Get(url)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch SOL price: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result Response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Price, nil
}

func (p *PumpFunClient) CoinDataFor(mint string, getHolders bool) (*CoinData, []CoinHolder, error) {
	accountsTask := utils.DoAsync(func() ([]Account, error) {
		if getHolders {
			return accountsFor(mint, p.apiKey)
		}
		return nil, nil
	})

	url := fmt.Sprintf("https://frontend-api.pump.fun/coins/%s", mint)
	resp, err := http.Get(url)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch token info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var coinDataResponse CoinData
	if err := json.NewDecoder(resp.Body).Decode(&coinDataResponse); err != nil {
		fmt.Println(resp.Body) // print the string version of this:
		return nil, nil, fmt.Errorf("failed to decode response: %w", err)
	}
	accounts, err := utils.GetAsync(accountsTask)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get accounts: %w", err)
	}

	holders, err := holdersFrom(accounts, coinDataResponse.AssociatedBondingCurve, coinDataResponse.TotalSupply)

	return &coinDataResponse, holders, err
}
