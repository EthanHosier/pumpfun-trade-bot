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
	tx, err := client.GetTransactionDataWithRetries("56WfEe3p2zPesLMqm2v2b1zqGwbw2SLiUCNE77pgSTK49cXEry4Qw47L5VTdA5eC3ZcYJt86v7s1EoDgJPF7jhfN", 3)
	if err != nil {
		t.Errorf("Error getting transaction data: %v", err)
	}

	t.Logf("Transaction: %+v", tx)
}
