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
	client := NewBlockchainClient(os.Getenv("HELIUS_API_KEY"))
	tx, err := client.GetTransactionDataWithRetries("2GztCWvPiuHKHgXboSi2A5L8ENonrqUbS3LHAZqdKHpSiKdfoA4mrYa4TfqQWKuzUcU8MYkeecU484zn3E2C2TtY", 3)
	if err != nil {
		t.Errorf("Error getting transaction data: %v", err)
	}

	t.Logf("Transaction: %+v", tx)
}
