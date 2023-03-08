package solana_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/easypmnt/checkout-api/solana"
	"github.com/portto/solana-go-sdk/types"
	"github.com/stretchr/testify/require"
)

var (
	solanaRPCEndpoint = "https://api.devnet.solana.com"
	solanaWSSEndpoint = "wss://api.devnet.solana.com"

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
		txResp, err := client.GetOldestTransactionForWallet(ctx, referenceAcc.PublicKey.ToBase58(), "")
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