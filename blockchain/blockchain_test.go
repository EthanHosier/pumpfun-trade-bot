package blockchain

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

func TestGetTransactionDataWithRetries(t *testing.T) {
	client := NewBlockchainClient(os.Getenv("HELIUS_API_KEY"), nil)
	tx, err := client.GetTransactionDataWithRetries("2UbydyYxAmzvysksVfFmVEfLB1NawTS7GreAuAsauoh8npJSzfZukw5QyU4RQMWp5DYRzQdAF2HqURGxUEUPbUku", 3)
	if err != nil {
		t.Errorf("Error getting transaction data: %v", err)
	}

	t.Logf("Transaction: %+v", tx)
}

func TestGetTransaction2(t *testing.T) {
	client := NewBlockchainClient(os.Getenv("HELIUS_API_KEY"), nil)
	tx, err := client.getTransactionData2("2UbydyYxAmzvysksVfFmVEfLB1NawTS7GreAuAsauoh8npJSzfZukw5QyU4RQMWp5DYRzQdAF2HqURGxUEUPbUku")
	if err != nil {
		t.Errorf("Error getting transaction data: %v", err)
	}

	t.Logf("Transaction: %+v", tx)
}
