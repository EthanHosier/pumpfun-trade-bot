package pumpfun

import (
	"os"
	"testing"

	"github.com/joho/godotenv"
)

func init() {
	if err := godotenv.Load("../.env"); err != nil {
		println("No .env file found")
	}
}

func TestSolPrice(t *testing.T) {
	client := NewPumpFunClient(os.Getenv("PUMPFUN_API_KEY"), os.Getenv("DATA_IMPULSE_PROXY_URL"))
	price, err := client.SolPrice()
	if err != nil {
		t.Errorf("Error getting SOL price: %v", err)
	}

	t.Logf("SOL price: %f", price)
}

func TestCoinDataWithHoldersFor(t *testing.T) {
	client := NewPumpFunClient(os.Getenv("PUMPFUN_API_KEY"), os.Getenv("DATA_IMPULSE_PROXY_URL"))
	coinData, holders, err := client.CoinDataFor("Df6yfrKC8kZE3KNkrHERKzAetSxbrWeniQfyJY4Jpump", true, false)
	if err != nil {
		t.Errorf("Error getting coin data: %v", err)
	}

	if coinData == nil {
		t.Error("Expected coinData to not be nil")
	}

	if holders == nil {
		t.Error("Expected holders to not be nil")
	}

	t.Logf("Coin data: %+v", coinData)
	t.Logf("Holders: %+v", holders)
}

func TestCoinDataWithoutHoldersFor(t *testing.T) {
	client := NewPumpFunClient(os.Getenv("PUMPFUN_API_KEY"), os.Getenv("DATA_IMPULSE_PROXY_URL"))
	coinData, holders, err := client.CoinDataFor("Df6yfrKC8kZE3KNkrHERKzAetSxbrWeniQfyJY4Jpump", false, false)
	if err != nil {
		t.Errorf("Error getting coin data: %v", err)
	}

	if coinData == nil {
		t.Error("Expected coinData to not be nil")
	}

	if holders != nil {
		t.Error("Expected holders to be nil")
	}

	t.Logf("Coin data: %+v", coinData)
}

func TestNumberOfTradesForMint(t *testing.T) {
	client := NewPumpFunClient(os.Getenv("PUMPFUN_API_KEY"), os.Getenv("DATA_IMPULSE_PROXY_URL"))
	count, err := client.numberOfTradesForMint("2HqtEiU1resCmVXyVe8RcpiekiMTPYtVkDK1Xytppump")
	if err != nil {
		t.Errorf("Error getting number of trades: %v", err)
	}

	if count < 1009 {
		t.Errorf("Expected number of trades to be greater than 1009, got %d", count)
	}

	t.Logf("Number of trades: %d", count)
}

func TestAllTradesForMint(t *testing.T) {
	client := NewPumpFunClient(os.Getenv("PUMPFUN_API_KEY"), os.Getenv("DATA_IMPULSE_PROXY_URL"))
	trades, err := client.AllTradesForMint("4cab2KDe448uFKgz21FitpiDM7JWiPzYWdTLTuj7pump")
	if err != nil {
		t.Errorf("Error getting all trades: %v", err)
	}

	t.Logf("Number of trades: %d", len(trades))
	t.Logf("Trades: %+v", trades)
}
