package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/easypmnt/checkout-api/arweave"
	"github.com/easypmnt/checkout-api/internal/utils"
	"github.com/easypmnt/checkout-api/solana"
	"github.com/easypmnt/checkout-api/solana/metadata"
	"github.com/fatih/color"
	"github.com/portto/solana-go-sdk/types"
	"github.com/spf13/cobra"
)

// createTokenCmd represents the createToken command
var createTokenCmd = &cobra.Command{
	Use:     "create-token",
	Aliases: []string{"ct", "token"},
	Short:   "Creates a new fungible token",
	Long: `
Creates a new fungible token. The token is created with the given name 
and symbol and is assigned to the address of the creator. The creator 
can then mint and burn tokens as they see fit. The total supply of the 
token is initially 0.

To create a new token, the creator must pay a network fee, so make sure
to have enough funds in your account to cover the fee (recommend to have at least 0.2 SOL).
	`,
	RunE: func(cmd *cobra.Command, args []string) error {
		decimals, err := cmd.Flags().GetInt8("decimals")
		if err != nil {
			return fmt.Errorf("decimals: %w", err)
		}
		attr, _ := cmd.Flags().GetStringToString("attributes") // nolint:errcheck

		mintAddr, err := mintFungibleToken(cmd.Context(), MintFungibleTokenParams{
			SolanaRPCEndpoint: cmd.Flag("solana-rpc-endpoint").Value.String(),
			ArweaveKey:        cmd.Flag("arweave-key").Value.String(),
			MintAuthority:     cmd.Flag("mint-authority").Value.String(),
			FeePayer:          cmd.Flag("fee-payer").Value.String(),

			Name:        cmd.Flag("name").Value.String(),
			Symbol:      cmd.Flag("symbol").Value.String(),
			Decimals:    decimals,
			Icon:        cmd.Flag("icon").Value.String(),
			ExternalURL: cmd.Flag("external_url").Value.String(),
			Description: cmd.Flag("description").Value.String(),
			Attributes:  attr,
		})
		if err != nil {
			return fmt.Errorf("mint fungible token: %w", err)
		}

		color.Green("Token created successfully. Mint address: %s", mintAddr)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(createTokenCmd)

	createTokenCmd.Flags().String("solana-rpc-endpoint", "https://api.devnet.solana.com", "Solana RPC endpoint URL.")
	createTokenCmd.Flags().String("name", "", "Name of the token")
	createTokenCmd.Flags().String("symbol", "", "Symbol of the token")
	createTokenCmd.Flags().Int8("decimals", 9, "Number of decimals for the token. Recommend set the same as a main token you will you to accept payments.")
	createTokenCmd.Flags().String("icon", "", "Path to the icon of the token.")
	createTokenCmd.Flags().String("external_url", "", "External URL of the token (optional).")
	createTokenCmd.Flags().String("description", "", "Description of the token (optional).")
	createTokenCmd.Flags().String("arweave-key", "./arweave-key.json", "Path to the arweave key to upload the token metadata to Arweave.")
	createTokenCmd.Flags().String("mint-authority", "[fee-payer]", "Base58 encoded private key of the mint authority (is signer).")
	createTokenCmd.Flags().String("fee-payer", "", "Base58 encoded private key of the fee payer.")
	createTokenCmd.Flags().StringToString("attributes", map[string]string{}, "Attributes of the token (optional). Example: --attributes key1=value1,key2=value2")
}

type MintFungibleTokenParams struct {
	SolanaRPCEndpoint string
	ArweaveKey        string
	MintAuthority     string
	FeePayer          string

	Name        string
	Symbol      string
	Decimals    int8
	Icon        string
	ExternalURL string
	Description string
	Attributes  map[string]string
}

// Validate validates the parameters.
func (p MintFungibleTokenParams) Validate() error {
	if p.SolanaRPCEndpoint == "" {
		return fmt.Errorf("solana-rpc-endpoint is required")
	}
	if p.MintAuthority == "" && p.FeePayer == "" {
		return fmt.Errorf("mint-authority or fee-payer is empty, at least one must be provided")
	}
	if p.Name == "" {
		return fmt.Errorf("name is required")
	}
	if len(p.Name) > 32 {
		return fmt.Errorf("name must be less than or equal to 32 characters")
	}
	if p.Symbol == "" {
		return fmt.Errorf("symbol is required")
	}
	if len(p.Symbol) > 10 {
		return fmt.Errorf("symbol must be less than or equal to 10 characters")
	}
	if p.Decimals < 0 || p.Decimals > 9 {
		return fmt.Errorf("decimals must be between 0 and 9")
	}
	if p.Icon != "" {
		if _, err := os.Stat(p.Icon); os.IsNotExist(err) {
			return fmt.Errorf("icon file does not exist")
		}
	}
	if p.Description == "" {
		return fmt.Errorf("description is required")
	}
	if p.ArweaveKey != "" {
		if _, err := os.Stat(p.ArweaveKey); os.IsNotExist(err) {
			return fmt.Errorf("arweave key file does not exist")
		}
	}
	return nil
}

func mintFungibleToken(ctx context.Context, arg MintFungibleTokenParams) (string, error) {
	color.Yellow("Minting a new fungible token...")
	if err := arg.Validate(); err != nil {
		return "", fmt.Errorf("invalid parameters: %w", err)
	}

	client := solana.NewClient(solana.WithRPCEndpoint(arg.SolanaRPCEndpoint))

	if arg.MintAuthority == "" {
		arg.MintAuthority = arg.FeePayer
	} else if arg.FeePayer == "" {
		arg.FeePayer = arg.MintAuthority
	}

	mint := types.NewAccount()
	mintAuth, err := types.AccountFromBase58(arg.MintAuthority)
	if err != nil {
		return "", fmt.Errorf("failed to parse mint authority: %w", err)
	}
	feePayer, err := types.AccountFromBase58(arg.FeePayer)
	if err != nil {
		return "", fmt.Errorf("failed to parse fee payer: %w", err)
	}

	// build transaction
	color.Yellow("Building transaction...")
	tx, err := solana.NewTransactionBuilder(client).
		SetFeePayer(feePayer.PublicKey.ToBase58()).
		AddSigner(mint).
		AddSigner(mintAuth).
		AddSigner(feePayer).
		AddInstruction(solana.CreateFungibleToken(solana.CreateFungibleTokenParam{
			Mint:        mint.PublicKey.ToBase58(),
			Owner:       mintAuth.PublicKey.ToBase58(),
			FeePayer:    feePayer.PublicKey.ToBase58(),
			Decimals:    uint8(arg.Decimals),
			TokenName:   arg.Name,
			TokenSymbol: arg.Symbol,
		})).
		Build(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to build transaction: %w", err)
	}

	// send transaction
	color.Yellow("Sending transaction...")
	txSig, err := client.SendTransaction(ctx, tx)
	if err != nil {
		return "", fmt.Errorf("failed to send transaction: %w", err)
	}

	// wait for transaction to be confirmed
	color.Yellow("Waiting for transaction to be confirmed...")
	status, err := client.WaitForTransactionConfirmed(ctx, txSig, time.Minute)
	if err != nil {
		return "", fmt.Errorf("failed to wait for transaction to be confirmed: %w", err)
	}
	if status != solana.TransactionStatusSuccess {
		return "", fmt.Errorf("transaction failed with status: %s", status)
	}
	color.Green("Transaction confirmed! Check it on Solana Explorer: https://explorer.solana.com/tx/%s", txSig)

	if arg.ArweaveKey == "" {
		color.Yellow("Arweave key is not set, skipping uploading metadata...")
		return mint.PublicKey.ToBase58(), nil
	}

	arClient := arweave.NewClient(arweave.InitWalletWithPath(arg.ArweaveKey))

	// upload icon to arweave
	color.Yellow("Uploading icon to Arweave...")
	imgBytes, err := utils.GetFileByPath(arg.Icon)
	if err != nil {
		return "", fmt.Errorf("failed to read icon file: %w", err)
	}

	iconURI, err := arClient.Upload(imgBytes, utils.GetFileTypeByURI(arg.Icon), filepath.Ext(arg.Icon))
	if err != nil {
		return "", fmt.Errorf("failed to upload icon to arweave: %w", err)
	}
	color.Green("Icon uploaded! It available by URI: %s", iconURI)

	// upload metadata to arweave
	color.Yellow("Uploading metadata to Arweave...")
	var md *metadata.Metadata
	if arg.Decimals == 0 {
		mdBuilder := metadata.NewFungibleAssetMetadataBuilder().
			SetName(arg.Name).
			SetSymbol(arg.Symbol).
			SetDescription(arg.Description).
			SetExternalURL(arg.ExternalURL).
			SetImage(iconURI)
		for k, v := range arg.Attributes {
			mdBuilder = mdBuilder.SetAttribute(k, v)
		}
		md, err = mdBuilder.Build()
		if err != nil {
			return "", fmt.Errorf("failed to build metadata: %w", err)
		}
	} else {
		md, err = metadata.NewFungibleTokenMetadataBuilder().
			SetName(arg.Name).
			SetSymbol(arg.Symbol).
			SetDescription(arg.Description).
			SetExternalURL(arg.ExternalURL).
			SetImage(iconURI).
			Build()
		if err != nil {
			return "", fmt.Errorf("failed to build metadata: %w", err)
		}
	}

	mdb, err := md.ToJSON()
	if err != nil {
		return "", fmt.Errorf("failed to marshal metadata: %w", err)
	}

	color.Yellow("Uploading metadata to Arweave...")
	metadataURI, err := arClient.Upload(mdb, "application/json", ".json")
	if err != nil {
		return "", fmt.Errorf("failed to upload metadata: %w", err)
	}
	color.Green("Metadata uploaded to Arweave: %s", metadataURI)

	// update metadata on chain
	color.Yellow("Updating metadata on chain...")
	tx, err = solana.NewTransactionBuilder(client).
		SetFeePayer(feePayer.PublicKey.ToBase58()).
		AddSigner(mintAuth).
		AddSigner(feePayer).
		AddInstruction(solana.UpdateFungibleMetadata(solana.UpdateFungibleMetadataParams{
			Mint:            mint.PublicKey.ToBase58(),
			UpdateAuthority: mintAuth.PublicKey.ToBase58(),
			MetadataUri:     metadataURI,
		})).
		Build(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to build transaction: %w", err)
	}

	// send transaction
	color.Yellow("Sending transaction...")
	txSig, err = client.SendTransaction(ctx, tx)
	if err != nil {
		return "", fmt.Errorf("failed to send transaction: %w", err)
	}
	color.Cyan("Transaction sent! Check it on Solana Explorer: https://explorer.solana.com/tx/%s", txSig)

	// wait for transaction to be confirmed
	color.Yellow("Waiting for transaction to be confirmed...")
	status, err = client.WaitForTransactionConfirmed(ctx, txSig, time.Minute)
	if err != nil {
		return "", fmt.Errorf("failed to wait for transaction to be confirmed: %w", err)
	}
	if status != solana.TransactionStatusSuccess {
		return "", fmt.Errorf("transaction failed with status: %s", status)
	}
	color.Green("Transaction confirmed! Check it on Solana Explorer: https://explorer.solana.com/tx/%s", txSig)

	return mint.PublicKey.ToBase58(), nil
}
