package tkn

import (
	"crypto/rand"
	"errors"
	"strings"
)

const (
	defaultTokenLength = 38
	alphanumericChars  = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
)

type GenerateOption func(opts *generateOptions)

// Config holds configuration parameters for token generation and hashing.
type generateOptions struct {
	length int    // Length of the random part of the token
	prefix string // Prefix to prepend to the token
}

// WithLength sets the length of the random part of the token.
func WithLength(length int) GenerateOption {
	return func(opts *generateOptions) {
		opts.length = length
	}
}

// WithPrefix sets the prefix to prepend to the token.
func WithPrefix(prefix string) GenerateOption {
	return func(opts *generateOptions) {
		opts.prefix = prefix
	}
}

// Generate generates a secure random token.
func Generate(opts ...GenerateOption) (string, error) {
	options := generateOptions{
		length: defaultTokenLength, // 226 bits of entropy
		prefix: "",
	}
	for _, opt := range opts {
		opt(&options)
	}

	length := options.length

	var sb strings.Builder
	sb.Grow(length)
	charSetLen := len(alphanumericChars)
	for i := 0; i < length; i++ {
		idx, err := randInt(charSetLen)
		if err != nil {
			return "", err
		}
		sb.WriteByte(alphanumericChars[idx])
	}

	return options.prefix + sb.String(), nil
}

func randInt(limit int) (int, error) {
	if limit <= 0 {
		return 0, errors.New("max must be positive")
	}
	b := make([]byte, 1)
	_, err := rand.Read(b)
	if err != nil {
		return 0, err
	}
	return int(b[0]) % limit, nil
}
