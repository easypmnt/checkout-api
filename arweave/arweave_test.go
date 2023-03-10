package arweave_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/easypmnt/checkout-api/arweave"
	"github.com/easypmnt/checkout-api/utils"
	"github.com/stretchr/testify/require"
)

func TestUploadTokenMetadata(t *testing.T) {
	t.Skip("comment this line to test arweave uploading")

	client := arweave.NewClient(arweave.InitWalletWithPath("./arweave-key.json"))
	require.NotNil(t, client)

	imgBytes, err := utils.GetFileByPath("./example.jpeg")
	require.NoError(t, err)
	require.NotNil(t, imgBytes)

	price, priceBps, err := client.CalcPrice(imgBytes)
	require.NoError(t, err)
	require.Greater(t, price, float64(0))
	require.Greater(t, priceBps, int64(0))

	imageURL, err := client.Upload(imgBytes, "image/jpeg", ".jpeg")
	require.NoError(t, err)
	require.NotEmpty(t, imageURL)

	metadata := &struct {
		Name        string `json:"name"`
		Symbol      string `json:"symbol"`
		Description string `json:"description"`
		Image       string `json:"image"`
		ExternalURL string `json:"external_url,omitempty"`
	}{
		Name:        "Test Fungible Token",
		Symbol:      "TFT",
		Description: "Test token generated by easypmnt/checkout-api package",
		Image:       imageURL,
		ExternalURL: "https://github.com/easypmnt/checkout-api",
	}

	b, err := json.Marshal(metadata)
	require.NoError(t, err)

	metaUrl, err := client.Upload(b, "application/json", ".json")
	require.NoError(t, err)
	require.NotEmpty(t, metaUrl)

	fmt.Println(metaUrl)
}
