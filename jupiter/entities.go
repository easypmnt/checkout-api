package jupiter

// MarketInfo is a market info object structure.
type MarketInfo struct {
	ID                 string `json:"id"`
	Label              string `json:"label"`
	InputMint          string `json:"inputMint"`
	OutputMint         string `json:"outputMint"`
	NotEnoughLiquidity bool   `json:"notEnoughLiquidity"`
	InAmount           string `json:"inAmount"`
	OutAmount          string `json:"outAmount"`
	MinInAmount        string `json:"minInAmount,omitempty"`
	MinOutAmount       string `json:"minOutAmount,omitempty"`
	PriceImpactPct     string `json:"priceImpactPct"`
	LpFee              *Fee   `json:"lpFee"`
	PlatformFee        *Fee   `json:"platformFee"`
}

// Fee is a fee object structure.
type Fee struct {
	Amount string `json:"amount"`
	Mint   string `json:"mint"`
	Pct    string `json:"pct"`
}

// Route is a route object structure.
type Route struct{}

// Price is a price object structure.
type Price struct{}

// PriceMap is a price map objects structure.
type PriceMap map[string]Price

// QuoteParams are the parameters for a quote request.
type QuoteParams struct {
	InputMint  string `json:"inputMint"`  // required
	OutputMint string `json:"outputMint"` // required
	Amount     string `json:"amount"`     // required

	SwapMode            string `json:"swapMode,omitempty"` // Swap mode, default is ExactIn; Available values : ExactIn, ExactOut.
	SlippageBps         int64  `json:"slippageBps,omitempty"`
	FeeBps              int64  `json:"feeBps,omitempty"`              // Fee BPS (only pass in if you want to charge a fee on this swap)
	OnlyDirectRoutes    bool   `json:"onlyDirectRoutes,omitempty"`    // Only return direct routes (no hoppings and split trade)
	AsLegacyTransaction bool   `json:"asLegacyTransaction,omitempty"` // Only return routes that can be done in a single legacy transaction. (Routes might be limited)
	UserPublicKey       string `json:"userPublicKey,omitempty"`       // Public key of the user (only pass in if you want deposit and fee being returned, might slow down query)
}

// QuoteResponse is the response from a quote request.
type QuoteResponse struct {
	InAmount             string       `json:"inAmount"`
	OutAmount            string       `json:"outAmount"`
	PriceImpactPct       int64        `json:"priceImpactPct"`
	MarketInfos          []MarketInfo `json:"marketInfos"`
	Amount               string       `json:"amount"`
	SlippageBps          int64        `json:"slippageBps"`          // minimum: 0, maximum: 10000
	OtherAmountThreshold string       `json:"otherAmountThreshold"` // The threshold for the swap based on the provided slippage: when swapMode is ExactIn the minimum out amount, when swapMode is ExactOut the maximum in amount
	SwapMode             string       `json:"swapMode"`
	Fees                 *struct {
		SignatureFee             int64   `json:"signatureFee"`             // This inidicate the total amount needed for signing transaction(s). Value in lamports.
		OpenOrdersDeposits       []int64 `json:"openOrdersDeposits"`       // This inidicate the total amount needed for deposit of serum order account(s). Value in lamports.
		AtaDeposits              []int64 `json:"ataDeposits"`              // This inidicate the total amount needed for deposit of associative token account(s). Value in lamports.
		TotalFeeAndDeposits      int64   `json:"totalFeeAndDeposits"`      // This indicate the total lamports needed for fees and deposits above.
		MinimumSolForTransaction int64   `json:"minimumSOLForTransaction"` // This inidicate the minimum lamports needed for transaction(s). Might be used to create wrapped SOL and will be returned when the wrapped SOL is closed. Also ensures rent exemption of the wallet.
	} `json:"fees,omitempty"`
}
