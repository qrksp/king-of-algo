package client

import (
	"context"
	"encoding/binary"
	"path/filepath"
	"time"

	"github.com/algorand/go-algorand-sdk/v2/client/v2/algod"
	"github.com/algorand/go-algorand-sdk/v2/crypto"
	"github.com/algorand/go-algorand-sdk/v2/transaction"
	"github.com/algorand/go-algorand-sdk/v2/types"
	"github.com/pkg/errors"
)

func Deploy(ctx context.Context, algodClient *algod.Client, account crypto.Account, reignPeriod time.Duration, creationNote string) (uint64, error) {
	globalInts := 6  // The prices, timestamp, period, admin fee and reward multiplier.
	globalBytes := 2 // current king address and admin address
	localInts := 0
	localBytes := 0

	gSchema := types.StateSchema{NumUint: uint64(globalInts), NumByteSlice: uint64(globalBytes)}
	lSchema := types.StateSchema{NumUint: uint64(localInts), NumByteSlice: uint64(localBytes)}

	approvalProgram, err := openFile(filepath.Join("..", "contracts", "approval.teal"))
	if err != nil {
		return 0, err
	}

	compiledApprovalProgram, err := compileProgram(ctx, algodClient, approvalProgram)
	if err != nil {
		return 0, err
	}

	clearProgram, err := openFile(filepath.Join("..", "contracts", "clear.teal"))
	if err != nil {
		return 0, err
	}

	compiledClearProgram, err := compileProgram(ctx, algodClient, clearProgram)
	if err != nil {
		return 0, err
	}

	// Parse unix timestamp to bytes.
	reignPeriodArg := make([]byte, 8)
	binary.BigEndian.PutUint64(reignPeriodArg, uint64(reignPeriod.Seconds()))

	waitRounds := uint64(5)
	suggestedParams, err := algodClient.SuggestedParams().Do(context.Background())
	if err != nil {
		return 0, errors.WithStack(err)
	}

	signedBytes, err := makeCreateAppTx(
		ctx,
		algodClient,
		suggestedParams,
		account,
		compiledApprovalProgram,
		compiledClearProgram,
		gSchema,
		lSchema,
		[][]byte{reignPeriodArg},
		[]byte(creationNote),
	)
	if err != nil {
		return 0, err
	}

	resp, err := sendWaitTransaction(ctx, algodClient, signedBytes, waitRounds)
	if err != nil {
		return 0, err
	}

	appID := resp.ApplicationIndex

	// Send minimum balance to app account 100000 0.1 ALGO.
	// If we don't do this then the init payment to this address has to be > 0.1 ALGO. Which limits the init king's price.
	err = sendInitBalance(ctx, algodClient, account, crypto.GetApplicationAddress(appID), 100000, waitRounds)
	if err != nil {
		return 0, err
	}

	saveAppID(appID)

	return appID, nil
}

func makeCreateAppTx(
	_ context.Context,
	_ *algod.Client,
	suggestedParams types.SuggestedParams,
	sender crypto.Account,
	approvalProgram []byte,
	clearProgram []byte,
	globalSchema types.StateSchema,
	localSchema types.StateSchema,
	appArgs [][]byte,
	note []byte,
) ([]byte, error) {
	tx, err := transaction.MakeApplicationCreateTx(
		false, // no-op
		approvalProgram,
		clearProgram,
		globalSchema,
		localSchema,
		appArgs,
		nil,
		nil,
		nil,
		suggestedParams,
		sender.Address,
		note,
		types.Digest{},
		[32]byte{},
		types.Address{},
	)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	_, signedBytes, err := crypto.SignTransaction(sender.PrivateKey, tx)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return signedBytes, nil
}

func sendInitBalance(ctx context.Context, client *algod.Client, sender crypto.Account, receiver types.Address, amount uint64, waitRounds uint64) error {
	suggestedParams, err := client.SuggestedParams().Do(context.Background())
	if err != nil {
		return errors.WithStack(err)
	}

	tx, err := transaction.MakePaymentTxn(sender.Address.String(), receiver.String(), amount, nil, "", suggestedParams)
	if err != nil {
		return errors.WithStack(err)
	}

	_, signedBytes, err := crypto.SignTransaction(sender.PrivateKey, tx)
	if err != nil {
		return errors.WithStack(err)
	}

	_, err = sendWaitTransaction(ctx, client, signedBytes, waitRounds)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}
