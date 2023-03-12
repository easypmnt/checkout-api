package solana_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/dmitrymomot/go-env"
	"github.com/easypmnt/checkout-api/internal/utils"
	"github.com/easypmnt/checkout-api/solana"
	"github.com/portto/solana-go-sdk/types"
	"github.com/stretchr/testify/require"
)

var (
	solanaRPCEndpoint = env.GetString("SOLANA_RPC_ENDPOINT", "https://api.devnet.solana.com")
	solanaWSSEndpoint = env.GetString("SOLANA_WSS_ENDPOINT", "wss://api.devnet.solana.com")

	wallet1, _ = types.AccountFromBase58("4JVyzx75j9s91TgwVqSPFN4pb2D8ACPNXUKKnNBvXuGukEzuFEg3sLqhPGwYe9RRbDnVoYHjz4bwQ5yUfyRZVGVU")
	wallet2, _ = types.AccountFromBase58("2x3dkFDgZbq9kjRPRv8zzXzcpj8rZKLCTEgGj52KT7RUmkNy8gSaSDCP5vDhPkspAam6WPEiZxVUatA8nHSSSj79")
)

func TestSendSOL_WithReference(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := solana.NewClient(solana.WithRPCEndpoint(solanaRPCEndpoint))

	amountToSend := uint64(2500000)              // 0.0025 SOL
	minBalanceAmount := uint64(amountToSend * 2) // 0.005 SOL
	referenceAcc := types.NewAccount()
	fmt.Println("referenceAcc", referenceAcc.PublicKey.ToBase58())

	// check wallet1 balance of SOL
	t.Run("check wallet1 balance of SOL", func(t *testing.T) {
		balance, err := client.GetSOLBalance(ctx, wallet1.PublicKey.ToBase58())
		require.NoError(t, err)
		if balance.Amount < minBalanceAmount {
			tx, err := client.RequestAirdrop(ctx, wallet1.PublicKey.ToBase58(), 1000000000)
			require.NoError(t, err)
			require.NotNil(t, tx)
			// wait for transaction to be confirmed
			status, err := client.WaitForTransactionConfirmed(ctx, tx, time.Minute)
			require.NoError(t, err)
			require.EqualValues(t, solana.TransactionStatusSuccess, status)
			// check wallet1 balance of SOL
			balance, err = client.GetSOLBalance(ctx, wallet1.PublicKey.ToBase58())
			require.NoError(t, err)
			require.GreaterOrEqual(t, balance.Amount, uint64(1000000000))
		}
	})

	// check wallet2 balance of SOL
	wallet2InitBalance, err := client.GetSOLBalance(ctx, wallet2.PublicKey.ToBase58())
	require.NoError(t, err)

	t.Run("send SOL", func(t *testing.T) {
		// build transaction
		tx, err := solana.NewTransactionBuilder(client).
			SetFeePayer(wallet1.PublicKey.ToBase58()).
			AddInstruction(solana.TransferSOL(solana.TransferSOLParams{
				Sender:    wallet1.PublicKey.ToBase58(),
				Recipient: wallet2.PublicKey.ToBase58(),
				Reference: referenceAcc.PublicKey.ToBase58(),
				Amount:    amountToSend,
			})).
			Build(ctx)
		require.NoError(t, err)
		require.NotNil(t, tx)

		// sign transaction
		tx, err = solana.SignTransaction(tx, wallet1)
		require.NoError(t, err)
		require.NotNil(t, tx)

		// send transaction
		txSig, err := client.SendTransaction(ctx, tx)
		if err != nil {
			// retry if transaction failed
			txSig, err = client.SendTransaction(ctx, tx)
		}
		require.NoError(t, err)
		require.NotNil(t, txSig)
		fmt.Println("txSig", txSig)

		// wait for transaction to be confirmed
		status, err := client.WaitForTransactionConfirmed(ctx, txSig, time.Minute)
		require.NoError(t, err)
		require.EqualValues(t, solana.TransactionStatusSuccess, status)

		// check wallet2 balance of SOL
		wallet2Balance, err := client.GetSOLBalance(ctx, wallet2.PublicKey.ToBase58())
		require.NoError(t, err)
		require.EqualValues(t, wallet2InitBalance.Amount+amountToSend, wallet2Balance.Amount)
	})

	t.Run("verify transaction by reference", func(t *testing.T) {
		_, txResp, err := client.GetOldestTransactionForWallet(ctx, referenceAcc.PublicKey.ToBase58(), "")
		require.NoError(t, err)
		require.NotNil(t, txResp)
		require.True(t, txResp.Meta.PreBalances[0] > txResp.Meta.PostBalances[0])
		require.True(t, txResp.Meta.PreBalances[1] < txResp.Meta.PostBalances[1])
		require.EqualValues(t, txResp.Meta.PostBalances[0], txResp.Meta.PreBalances[0]-int64(amountToSend)-int64(txResp.Meta.Fee))
	})

	// check wallet2 balance of SOL
	wallet2Balance, err := client.GetSOLBalance(ctx, wallet2.PublicKey.ToBase58())
	require.NoError(t, err)
	require.EqualValues(t, wallet2InitBalance.Amount+amountToSend, wallet2Balance.Amount)
}

func TestGetDeprecatedTokenMeta(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := solana.NewClient(solana.WithRPCEndpoint(solanaRPCEndpoint))

	mintAddr := "So11111111111111111111111111111111111111112"
	tokenMeta, err := client.GetFungibleTokenMetadata(ctx, mintAddr)
	// utils.PrettyPrint(tokenMeta)
	require.NoError(t, err)
	require.NotNil(t, tokenMeta)
	require.EqualValues(t, "Wrapped SOL", tokenMeta.Name)
	require.EqualValues(t, "SOL", tokenMeta.Symbol)
	require.EqualValues(t, 9, tokenMeta.Decimals)
	require.EqualValues(t, "https://solana.com/", tokenMeta.ExternalURL)
}

func TestGetOnChainTokenMeta(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := solana.NewClient(solana.WithRPCEndpoint(solanaRPCEndpoint))

	mintAddr := "DUSTawucrTsGU8hcqRdHDCbuYhCPADMLM2VcCb8VnFnQ"
	tokenMeta, err := client.GetFungibleTokenMetadata(ctx, mintAddr)
	require.NoError(t, err)
	require.NotNil(t, tokenMeta)
	require.EqualValues(t, "DUST Protocol", tokenMeta.Name)
	require.EqualValues(t, "DUST", tokenMeta.Symbol)
	require.EqualValues(t, uint8(9), tokenMeta.Decimals)
	require.Empty(t, tokenMeta.ExternalURL)
}

func TestFungibleToken(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := solana.NewClient(solana.WithRPCEndpoint(solanaRPCEndpoint))

	var (
		mint         = types.NewAccount()
		tokenName    = "Test Fungible Token"
		tokenSymbol  = "TFT"
		metadataURI  = "https://www.arweave.net/wxfM3Ca3A4gRQg7oc8pvqPX7AN0dh2hKQ-nbE1ZKxkc?ext=json"
		referenceAcc = types.NewAccount()
	)

	t.Run("create fungible token", func(t *testing.T) {
		// build transaction
		tx, err := solana.NewTransactionBuilder(client).
			SetFeePayer(wallet1.PublicKey.ToBase58()).
			AddSigner(mint).
			AddInstruction(solana.CreateFungibleToken(solana.CreateFungibleTokenParam{
				Mint:        mint.PublicKey.ToBase58(),
				Owner:       wallet1.PublicKey.ToBase58(),
				FeePayer:    wallet1.PublicKey.ToBase58(),
				Decimals:    0,
				TokenName:   tokenName,
				TokenSymbol: tokenSymbol,
			})).
			Build(ctx)
		require.NoError(t, err)
		require.NotNil(t, tx)

		// sign transaction
		tx, err = solana.SignTransaction(tx, wallet1)
		require.NoError(t, err)
		require.NotNil(t, tx)

		// send transaction
		txSig, err := client.SendTransaction(ctx, tx)
		if err != nil {
			// retry if transaction failed
			txSig, err = client.SendTransaction(ctx, tx)
		}
		require.NoError(t, err)
		require.NotNil(t, txSig)
		fmt.Println("txSig", txSig)

		// wait for transaction to be confirmed
		status, err := client.WaitForTransactionConfirmed(ctx, txSig, time.Minute)
		require.NoError(t, err)
		require.EqualValues(t, solana.TransactionStatusSuccess, status)

		// check token metadata
		tokenMeta, err := client.GetFungibleTokenMetadata(ctx, mint.PublicKey.ToBase58())
		require.NoError(t, err)
		require.NotNil(t, tokenMeta)
		require.EqualValues(t, tokenName, tokenMeta.Name)
		require.EqualValues(t, tokenSymbol, tokenMeta.Symbol)
		require.EqualValues(t, 0, tokenMeta.Decimals)
		require.Empty(t, tokenMeta.ExternalURL)
		require.Empty(t, tokenMeta.LogoURI)
	})

	t.Run("update metadata", func(t *testing.T) {
		// build transaction
		tx, err := solana.NewTransactionBuilder(client).
			SetFeePayer(wallet1.PublicKey.ToBase58()).
			AddInstruction(solana.UpdateFungibleMetadata(solana.UpdateFungibleMetadataParams{
				Mint:            mint.PublicKey.ToBase58(),
				UpdateAuthority: wallet1.PublicKey.ToBase58(),
				MetadataUri:     metadataURI,
			})).
			Build(ctx)
		require.NoError(t, err)
		require.NotNil(t, tx)

		// sign transaction
		tx, err = solana.SignTransaction(tx, wallet1)
		require.NoError(t, err)
		require.NotNil(t, tx)

		// send transaction
		txSig, err := client.SendTransaction(ctx, tx)
		if err != nil {
			// retry if transaction failed
			txSig, err = client.SendTransaction(ctx, tx)
		}
		require.NoError(t, err)
		require.NotNil(t, txSig)
		fmt.Println("txSig", txSig)

		// wait for transaction to be confirmed
		status, err := client.WaitForTransactionConfirmed(ctx, txSig, time.Minute)
		require.NoError(t, err)
		require.EqualValues(t, solana.TransactionStatusSuccess, status)

		// check token metadata
		tokenMeta, err := client.GetFungibleTokenMetadata(ctx, mint.PublicKey.ToBase58())
		require.NoError(t, err)
		require.NotNil(t, tokenMeta)
		require.EqualValues(t, tokenName, tokenMeta.Name)
		require.EqualValues(t, tokenSymbol, tokenMeta.Symbol)
		require.EqualValues(t, 0, tokenMeta.Decimals)
		require.NotEmpty(t, tokenMeta.ExternalURL)
		require.NotEmpty(t, tokenMeta.LogoURI)
	})

	t.Run("mint fungible token", func(t *testing.T) {
		// build transaction
		tx, err := solana.NewTransactionBuilder(client).
			SetFeePayer(wallet1.PublicKey.ToBase58()).
			AddInstruction(solana.MintFungibleToken(solana.MintFungibleTokenParams{
				Funder:    wallet2.PublicKey.ToBase58(),
				Mint:      mint.PublicKey.ToBase58(),
				MintOwner: wallet1.PublicKey.ToBase58(),
				MintTo:    wallet2.PublicKey.ToBase58(),
				Amount:    1000,
			})).
			Build(ctx)
		require.NoError(t, err)
		require.NotNil(t, tx)

		// sign transaction by wallet1
		tx, err = solana.SignTransaction(tx, wallet1)
		require.NoError(t, err)
		require.NotNil(t, tx)

		// sign transaction by wallet2
		tx, err = solana.SignTransaction(tx, wallet2)
		require.NoError(t, err)
		require.NotNil(t, tx)

		// send transaction
		txSig, err := client.SendTransaction(ctx, tx)
		if err != nil {
			// retry if transaction failed
			txSig, err = client.SendTransaction(ctx, tx)
		}
		require.NoError(t, err)
		require.NotNil(t, txSig)
		fmt.Println("txSig", txSig)

		// wait for transaction to be confirmed
		status, err := client.WaitForTransactionConfirmed(ctx, txSig, time.Minute)
		require.NoError(t, err)
		require.EqualValues(t, solana.TransactionStatusSuccess, status)

		// check wallet2 balance of token
		wallet2Balance, err := client.GetTokenBalance(ctx, wallet2.PublicKey.ToBase58(), mint.PublicKey.ToBase58())
		require.NoError(t, err)
		require.EqualValues(t, 1000, wallet2Balance.Amount)
		require.EqualValues(t, uint8(0), wallet2Balance.Decimals)
		require.EqualValues(t, "1000", wallet2Balance.UIAmountString)
		require.EqualValues(t, float64(1000), wallet2Balance.UIAmount)
	})

	t.Run("transfer fungible token", func(t *testing.T) {
		// build transaction
		tx, err := solana.NewTransactionBuilder(client).
			SetFeePayer(wallet2.PublicKey.ToBase58()).
			AddInstruction(solana.CreateAssociatedTokenAccountIfNotExists(solana.CreateAssociatedTokenAccountParam{
				Funder: wallet2.PublicKey.ToBase58(),
				Owner:  wallet1.PublicKey.ToBase58(),
				Mint:   mint.PublicKey.ToBase58(),
			})).
			AddInstruction(solana.TransferToken(solana.TransferTokenParam{
				Sender:    wallet2.PublicKey.ToBase58(),
				Recipient: wallet1.PublicKey.ToBase58(),
				Mint:      mint.PublicKey.ToBase58(),
				Reference: referenceAcc.PublicKey.ToBase58(),
				Amount:    10,
			})).
			Build(ctx)
		require.NoError(t, err)
		require.NotNil(t, tx)

		// sign transaction by wallet2
		tx, err = solana.SignTransaction(tx, wallet2)
		require.NoError(t, err)
		require.NotNil(t, tx)

		// send transaction
		txSig, err := client.SendTransaction(ctx, tx)
		if err != nil {
			// retry if transaction failed
			txSig, err = client.SendTransaction(ctx, tx)
		}
		require.NoError(t, err)
		require.NotNil(t, txSig)
		fmt.Println("txSig", txSig)

		// wait for transaction to be confirmed
		status, err := client.WaitForTransactionConfirmed(ctx, txSig, time.Minute)
		require.NoError(t, err)
		require.EqualValues(t, solana.TransactionStatusSuccess, status)

		// check wallet2 balance of token
		wallet2Balance, err := client.GetTokenBalance(ctx, wallet2.PublicKey.ToBase58(), mint.PublicKey.ToBase58())
		require.NoError(t, err)
		require.EqualValues(t, 990, wallet2Balance.Amount)
		require.EqualValues(t, uint8(0), wallet2Balance.Decimals)
		require.EqualValues(t, "990", wallet2Balance.UIAmountString)
		require.EqualValues(t, float64(990), wallet2Balance.UIAmount)

		// check wallet1 balance of token
		wallet1Balance, err := client.GetTokenBalance(ctx, wallet1.PublicKey.ToBase58(), mint.PublicKey.ToBase58())
		require.NoError(t, err)
		require.EqualValues(t, 10, wallet1Balance.Amount)
		require.EqualValues(t, uint8(0), wallet1Balance.Decimals)
		require.EqualValues(t, "10", wallet1Balance.UIAmountString)
		require.EqualValues(t, float64(10), wallet1Balance.UIAmount)
	})

	t.Run("verify transaction by reference", func(t *testing.T) {
		_, txResp, err := client.GetOldestTransactionForWallet(ctx, referenceAcc.PublicKey.ToBase58(), "")
		require.NoError(t, err)
		require.NotNil(t, txResp)
		err = solana.CheckTokenTransferTransaction(
			txResp.Meta,
			txResp.Transaction,
			mint.PublicKey.ToBase58(),
			wallet1.PublicKey.ToBase58(),
			10,
		)
		require.NoError(t, err)
		// utils.PrettyPrint(txResp)
	})

	t.Run("burn fungible token", func(t *testing.T) {
		t.Run("wallet1", func(t *testing.T) {
			balance, err := client.GetTokenBalance(ctx, wallet1.PublicKey.ToBase58(), mint.PublicKey.ToBase58())
			require.NoError(t, err)
			require.Greater(t, balance.Amount, uint64(0))

			// build transaction
			tx, err := solana.NewTransactionBuilder(client).
				SetFeePayer(wallet1.PublicKey.ToBase58()).
				AddInstruction(solana.BurnToken(solana.BurnTokenParams{
					Mint:              mint.PublicKey.ToBase58(),
					TokenAccountOwner: wallet1.PublicKey.ToBase58(),
					Amount:            balance.Amount,
				})).
				AddInstruction(solana.CloseTokenAccount(solana.CloseTokenAccountParams{
					Owner: wallet1.PublicKey.ToBase58(),
					Mint:  utils.Pointer(mint.PublicKey.ToBase58()),
				})).
				Build(ctx)
			require.NoError(t, err)
			require.NotNil(t, tx)

			// sign transaction by wallet1
			tx, err = solana.SignTransaction(tx, wallet1)
			require.NoError(t, err)
			require.NotNil(t, tx)

			// send transaction
			txSig, err := client.SendTransaction(ctx, tx)
			if err != nil {
				// retry if transaction failed
				txSig, err = client.SendTransaction(ctx, tx)
			}
			require.NoError(t, err)
			require.NotNil(t, txSig)
			fmt.Println("txSig", txSig)

			// wait for transaction to be confirmed
			status, err := client.WaitForTransactionConfirmed(ctx, txSig, time.Minute)
			require.NoError(t, err)
			require.EqualValues(t, solana.TransactionStatusSuccess, status)

			// check wallet1 balance of token
			_, err = client.GetTokenBalance(ctx, wallet1.PublicKey.ToBase58(), mint.PublicKey.ToBase58())
			require.Error(t, err)
		})

		t.Run("wallet2", func(t *testing.T) {
			balance, err := client.GetTokenBalance(ctx, wallet2.PublicKey.ToBase58(), mint.PublicKey.ToBase58())
			require.NoError(t, err)
			require.Greater(t, balance.Amount, uint64(0))

			// build transaction
			tx, err := solana.NewTransactionBuilder(client).
				SetFeePayer(wallet2.PublicKey.ToBase58()).
				AddInstruction(solana.BurnToken(solana.BurnTokenParams{
					Mint:              mint.PublicKey.ToBase58(),
					TokenAccountOwner: wallet2.PublicKey.ToBase58(),
					Amount:            balance.Amount,
				})).
				AddInstruction(solana.CloseTokenAccount(solana.CloseTokenAccountParams{
					Owner: wallet2.PublicKey.ToBase58(),
					Mint:  utils.Pointer(mint.PublicKey.ToBase58()),
				})).
				Build(ctx)
			require.NoError(t, err)
			require.NotNil(t, tx)

			// sign transaction
			tx, err = solana.SignTransaction(tx, wallet2)
			require.NoError(t, err)
			require.NotNil(t, tx)

			// send transaction
			txSig, err := client.SendTransaction(ctx, tx)
			if err != nil {
				// retry if transaction failed
				txSig, err = client.SendTransaction(ctx, tx)
			}
			require.NoError(t, err)
			require.NotNil(t, txSig)
			fmt.Println("txSig", txSig)

			// wait for transaction to be confirmed
			status, err := client.WaitForTransactionConfirmed(ctx, txSig, time.Minute)
			require.NoError(t, err)
			require.EqualValues(t, solana.TransactionStatusSuccess, status)

			// check wallet balance of token
			_, err = client.GetTokenBalance(ctx, wallet2.PublicKey.ToBase58(), mint.PublicKey.ToBase58())
			require.Error(t, err)
		})
	})
}

func TestCheckTokenTransferTransaction(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := solana.NewClient(solana.WithRPCEndpoint(solanaRPCEndpoint))

	var (
		mint        = "5qJZxvjTdsfY17hwCmwhfucAjUi9zcUKc7Mrw8Mz2HRy"
		destination = wallet1.PublicKey.ToBase58()
		amount      = uint64(10)
		txSig       = "5MEk2T9mVpoQNFbp7PopzAP66rUkpVR6FCviJtnnt6QK89tiMWdrzLt6y3mydTHTq2eXp7wACemUCs8cWSqByWbY"
	)

	tx, err := client.GetTransaction(ctx, txSig)
	require.NoError(t, err)
	require.NotNil(t, tx)

	// utils.PrettyPrint(tx)

	t.Run("check token transfer transaction", func(t *testing.T) {
		err = solana.CheckTokenTransferTransaction(tx.Meta, tx.Transaction, mint, destination, amount)
		require.NoError(t, err)
	})

	t.Run("check token transfer transaction with wrong mint", func(t *testing.T) {
		err = solana.CheckTokenTransferTransaction(tx.Meta, tx.Transaction, "wrong mint", destination, amount)
		require.Error(t, err)
	})

	t.Run("check token transfer transaction with wrong destination", func(t *testing.T) {
		err = solana.CheckTokenTransferTransaction(tx.Meta, tx.Transaction, mint, "wrong destination", amount)
		require.Error(t, err)
	})

	t.Run("check token transfer transaction with wrong amount", func(t *testing.T) {
		err = solana.CheckTokenTransferTransaction(tx.Meta, tx.Transaction, mint, destination, 100)
		require.Error(t, err)
	})
}

func TestCheckSolTransferTransaction(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := solana.NewClient(solana.WithRPCEndpoint(solanaRPCEndpoint))

	var (
		destination = wallet2.PublicKey.ToBase58()
		amount      = uint64(2500000)
		txSig       = "3RXc9e1d3gwYR1ho7WAqwoV9uieuABAvmG5KPHcsoGtXgsCVamZrdd1ky1AmeKUBhscrHAt4hKWN6BoAdfq1dTdM"
	)

	tx, err := client.GetTransaction(ctx, txSig)
	require.NoError(t, err)
	require.NotNil(t, tx)

	// utils.PrettyPrint(tx)

	t.Run("check sol transfer transaction", func(t *testing.T) {
		err = solana.CheckSolTransferTransaction(tx.Meta, tx.Transaction, destination, amount)
		require.NoError(t, err)
	})

	t.Run("check sol transfer transaction with wrong destination", func(t *testing.T) {
		err = solana.CheckSolTransferTransaction(tx.Meta, tx.Transaction, "wrong destination", amount)
		require.Error(t, err)
	})

	t.Run("check sol transfer transaction with wrong amount", func(t *testing.T) {
		err = solana.CheckSolTransferTransaction(tx.Meta, tx.Transaction, destination, 100)
		require.Error(t, err)
	})
}
