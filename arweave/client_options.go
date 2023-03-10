package arweave

import "github.com/everFinance/goar"

// WithWalletInstance sets the wallet instance to be used for transactions.
func WithWalletInstance(wallet *goar.Wallet) ClientOption {
	return func(c *Client) {
		if c.ar != nil {
			panic("wallet instance already set")
		}

		if wallet != nil {
			c.ar = wallet
		}
	}
}

// InitWalletWithPath initializes a wallet instance from a given path.
func InitWalletWithPath(walletPath string) ClientOption {
	return func(c *Client) {
		if c.ar != nil {
			panic("wallet instance already set")
		}

		wallet, err := goar.NewWalletFromPath(walletPath, DefaultArweaveNodeURL)
		if err != nil {
			panic(err)
		}

		c.ar = wallet
	}
}

// InitWalletWithPathAndNode initializes a wallet instance from a given path and custom node URL.
func InitWalletWithPathAndNode(walletPath string, nodeURL string) ClientOption {
	return func(c *Client) {
		if c.ar != nil {
			panic("wallet instance already set")
		}

		wallet, err := goar.NewWalletFromPath(walletPath, nodeURL)
		if err != nil {
			panic(err)
		}

		c.ar = wallet
	}
}

// InitWalletWithPrivateKey initializes a wallet instance from a given private key.
func InitWalletWithPrivateKey(privateKey []byte) ClientOption {
	return func(c *Client) {
		if c.ar != nil {
			panic("wallet instance already set")
		}

		wallet, err := goar.NewWallet(privateKey, DefaultArweaveNodeURL)
		if err != nil {
			panic(err)
		}

		c.ar = wallet
	}
}

// InitWalletWithPrivateKeyAndNode initializes a wallet instance from a given private key and custom node URL.
func InitWalletWithPrivateKeyAndNode(privateKey []byte, nodeURL string) ClientOption {
	return func(c *Client) {
		if c.ar != nil {
			panic("wallet instance already set")
		}

		wallet, err := goar.NewWallet(privateKey, nodeURL)
		if err != nil {
			panic(err)
		}

		c.ar = wallet
	}
}

// WithCustomSpeedFactor sets the speed factor to be used for transactions.
// Default is 0.
func WithCustomSpeedFactor(speedFactor int64) ClientOption {
	return func(c *Client) {
		c.speedFactor = speedFactor
	}
}
