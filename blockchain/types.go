package blockchain

type Transaction struct {
	Meta struct {
		Err               interface{} `json:"err"`
		Fee               int64       `json:"fee"`
		LogMessages       []string    `json:"logMessages"`
		PostTokenBalances []struct {
			Mint          string `json:"mint"`
			UiTokenAmount struct {
				UiAmount float64 `json:"uiAmount"`
				Decimals int     `json:"decimals"`
			} `json:"uiTokenAmount"`
			Owner string `json:"owner"`
		} `json:"postTokenBalances"`
	} `json:"meta"`
	BlockTime int64 `json:"blockTime"`
}

// LogResponse is the response from the logsNotification method in the websocket connection
type LogResponse struct {
	Jsonrpc string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  struct {
		Result struct {
			Context struct {
				Slot float64 `json:"slot"`
			} `json:"context"`
			Value struct {
				Err struct {
					InstructionError []interface{} `json:"InstructionError"`
				} `json:"err"`
				Logs      []string `json:"logs"`
				Signature string   `json:"signature"`
			} `json:"value"`
		} `json:"result"`
		Subscription int `json:"subscription"`
	} `json:"params"`
}

type WalletTransactionSignature struct {
	Wallet    string `json:"wallet"`
	Signature string `json:"signature"`
}

type BuyTokenResult struct {
	TxID                          string
	AmountInLampts                uint64
	MaxAmountLampts               uint64
	AssociatedTokenAccountAddress string
	TokenAmount                   float64
}

type TransactionDataInstruction struct {
	Accounts  []string
	Data      string
	ProgramID int
}

type TransactionData struct {
	Signatures      []string
	Instructions    []TransactionDataInstruction
	LogMessages     []string
	RecentBlockhash string
}
