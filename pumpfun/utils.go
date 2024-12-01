package pumpfun

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"
)

func holderDataResponseFor(mint string, apiKey string) (*http.Response, error) {
	var (
		url     = fmt.Sprintf("https://pump-fe.helius-rpc.com/?api-key=%s", apiKey)
		headers = map[string]string{
			"Accept":             "*/*",
			"Accept-Encoding":    "gzip, deflate, br, zstd",
			"Accept-Language":    "en-GB,en;q=0.9,en;q=0.8",
			"Content-Type":       "application/json",
			"Origin":             "https://pump.fun",
			"Referer":            "https://pump.fun/",
			"Sec-CH-UA":          `"Not_A Brand";v="8", "Chromium";v="120", "Google Chrome";v="120"`,
			"Sec-CH-UA-Mobile":   "?0",
			"Sec-CH-UA-Platform": `"Linux"`,
			"Sec-Fetch-Dest":     "empty",
			"Sec-Fetch-Mode":     "cors",
			"Sec-Fetch-Site":     "cross-site",
			"Solana-Client":      "js/1.0.0-maintenance",
			"User-Agent":         "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		}
		body = map[string]interface{}{
			"id":      "58e2a1a2-adce-4a4e-a3cd-2692beb6b86c",
			"jsonrpc": "2.0",
			"method":  "getTokenLargestAccounts",
			"params": []interface{}{
				mint,
				map[string]string{"commitment": "confirmed"},
			},
		}
	)

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("Error creating HTTP request: %v", err)
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{}
	return client.Do(req)
}

func accountsFor(mint string, apiKey string) ([]Account, error) {
	type Context struct {
		APIVersion string `json:"apiVersion"`
		Slot       int    `json:"slot"`
	}

	type Result struct {
		Context Context   `json:"context"`
		Value   []Account `json:"value"`
	}

	type Response struct {
		JSONRPC string `json:"jsonrpc"`
		Result  Result `json:"result"`
		ID      string `json:"id"`
	}

	resp, err := holderDataResponseFor(mint, apiKey)
	if err != nil {
		return nil, fmt.Errorf("Error sending request: %v", err)
	}

	defer resp.Body.Close()
	var reader io.Reader
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err = gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("Error creating gzip reader: %v", err)
		}
		defer reader.(*gzip.Reader).Close()
	default:
		reader = resp.Body
	}

	bodyBytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("Error reading response body: %v", err)
	}

	var response Response
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		return nil, fmt.Errorf("Error unmarshaling JSON response: %v", err)
	}

	return response.Result.Value, nil
}

func holdersFrom(accounts []Account, associatedBondingCurve string, totalSupply int64) ([]CoinHolder, error) {

	holders := make([]CoinHolder, len(accounts))

	for i, account := range accounts {
		a := account.UIAmount * float64(math.Pow(10, float64(account.Decimals)))

		holders[i] = CoinHolder{
			Address:        account.Address,
			Amount:         a,
			PercentageHeld: 100 * a / float64(totalSupply),
			IsBondingCurve: account.Address == associatedBondingCurve,
		}
	}

	return holders, nil
}

//////////////////////////////////

// Pump Curve Constants
const (
	TokenDecimals           = 6
	LamportsPerSol          = 1_000_000_000
	VirtualTokenReservesPos = 0x08
	VirtualSolReservesPos   = 0x10
)

// Structs for JSON Parsing
type RpcResponse struct {
	Jsonrpc string `json:"jsonrpc"`
	Result  struct {
		Value struct {
			Data []string `json:"data"`
		} `json:"value"`
	} `json:"result"`
}

// Decode Base64 data into byte array
func decodeBase64(data string) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %v", err)
	}
	return decoded, nil
}

// Read uint64 value from byte array
func readUint64LE(data []byte, offset int) (uint64, error) {
	if len(data) < offset+8 {
		return 0, errors.New("buffer too small to read uint64")
	}
	return binary.LittleEndian.Uint64(data[offset : offset+8]), nil
}

// Calculate token price in SOL
func calculatePrice(data []byte) (float64, error) {
	virtualTokenReserves, err := readUint64LE(data, VirtualTokenReservesPos)
	if err != nil {
		return 0, fmt.Errorf("failed to read virtual token reserves: %v", err)
	}

	virtualSolReserves, err := readUint64LE(data, VirtualSolReservesPos)
	if err != nil {
		return 0, fmt.Errorf("failed to read virtual SOL reserves: %v", err)
	}

	if virtualTokenReserves == 0 || virtualSolReserves == 0 {
		return 0, errors.New("invalid reserves in curve state")
	}

	// Calculate price as (VirtualSolReserves / LamportsPerSol) / (VirtualTokenReserves / 10^TokenDecimals)
	price := (float64(virtualSolReserves) / LamportsPerSol) / (float64(virtualTokenReserves) / math.Pow10(TokenDecimals))
	return price, nil
}

// Fetch data from the API
func fetchCurveData(bondingCurveAddress string) ([]byte, error) {
	apiURL := "https://pump-fe.helius-rpc.com/?api-key=1b8db865-a5a1-4535-9aec-01061440523b"
	apiURL = fmt.Sprintf("%s&nocache=%d", apiURL, time.Now().UnixNano()) // Add a random query parameter to prevent caching

	payload := fmt.Sprintf(`{
		"id": "218b5eb0-7e47-4042-b7c0-b8eaab6d9edb",
		"jsonrpc": "2.0",
		"method": "getAccountInfo",
		"params": ["%s", {"encoding": "base64", "commitment": "confirmed"}]
	}`, bondingCurveAddress)

	client := &http.Client{}
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer([]byte(payload)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")
	req.Header.Set("Accept-Language", "en-GB,en-US;q=0.9,en;q=0.8")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://pump.fun")
	req.Header.Set("Referer", "https://pump.fun/")
	req.Header.Set("Sec-Ch-Ua", `"Not_A Brand";v="8", "Chromium";v="120", "Google Chrome";v="120"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", `"Linux"`)
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data: %v", err)
	}
	defer resp.Body.Close()

	var body []byte
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to create gzip reader: %v", err)
		}
		defer gzipReader.Close()
		body, err = io.ReadAll(gzipReader)
		if err != nil {
			return nil, fmt.Errorf("failed to read gzip response: %v", err)
		}
	} else {
		body, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response: %v", err)
		}
	}

	var rpcResponse RpcResponse
	err = json.Unmarshal(body, &rpcResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	if len(rpcResponse.Result.Value.Data) < 1 {
		return nil, errors.New("no data in response")
	}

	return decodeBase64(rpcResponse.Result.Value.Data[0])
}
