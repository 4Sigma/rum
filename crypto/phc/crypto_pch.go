package phc

import (
	"crypto/rand"
	"math"
	"strings"
	"unicode"
)

type cryptoPHCBackendName string

const (
	Argon2Id cryptoPHCBackendName = "argon2id"
)

type cryptoPHCBackend interface {
	GenerateFromString(password string) (string, error)
	GenerateFromBytes(secret []byte) (string, error)

	CheckSecret(encodedHash string, secret []byte) (bool, error)
	CheckPassword(encodedHash, password string) (bool, error)
}

type CryptoPHC struct {
	backend cryptoPHCBackend
}

func GetDefault() *CryptoPHC {
	return &CryptoPHC{
		backend: newArgon2PHCDefault(),
	}
}

func GetByAlgoName(backend cryptoPHCBackendName) *CryptoPHC {
	switch backend {
	case Argon2Id:
		return &CryptoPHC{
			backend: newArgon2PHCDefault(),
		}
	default:
		return nil
	}
}

func (c *CryptoPHC) GenerateFromString(password string) (string, error) {
	return c.backend.GenerateFromString(password)
}

func (c *CryptoPHC) GenerateFromBytes(secret []byte) (string, error) {
	return c.backend.GenerateFromBytes(secret)
}

func (c *CryptoPHC) CheckSecret(encodedHash string, secret []byte) (bool, error) {
	vals := cryptoPHCBackendName(strings.Split(encodedHash, "$")[0])

	switch vals {
	case Argon2Id:
		return c.backend.CheckSecret(encodedHash, secret)
	default:
		return false, nil
	}
}
func (c *CryptoPHC) CheckPassword(encodedHash, password string) (bool, error) {
	return c.backend.CheckPassword(encodedHash, password)
}

func EstimateEntropy(password string) float64 {
	if len(password) == 0 {
		return 0
	}

	var hasLower, hasUpper, hasDigit, hasSymbol bool

	for _, c := range password {
		switch {
		case unicode.IsLower(c):
			hasLower = true
		case unicode.IsUpper(c):
			hasUpper = true
		case unicode.IsDigit(c):
			hasDigit = true
		default:
			hasSymbol = true
		}
	}

	var charset float64
	if hasLower {
		charset += 26
	}
	if hasUpper {
		charset += 26
	}
	if hasDigit {
		charset += 10
	}
	if hasSymbol {
		charset += 32
	}

	if charset == 0 {
		return 0
	}

	return float64(len(password)) * math.Log2(charset)
}

func generateRandomBytes(n uint32) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}

	return b, nil
}
