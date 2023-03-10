package arweave

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/everFinance/goar"
	"github.com/everFinance/goar/types"
	"github.com/everFinance/goar/utils"
)

const (
	// DefaultArweaveNodeURL is a default arweave node URL
	DefaultArweaveNodeURL = "https://arweave.net"
)

type (
	// Client is a wrapper for goar.Wallet
	Client struct {
		ar          *goar.Wallet
		speedFactor int64
	}

	// Client option function interface to configure the client.
	ClientOption func(*Client)
)

// NewClient creates a new arweave client
func NewClient(opts ...ClientOption) *Client {
	c := &Client{
		speedFactor: 0,
	}

	for _, opt := range opts {
		opt(c)
	}

	if c.ar == nil {
		panic("wallet instance not set")
	}

	return c
}

// CalcPrice returns price for store data
// 1 AR = 1000000000000 Winston (12 zeros) and 1 Winston = 0.000000000001 AR
// [0] - float64 - price in AR coins
// [1] - int64 - price in Winstons
// [2] - error
func (c *Client) CalcPrice(data []byte) (float64, int64, error) {
	reward, err := c.ar.Client.GetTransactionPrice(data, nil)
	if err != nil {
		return 0, 0, fmt.Errorf("%w: %s", ErrFailedToCalcPrice, err.Error())
	}

	ar := utils.WinstonToAR(big.NewInt(reward))
	rewardsAR, _ := ar.Float64()

	return rewardsAR, reward, nil
}

// Upload function uploads data to Arweave
// returns file URL or error
func (c *Client) Upload(data []byte, contentType, ext string) (string, error) {
	ext = strings.TrimLeft(strings.TrimSpace(strings.ToLower(ext)), ".")

	tx, err := c.ar.SendDataSpeedUp(data, []types.Tag{{
		Name:  "Content-Type",
		Value: contentType,
	}}, c.speedFactor)
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrFailedToUploadData, err.Error())
	}

	return fmt.Sprintf("https://www.arweave.net/%s?ext=%s", tx.ID, ext), nil
}
