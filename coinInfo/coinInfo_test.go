package coinInfo

import (
	"os"
	"testing"

	"github.com/ethanhosier/pumpfun-trade-bot/pumpfun"
	"github.com/joho/godotenv"
)

func init() {
	if err := godotenv.Load("../.env"); err != nil {
		println("No .env file found")
	}
}

func TestSolPrice(t *testing.T) {
	client := NewCoinInfoClient(pumpfun.NewPumpFunClient(os.Getenv("PUMPFUN_API_KEY")))
	price, err := client.SolPrice()
	if err != nil {
		t.Errorf("Error getting SOL price: %v", err)
	}

	t.Logf("SOL price: %f", price)
}

func TestCoinDataWithHoldersFor(t *testing.T) {
	client := NewCoinInfoClient(pumpfun.NewPumpFunClient(os.Getenv("PUMPFUN_API_KEY")))
	coinData, holders, err := client.CoinDataFor("Df6yfrKC8kZE3KNkrHERKzAetSxbrWeniQfyJY4Jpump", true)
	if err != nil {
		t.Errorf("Error getting coin data: %v", err)
	}

	t.Logf("Coin data: %+v", coinData)
	t.Logf("Holders: %+v", holders)
}

func TestCoinDataWithoutHoldersFor(t *testing.T) {
	client := NewCoinInfoClient(pumpfun.NewPumpFunClient(os.Getenv("PUMPFUN_API_KEY")))
	coinData, holders, err := client.CoinDataFor("CBW2QFfKP8VYLn2UfeU6MWTTKQ8n82H47aqKWdKTpump", false)
	if err != nil {
		t.Errorf("Error getting coin data: %v", err)
	}

	if holders != nil {
		t.Error("Expected holders to be nil")
	}

	t.Logf("Coin data: %+v", coinData)
}

func TestPriceInSolFromBondingCurveAddress(t *testing.T) {
	client := NewCoinInfoClient(pumpfun.NewPumpFunClient(os.Getenv("PUMPFUN_API_KEY")))
	price, err := client.PriceInSolFromBondingCurveAddress("3yDrKYwVa5ezQUvBW8hFHW1TYEdXZ6QziYjze9FvWG67")
	if err != nil {
		t.Errorf("Error getting price in SOL from bonding curve address: %v", err)
	}

	t.Logf("Price in SOL: %f", price)
}
