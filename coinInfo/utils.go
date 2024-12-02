package coinInfo

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

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
		return nil, fmt.Errorf("no data in response")
	}

	return decodeBase64(rpcResponse.Result.Value.Data[0])
}

func decodeBase64(data string) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %v", err)
	}
	return decoded, nil
}

func calculatePrice(data []byte) (float64, error) {
	virtualTokenReserves, err := readUint64LE(data, VirtualTokenReservesPos)
	if err != nil {
		return 0, fmt.Errorf("failed to read virtual token reserves: %v", err)
	}

	virtualSolReserves, err := readUint64LE(data, VirtualSolReservesPos)
	if err != nil {
		return 0, fmt.Errorf("failed to read virtual SOL reserves: %v", err)
	}
	log.Printf("Virtual sol reserves: %d", virtualSolReserves)
	log.Printf("Virtual token reserves: %d", virtualTokenReserves)

	if virtualTokenReserves == 0 || virtualSolReserves == 0 {
		return 0, fmt.Errorf("invalid reserves in curve state")
	}

	// Calculate price as (VirtualSolReserves / LamportsPerSol) / (VirtualTokenReserves / 10^TokenDecimals)
	price := float64(virtualSolReserves) / float64(virtualTokenReserves)
	return price, nil
}

// Read uint64 value from byte array
func readUint64LE(data []byte, offset int) (uint64, error) {
	if len(data) < offset+8 {
		return 0, fmt.Errorf("buffer too small to read uint64")
	}
	return binary.LittleEndian.Uint64(data[offset : offset+8]), nil
}
