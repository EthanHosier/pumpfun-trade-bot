package openai

import (
	"context"
	"os"
	"testing"

	"github.com/joho/godotenv"
)

func init() {
	err := godotenv.Load("../.env")
	if err != nil {
		panic(err)
	}
}

func TestOpenAiClient_ChatCompletion(t *testing.T) {
	client := NewOpenAiClient(os.Getenv("OPENAI_API_KEY"))
	resp, err := client.ChatCompletion(context.Background(), "Hello, world!")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(resp)
}
