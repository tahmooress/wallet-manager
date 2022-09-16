package ulid

import (
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"strconv"
	"time"

	"github.com/oklog/ulid"
)

var ErrCannotGenerateUlid = errors.New("can't generate int of ulid")

// Generate for generate ulid
// for more data about ulid you can read original documentation https://github.com/ulid/spec
//nolint: gosec
func Generate() string {
	entropy := ulid.Monotonic(rand.New(rand.NewSource(time.Now().UnixNano())), 0)

	return ulid.MustNew(ulid.Timestamp(time.Now()), entropy).String()
}

// ToInt64 for convert ulid to int64. ulid is string ulid and
// switchNumber is one custom number get in entry and add in the first number
// convert to int64 first convert to base36 and after convert to 12 number of string and add switchNumber in the first
// final number.
func ToInt64(ulID, switchNumber string) (int64, error) {
	const (
		base         = 36
		baseStandard = 10
		lenSize      = 12
		bitSize      = 64
	)

	bigNum, ok := new(big.Int).SetString(ulID, base)
	if !ok {
		return 0, ErrCannotGenerateUlid
	}

	i, err := strconv.ParseInt(switchNumber+bigNum.String()[len(bigNum.String())-lenSize:], baseStandard, bitSize)
	if err != nil {
		return 0, fmt.Errorf("ToInt64 package >> strconv.ParseInt >> %w", err)
	}

	return i, nil
}
