package openai

import (
	"context"

	"github.com/sashabaranov/go-openai"
)

type OpenAiClient struct {
	client *openai.Client
}

func NewOpenAiClient(apiKey string) *OpenAiClient {
	return &OpenAiClient{client: openai.NewClient(apiKey)}
}

func (o *OpenAiClient) ChatCompletion(ctx context.Context, prompt string) (string, error) {
	resp, err := o.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: openai.GPT4o,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleUser, Content: prompt},
		},
	})
	if err != nil {
		return "", err
	}
	return resp.Choices[0].Message.Content, nil
}
