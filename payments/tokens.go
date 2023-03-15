package payments

import "strings"

const (
	SOL  = "So11111111111111111111111111111111111111112"
	USDC = "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"
	USDT = "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB"
)

// Default mints.
var defaultMints = map[string]string{
	"USDC": USDC,
	"USDT": USDT,
	"SOL":  SOL,
}

// MintAddress returns the mint address by symbol.
// If the symbol is not found, it returns the fallback address.
// Supports only default mints.
func MintAddress(currency string, fallback string) string {
	if currency == "" {
		currency = fallback
	}
	if address, ok := defaultMints[strings.ToUpper(currency)]; ok {
		return address
	}
	if len(currency) < 40 {
		return SOL
	}
	return currency
}

// IsSOL checks if the currency is SOL.
func IsSOL(currency string) bool {
	c := strings.ToUpper(currency)
	return c == "SOL" || defaultMints["SOL"] == currency
}
