package service

import (
	"crypto/rand"
	"math/big"
)

const base62 = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

type RandomCodeGenerator struct{}

func NewRandomCodeGenerator() *RandomCodeGenerator {
	return &RandomCodeGenerator{}
}

func (r *RandomCodeGenerator) newCode(n int) (string, error) {
	if n <= 0 {
		return "", nil
	}

	out := make([]byte, n)
	maxLen := big.NewInt(int64(len(base62)))

	for i := 0; i < n; i++ {
		x, err := rand.Int(rand.Reader, maxLen)
		if err != nil {
			return "", err
		}
		out[i] = base62[x.Int64()]
	}
	return string(out), nil
}
