package client

import (
	"context"
	"encoding/base64"

	"github.com/algorand/go-algorand-sdk/v2/client/v2/algod"
	"github.com/pkg/errors"
)

func compileProgram(ctx context.Context, client *algod.Client, sourceCode []byte) ([]byte, error) {
	compileResult, err := client.TealCompile(sourceCode).Do(ctx)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return base64.StdEncoding.DecodeString(compileResult.Result)
}
