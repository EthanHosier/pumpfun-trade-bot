package botFinder

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ethanhosier/pumpfun-trade-bot/openai"
	"github.com/ethanhosier/pumpfun-trade-bot/pumpfun"
)

const findBotsInTransactionPrompt = `Given the following list of transactions, identify potential bot user codes. A bot is characterized by:

- **Burst trades at the same timestamp**: Multiple trades occurring at the exact same timestamp from different users.
- **Similar trade amounts**: Trades involving similar 'SolAmount' or 'TokenAmount', especially if the amounts seem algorithmically determined.
- **Repetitive user behavior**: Users executing multiple trades in a short time span with similar patterns.

**Instructions:**

- Analyze the transactions based on the criteria above.
- Return the list of potential bot user codes as a JSON array.
- Do not include any additional explanation or formatting—just the JSON array.

**Transactions:**
%+v
`

func (b *BotFinder) findBotsInTradesWithChatgpt(trades []pumpfun.Trade) ([]string, error) {
	prompt := fmt.Sprintf(findBotsInTransactionPrompt, trades)
	resp, err := b.openaiClient.ChatCompletion(context.Background(), prompt)
	if err != nil {
		return nil, err
	}

	jsonDataStr, err := openai.ExtractJsonData(resp, openai.JSONArray)
	if err != nil {
		return nil, err
	}

	var userCodes []string
	err = json.Unmarshal([]byte(jsonDataStr), &userCodes)
	if err != nil {
		return nil, err
	}

	return userCodes, nil
}
