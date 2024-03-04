package integration

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"math"
	"os"
	"strings"
	"sync"

	"github.com/algorand/go-algorand-sdk/v2/client/v2/algod"
	"github.com/algorand/go-algorand-sdk/v2/client/v2/common"
	"github.com/algorand/go-algorand-sdk/v2/client/v2/common/models"
	"github.com/algorand/go-algorand-sdk/v2/crypto"
	"github.com/algorand/go-algorand-sdk/v2/mnemonic"
	"github.com/algorand/go-algorand-sdk/v2/types"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

type suite struct {
	Algod    *algod.Client
	Accounts []crypto.Account
}

func NewSuite() *suite {
	cfg, err := newConfig()
	if err != nil {
		panic(err)
	}

	algodClient, err := algod.MakeClientWithHeaders(cfg.AlgodEndpoint, cfg.AlgodToken, []*common.Header{
		{
			Key:   "x-api-key",
			Value: cfg.AlgodToken,
		},
	})
	if err != nil {
		panic(err)
	}

	status, err := algodClient.Status().Do(context.Background())
	if err != nil {
		panic(err)
	}

	if status.LastRound == 0 {
		fmt.Println("SANDBOX OFF")
		// err := sandbox.start()
		// if err != nil {
		// 	panic(err)
		// }
	}

	// I don't want to commit my home dir.
	dirname, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	sandboxPath := strings.Replace(cfg.SandboxRepoPath, "$HOME", dirname, -1)

	sandbox := &sandbox{
		sandboxPath: sandboxPath,
	}

	accounts, err := sandbox.getAccounts()
	if err != nil {
		fmt.Printf("%+v", err)
		os.Exit(1)
	}

	err = saveTestAccounts(accounts)
	if err != nil {
		fmt.Printf("%+v", err)
		os.Exit(1)
	}

	return &suite{
		Algod:    algodClient,
		Accounts: accounts,
	}
}

func saveTestAccounts(accounts []crypto.Account) error {
	buf := bytes.NewBuffer(nil)
	for i, acc := range accounts {
		words, err := mnemonic.FromPrivateKey(acc.PrivateKey)
		if err != nil {
			return errors.WithStack(err)
		}

		buf.WriteString(fmt.Sprintf("Account %d: %s", i, words))
	}

	err := os.WriteFile(
		"latest-generated-accounts",
		buf.Bytes(),
		fs.ModePerm,
	)

	return errors.WithStack(err)
}

func (s *suite) getAccountsBalances() map[string]uint64 {
	g := new(errgroup.Group)
	result := map[string]uint64{}
	mutex := &sync.RWMutex{}

	for _, acc := range s.Accounts {
		account := acc
		g.Go(func() error {
			info, err := s.Algod.AccountInformation(account.Address.String()).Do(context.Background())
			if err != nil {
				return err
			}

			mutex.Lock()
			result[account.Address.String()] = info.Amount
			mutex.Unlock()

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		panic(err)
	}

	return result
}

func (s *suite) getContractAccountInfo(appID uint64) models.Account {
	account := crypto.GetApplicationAddress(appID)

	// This still doesn't return the min-balance https://github.com/algorand/go-algorand-sdk/issues/272
	info, err := s.Algod.AccountInformation(account.String()).Do(context.Background())
	if err != nil {
		panic(err)
	}

	return info
}

// TODO: remove this after its added to account info.
func (s *suite) minBalance() uint64 {
	return 100000
}

func (s *suite) getSuggestedParams() types.SuggestedParams {
	suggestedParams, err := s.Algod.SuggestedParams().Do(context.Background())
	if err != nil {
		panic(err)
	}

	return suggestedParams
}
func multiplyPercentage(amount uint64, percentage uint64) uint64 {
	return uint64(math.Ceil(float64(amount) * float64(percentage) / 100))
}
