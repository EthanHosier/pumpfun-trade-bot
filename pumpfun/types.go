package pumpfun

type Account struct {
	Address        string  `json:"address"`
	Amount         string  `json:"amount"`
	Decimals       int     `json:"decimals"`
	UIAmount       float64 `json:"uiAmount"`
	UIAmountString string  `json:"uiAmountString"`
}

type CoinHolder struct {
	Address        string
	Amount         float64
	PercentageHeld float64
	// IsDev          bool   <- no clue how this is wokred out
	IsBondingCurve bool
}

type CoinData struct {
	Mint                   string  `json:"mint"`
	Name                   string  `json:"name"`
	Symbol                 string  `json:"symbol"`
	Description            string  `json:"description"`
	ImageURI               string  `json:"image_uri"`
	VideoURI               *string `json:"video_uri"`
	MetadataURI            string  `json:"metadata_uri"`
	Twitter                string  `json:"twitter"`
	Telegram               *string `json:"telegram"`
	BondingCurve           string  `json:"bonding_curve"`
	AssociatedBondingCurve string  `json:"associated_bonding_curve"`
	Creator                string  `json:"creator"`
	CreatedTimestamp       int64   `json:"created_timestamp"`
	RaydiumPool            *string `json:"raydium_pool"`
	Complete               bool    `json:"complete"`
	VirtualSolReserves     int64   `json:"virtual_sol_reserves"`
	VirtualTokenReserves   int64   `json:"virtual_token_reserves"`
	TotalSupply            int64   `json:"total_supply"`
	Website                string  `json:"website"`
	ShowName               bool    `json:"show_name"`
	KingOfTheHillTimestamp int64   `json:"king_of_the_hill_timestamp"`
	MarketCap              float64 `json:"market_cap"`
	ReplyCount             int     `json:"reply_count"`
	LastReply              int64   `json:"last_reply"`
	Nsfw                   bool    `json:"nsfw"`
	MarketID               *string `json:"market_id"`
	Inverted               bool    `json:"inverted"`
	IsCurrentlyLive        bool    `json:"is_currently_live"`
	Username               *string `json:"username"`
	ProfileImage           *string `json:"profile_image"`
	UsdMarketCap           float64 `json:"usd_market_cap"`
}
