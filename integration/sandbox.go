package integration

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/algorand/go-algorand-sdk/v2/crypto"
	"github.com/algorand/go-algorand-sdk/v2/mnemonic"
	"github.com/pkg/errors"
)

type sandbox struct {
	sandboxPath string
}

func (s *sandbox) start() error {
	cmd := exec.Command("./sandbox", "up", "-v")
	cmd.Dir = s.sandboxPath

	stdout, err := cmd.Output()
	if err != nil {
		return errors.WithStack(errors.Errorf("%v, %s", err, string(stdout)))
	}

	return nil
}

var accountRegex = regexp.MustCompile("([A-Z0-9]){58}")

func (s *sandbox) getAccounts() ([]crypto.Account, error) {
	cmd := exec.Command("./sandbox", "goal", "account", "list")

	cmd.Dir = s.sandboxPath

	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	accountAddresses := accountRegex.FindAllString(out.String(), -1)

	result := []crypto.Account{}
	done := map[string]interface{}{}
	for _, addr := range accountAddresses {
		if _, ok := done[addr]; ok {
			continue
		}
		done[addr] = nil

		mnemonicWords, err := s.getMnemonicOfAccount(addr)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		key, err := mnemonic.ToPrivateKey(mnemonicWords)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		acc, err := crypto.AccountFromPrivateKey(key)
		if err != nil {
			return nil, err
		}

		result = append(result, acc)
	}

	return result, nil
}

var mnemonicWordsRegex = regexp.MustCompile(`"(.*?)"`)

func (s *sandbox) getMnemonicOfAccount(address string) (string, error) {
	command := fmt.Sprintf("./sandbox goal account export -a %s", address)
	cmdParts := strings.Split(command, " ")
	cmd := exec.Command(cmdParts[0], cmdParts[1:]...)

	cmd.Dir = s.sandboxPath

	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		return "", errors.WithStack(err)
	}

	matches := mnemonicWordsRegex.FindAllStringSubmatch(out.String(), 1)

	if len(matches) == 0 || len(matches[0]) != 2 {
		return "", errors.New("no matches")
	}

	return matches[0][1], nil
}
