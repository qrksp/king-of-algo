package client

import (
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"time"

	"github.com/pkg/errors"
)

func openFile(fileName string) ([]byte, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	defer f.Close()

	fileBytes, err := io.ReadAll(f)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return fileBytes, nil
}

func saveAppID(appID uint64) error {
	err := os.WriteFile(
		fmt.Sprintf("new-app-%s", time.Now()),
		[]byte(fmt.Sprintf("{\"AppID\": %d}", appID)),
		fs.ModePerm,
	)

	return errors.WithStack(err)
}

func multiplyPercentage(amount uint64, percentage uint64) uint64 {
	return uint64(math.Ceil(float64(amount) * float64(percentage) / 100))
}
