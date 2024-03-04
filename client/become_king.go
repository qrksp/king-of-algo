package client

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/algorand/go-algorand-sdk/v2/client/v2/algod"
	"github.com/algorand/go-algorand-sdk/v2/client/v2/common/models"
	"github.com/algorand/go-algorand-sdk/v2/crypto"
	"github.com/algorand/go-algorand-sdk/v2/encoding/msgpack"
	"github.com/algorand/go-algorand-sdk/v2/transaction"
	"github.com/algorand/go-algorand-sdk/v2/types"
	"github.com/pkg/errors"
)

const noteFormat = "kingOfAlgo/v1:u%s"

func BecomeKing(
	ctx context.Context,
	client *algod.Client,
	debug bool,
	params BecomeKingParams,
	waitRounds uint64,
) (models.PendingTransactionInfoResponse, error) {
	signedTxnBytes, signedGroup, err := MakeBecomeKingTx(params)
	if err != nil {
		return models.PendingTransactionInfoResponse{}, errors.WithStack(err)
	}

	if debug {
		err = debugTxn(ctx, client, signedGroup)
		if err != nil {
			return models.PendingTransactionInfoResponse{}, errors.WithStack(err)
		}
	}

	return sendWaitTransaction(ctx, client, signedTxnBytes, waitRounds)
}

// MakeBecomeKingTx creates the signed transactions to become king.
func MakeBecomeKingTx(params BecomeKingParams) ([]byte, []types.SignedTxn, error) {
	suggestedParams := params.txParams
	// We need to give more fee for the inner tx to pay the previous king.
	// When you do flat fee you can put whatever fee you want and in this case because we have one inner tx inside the contract
	accounts := []string{}
	if params.isReignEnded() && params.state.King != "" {
		accounts = append(accounts, params.state.King)

		// I could add all the fees in the first tx of the group (1000 * 5) but no, we put 2 * min in the appTx for the innertx and the suggested to the compTx, rewardTx and the adminFeeTx
		suggestedParams.FlatFee = true
		suggestedParams.Fee = transaction.MinTxnFee * 2
	}

	// NOOP TX.
	noOpTx, err := transaction.MakeApplicationNoOpTx(
		params.appIndex,
		[][]byte{[]byte("claim_throne")},
		accounts,
		nil,
		nil,
		suggestedParams,
		params.sender.Address,
		[]byte(fmt.Sprintf(noteFormat, params.message)),
		types.Digest{},
		[32]byte{},
		types.ZeroAddress,
	)
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}

	transactions := []types.Transaction{}
	transactions = append(transactions, noOpTx)

	adminFee := multiplyPercentage(params.getPayAmount(), params.state.AdminFee)

	// Payment of fee to contract admin.
	adminFeeTx, err := transaction.MakePaymentTxn(
		params.sender.Address.String(),
		params.state.Admin,
		adminFee,
		[]byte(fmt.Sprintf(noteFormat, "admin_fee_tx")),
		"",
		params.txParams)
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}

	transactions = append(transactions, adminFeeTx)

	reward := multiplyPercentage(params.getPayAmount(), params.state.RewardMultiplier)
	if !params.isKingSet() {
		reward = 0
	}

	// Compensation amount to the contracts address.
	comp := params.getPayAmount() - adminFee - reward

	compTx, err := transaction.MakePaymentTxn(
		params.sender.Address.String(),
		crypto.GetApplicationAddress(params.appIndex).String(),
		comp,
		[]byte(fmt.Sprintf(noteFormat, "comp_tx")),
		"",
		params.txParams)
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}

	transactions = append(transactions, compTx)

	// If there is no previous king we omit this tx.
	if params.isKingSet() {
		rewardTx, err := transaction.MakePaymentTxn(
			params.sender.Address.String(),
			params.state.King,
			reward,
			[]byte(fmt.Sprintf(noteFormat, "reward_tx")),
			"",
			params.txParams)
		if err != nil {
			return nil, nil, errors.WithStack(err)
		}

		transactions = append(transactions, rewardTx)
	}

	groupedTxs, err := transaction.AssignGroupID(transactions, "")
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}

	signedGrpBytes := []byte{}
	signedGroup := []types.SignedTxn{}
	signedTx := types.SignedTxn{}
	for _, tx := range groupedTxs {
		_, signedBytes, err := crypto.SignTransaction(params.sender.PrivateKey, tx)
		if err != nil {
			return nil, nil, errors.WithStack(err)
		}

		signedGrpBytes = append(signedGrpBytes, signedBytes...)

		err = msgpack.Decode(signedBytes, &signedTx)
		if err != nil {
			return nil, nil, errors.WithStack(err)
		}

		signedGroup = append(signedGroup, signedTx)
	}

	return signedGrpBytes, signedGroup, nil
}

type BecomeKingParams struct {
	txParams types.SuggestedParams
	state    State
	sender   crypto.Account
	message  string
	appIndex uint64
}

func NewBecomeKingParams(txParams types.SuggestedParams, appIndex uint64, state State, sender crypto.Account, message string) BecomeKingParams {
	return BecomeKingParams{
		txParams: txParams,
		state:    state,
		sender:   sender,
		message:  message,
		appIndex: appIndex,
	}
}

func (p BecomeKingParams) isReignEnded() bool {
	return p.state.EndOfReign.Before(time.Now())
}

func (p BecomeKingParams) getPayAmount() uint64 {
	if p.isReignEnded() {
		return p.state.InitPrice
	}

	return p.state.KingPrice
}

func (p BecomeKingParams) isKingSet() bool {
	return p.state.King != ""
}

func sendWaitTransaction(ctx context.Context, client *algod.Client, rawTxn []byte, waitRounds uint64) (models.PendingTransactionInfoResponse, error) {
	txID, err := client.SendRawTransaction(rawTxn).Do(ctx)
	if err != nil {
		return models.PendingTransactionInfoResponse{}, errors.WithStack(err)
	}

	confirmedTxn, err := transaction.WaitForConfirmation(client, txID, waitRounds, ctx)
	if err != nil {
		return models.PendingTransactionInfoResponse{}, errors.WithStack(err)
	}

	return confirmedTxn, nil
}

func debugTxn(ctx context.Context, client *algod.Client, txns []types.SignedTxn) error {
	drr, err := transaction.CreateDryrun(
		client,
		txns,
		&models.DryrunRequest{
			LatestTimestamp: uint64(time.Now().Unix()),
		},
		ctx,
	)
	if err != nil {
		return errors.WithStack(err)
	}

	path := "dryruns"
	// ignore the error
	err = os.Mkdir(path, os.ModePerm)
	if err != nil && !os.IsExist(err) {
		return errors.WithStack(err)
	}

	filename := fmt.Sprintf("dryruns/dryrun-%s.msgp", time.Now().UTC())
	err = os.WriteFile(filename, msgpack.Encode(drr), 0666)
	if err != nil {
		return errors.WithStack(err)
	}

	res, err := client.TealDryrun(drr).Do(ctx)
	if err != nil {
		return errors.WithStack(err)
	}

	drresp, err := transaction.NewDryrunResponse(res)
	if err != nil {
		return errors.WithStack(err)
	}

	for idx, txResult := range drresp.Txns {
		if txResult.AppCallRejected() {
			fmt.Printf("Failed app call in %d:\n%s", idx, txResult.GetAppCallTrace(transaction.DefaultStackPrinterConfig()))
		}
	}

	return nil
}
