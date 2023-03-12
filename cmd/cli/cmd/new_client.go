package cmd

import (
	"fmt"

	"github.com/dmitrymomot/random"
	"github.com/fatih/color"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/bcrypt"
)

// newClientCmd represents the newClient command
var newClientCmd = &cobra.Command{
	Use:     "new-client",
	Aliases: []string{"nc", "client"},
	Short:   "Generates a new client credentials",
	Long: `
Generates a new client credentials (client_id and client_secret) and prints them to the console.
These credentials are used to authenticate the checkout API. How to set up the client credentials
is described in the Wiki of this repository.
Credentials will not be stored in the database, so you will need to save them somewhere safe.
	`,
	Run: func(cmd *cobra.Command, args []string) {
		clientID := uuid.New().String()
		clientSecret := random.String(50)
		clientSecretHash, err := bcrypt.GenerateFromPassword([]byte(clientSecret), bcrypt.DefaultCost)
		if err != nil {
			panic(err)
		}

		color.Green("\nNew client generated")
		bold := color.New(color.Bold).SprintFunc()
		fmt.Println("---------------------------------------------------------------------------------")
		fmt.Println(bold("Client ID:          "), clientID)
		fmt.Println(bold("Client Secret:      "), clientSecret)
		fmt.Println(bold("Client Secret Hash: "), string(clientSecretHash))
		fmt.Println("---------------------------------------------------------------------------------")
		color.Yellow("Please save the client ID and client secret somewhere safe.")
	},
}

func init() {
	rootCmd.AddCommand(newClientCmd)
}
