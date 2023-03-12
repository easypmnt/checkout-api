package cmd

import (
	"fmt"

	"github.com/easypmnt/checkout-api/internal/utils"
	"github.com/fatih/color"
	"github.com/portto/solana-go-sdk/types"
	"github.com/spf13/cobra"
)

// newWalletCmd represents the newWallet command
var newWalletCmd = &cobra.Command{
	Use:     "new-wallet",
	Aliases: []string{"nw", "wallet"},
	Short:   "Generates a new wallet",
	Long: `
Generates a new wallet public key, private key and prints it to the console. 
Generated wallet will not be stored in the database, so you will need to save 
it somewhere.
	`,
	Run: func(cmd *cobra.Command, args []string) {
		wallet := types.NewAccount()

		color.Green("\nNew wallet generated")
		fmt.Println("---------------------------------------------------------------------------------")
		bold := color.New(color.Bold).SprintFunc()
		fmt.Println(bold("Public Key:  "), wallet.PublicKey.ToBase58())
		fmt.Println(bold("Private Key: "), utils.BytesToBase58(wallet.PrivateKey))
		fmt.Println("---------------------------------------------------------------------------------")
		color.Yellow("Save your private key somewhere safe, you will need it to sign transactions")
	},
}

func init() {
	rootCmd.AddCommand(newWalletCmd)
}
