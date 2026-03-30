package utils

import (
	"crypto/rand"
	"math/big"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

const (
	shortCodeAlphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	shortCodeLength   = 7
)

func GenerateRandomShortCode() (string, error) {
	code := make([]byte, shortCodeLength)
	alphabetSize := big.NewInt(int64(len(shortCodeAlphabet)))

	for idx := range code {
		randomIndex, err := rand.Int(rand.Reader, alphabetSize)
		if err != nil {
			return "", err
		}

		code[idx] = shortCodeAlphabet[randomIndex.Int64()]
	}

	return string(code), nil
}

func ParseExpireTime(expireTime *string) (pgtype.Timestamptz, error) {
	if expireTime == nil || *expireTime == "" {
		return pgtype.Timestamptz{Valid: false}, nil
	}

	parsedTime, err := time.Parse(time.RFC3339, *expireTime)
	if err != nil {
		return pgtype.Timestamptz{}, err
	}

	return pgtype.Timestamptz{
		Time:  parsedTime,
		Valid: true,
	}, nil
}