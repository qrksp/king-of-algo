package client

import (
	"context"
	"encoding/base64"
	"time"

	"github.com/algorand/go-algorand-sdk/v2/client/v2/algod"
	"github.com/algorand/go-algorand-sdk/v2/client/v2/common/models"
	"github.com/algorand/go-algorand-sdk/v2/crypto"
	"github.com/algorand/go-algorand-sdk/v2/types"
	"github.com/pkg/errors"
)

func ReadGlobalState(ctx context.Context, client *algod.Client, address string, appID uint64) ([]models.TealKeyValue, error) {
	info, err := client.AccountApplicationInformation(address, appID).Do(ctx)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return info.CreatedApp.GlobalState, nil
}

type State struct {
	Admin            string
	EndOfReign       time.Time
	KingPrice        uint64
	RewardMultiplier uint64
	ReignPeriod      uint64
	InitPrice        uint64
	King             string
	AdminFee         uint64
}

func FormatState(rawState []models.TealKeyValue) (State, error) {
	state := State{}
	for _, keyValue := range rawState {
		key, err := base64.StdEncoding.DecodeString(keyValue.Key)
		if err != nil {
			return State{}, errors.WithStack(err)
		}

		switch string(key) {
		case "king":
			adr, err := decodeAddress(keyValue.Value.Bytes)
			if err != nil {
				return State{}, errors.WithStack(err)
			}

			state.King = adr
		case "admin":
			adr, err := decodeAddress(keyValue.Value.Bytes)
			if err != nil {
				return State{}, errors.WithStack(err)
			}

			state.Admin = adr

		case "end_of_reign_timestamp":
			state.EndOfReign = time.Unix(int64(keyValue.Value.Uint), 0)

		case "king_price":
			state.KingPrice = keyValue.Value.Uint

		case "reward_multiplier":
			state.RewardMultiplier = keyValue.Value.Uint

		case "reign_period":
			state.ReignPeriod = keyValue.Value.Uint

		case "init_price":
			state.InitPrice = keyValue.Value.Uint

		case "admin_fee":
			state.AdminFee = keyValue.Value.Uint
		}
	}

	return state, nil
}

func decodeAddress(b string) (string, error) {
	if len(b) == 0 {
		return "", nil
	}

	valueBytes, err := base64.StdEncoding.DecodeString(b)
	if err != nil {
		return "", errors.WithStack(err)
	}

	address, err := types.EncodeAddress(valueBytes)
	if err != nil {
		return "", errors.WithStack(err)
	}

	return address, nil
}

func GetContractState(ctx context.Context, client *algod.Client, owner crypto.Account, appID uint64) (State, error) {
	state, err := ReadGlobalState(ctx, client, owner.Address.String(), appID)
	if err != nil {
		return State{}, errors.WithStack(err)
	}

	formattedState, err := FormatState(state)
	if err != nil {
		return State{}, errors.WithStack(err)
	}

	return formattedState, nil
}
