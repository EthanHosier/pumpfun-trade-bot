package pumpfun

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/ethanhosier/pumpfun-trade-bot/utils"
)

const (
	maxTradePageSize = 200
)

type PumpFunClient struct {
	apiKey   string
	proxyUrl string
}

func NewPumpFunClient(apiKey string, proxyUrl string) *PumpFunClient {
	return &PumpFunClient{apiKey, proxyUrl}
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

func (p *PumpFunClient) CoinDataFor(mint string, getHolders bool, useProxy bool) (*CoinData, []CoinHolder, error) {
	accountsTask := utils.DoAsync(func() ([]Account, error) {
		if getHolders {
			return accountsFor(mint, p.apiKey)
		}
		return nil, nil
	})

	var client *http.Client
	if useProxy {
		proxy, err := url.Parse(p.proxyUrl)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse proxy URL: %w", err)
		}
		client = &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(proxy),
			},
		}
	} else {
		client = &http.Client{}
	}

	url := fmt.Sprintf("https://frontend-api.pump.fun/coins/%s", mint)
	resp, err := client.Get(url)
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

func (p *PumpFunClient) KingOfTheHillCoinData() (*CoinData, error) {
	// Add a timestamp to the URL to prevent caching
	proxy, err := url.Parse(p.proxyUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to parse proxy URL: %w", err)
	}

	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxy),
		},
	}

	url := fmt.Sprintf("https://frontend-api.pump.fun/coins/king-of-the-hill?includeNsfw=true&_=%d", time.Now().UnixNano())

	// Create a new request with headers
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add common browser headers
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json,text/plain,*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Referer", "https://pump.fun/")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch king of the hill coin data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var coinDataResponse CoinData
	if err := json.NewDecoder(resp.Body).Decode(&coinDataResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &coinDataResponse, nil
}

func (p *PumpFunClient) numberOfTradesForMint(mint string) (int, error) {
	url := fmt.Sprintf("https://frontend-api-v2.pump.fun/trades/count/%s?minimumSize=50000000", mint)

	resp, err := http.Get(url)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch trade count: %w", err)
	}
	defer resp.Body.Close()

	var count int
	if err := json.NewDecoder(resp.Body).Decode(&count); err != nil {
		return 0, fmt.Errorf("failed to decode response: %w", err)
	}

	return count, nil
}

// seems to return too many right now??
func (p *PumpFunClient) AllTradesForMint(mint string) ([]Trade, error) {
	numOfTrades, err := p.numberOfTradesForMint(mint)
	if err != nil {
		return nil, fmt.Errorf("failed to get number of trades: %w", err)
	}

	numPages := (numOfTrades + maxTradePageSize - 1) / maxTradePageSize

	// Create a slice with sequential numbers from 0 to numPages-1
	pages := make([]int, numPages)
	for i := range pages {
		pages[i] = i
	}

	tradesTasks := utils.DoAsyncList(pages, func(page int) ([]Trade, error) {
		return p.tradesForMint(mint, page)
	})

	tradesArrs, err := utils.GetAsyncList(tradesTasks)
	if err != nil {
		return nil, fmt.Errorf("failed to get trades: %w", err)
	}

	flattenedTrades := make([]Trade, 0)
	for _, trades := range tradesArrs {
		flattenedTrades = append(flattenedTrades, trades...)
	}

	return flattenedTrades, nil
}

func (p *PumpFunClient) tradesForMint(mint string, page int) ([]Trade, error) {
	fmt.Println(page)

	url := fmt.Sprintf("https://frontend-api-v2.pump.fun/trades/all/%s?limit=%d&offset=%d&minimumSize=0", mint, maxTradePageSize, page*maxTradePageSize)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch trades: %w", err)
	}

	defer resp.Body.Close()

	var trades []Trade
	if err := json.NewDecoder(resp.Body).Decode(&trades); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return trades, nil
}
