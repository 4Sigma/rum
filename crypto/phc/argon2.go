package phc

import (
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

var (
	ErrInvalidHash         = errors.New("the encoded hash is not in the correct format")
	ErrIncompatibleVersion = errors.New("incompatible version of argon2")
)

type Argon2Config struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
	saltLength  uint32
	keyLength   uint32
}

type argon2Pch struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
	saltLength  uint32
	keyLength   uint32
}

func GetDefaultArgon2Config() *Argon2Config {
	return &Argon2Config{
		memory:      64 * 1024,
		iterations:  3,
		parallelism: 2,
		saltLength:  16,
		keyLength:   32,
	}
}

func newArgon2PHCDefault() *argon2Pch {
	return NewArgon2PHC(GetDefaultArgon2Config())
}

func NewArgon2PHC(config *Argon2Config) *argon2Pch {
	return &argon2Pch{
		memory:      config.memory,
		iterations:  config.iterations,
		parallelism: config.parallelism,
		saltLength:  config.saltLength,
		keyLength:   config.keyLength,
	}
}

func (a *argon2Pch) GenerateFromBytes(secret []byte) (encodedHash string, err error) {
	salt, err := generateRandomBytes(a.saltLength)
	if err != nil {
		return "", err
	}

	hash := argon2.IDKey(secret, salt, a.iterations, a.memory, a.parallelism, a.keyLength)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	encodedHash = fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, a.memory, a.iterations, a.parallelism, b64Salt, b64Hash,
	)

	return encodedHash, nil
}

func (a *argon2Pch) GenerateFromString(password string) (encodedHash string, err error) {
	return a.GenerateFromBytes([]byte(password))
}

func (a *argon2Pch) decodeHash(encodedHash string) (cfg *Argon2Config, salt, hash []byte, err error) {
	vals := strings.Split(encodedHash, "$")
	if len(vals) != 6 {
		return nil, nil, nil, ErrInvalidHash
	}

	var version int
	_, err = fmt.Sscanf(vals[2], "v=%d", &version)
	if err != nil {
		return nil, nil, nil, err
	}

	if version != argon2.Version {
		return nil, nil, nil, ErrIncompatibleVersion
	}

	p := Argon2Config{}
	_, err = fmt.Sscanf(vals[3], "m=%d,t=%d,p=%d", &p.memory, &p.iterations, &p.parallelism)
	if err != nil {
		return nil, nil, nil, err
	}

	salt, err = base64.RawStdEncoding.Strict().DecodeString(vals[4])
	if err != nil {
		return nil, nil, nil, err
	}
	p.saltLength = uint32(len(salt))

	hash, err = base64.RawStdEncoding.Strict().DecodeString(vals[5])
	if err != nil {
		return nil, nil, nil, err
	}
	p.keyLength = uint32(len(hash))

	return &p, salt, hash, nil
}

func (a *argon2Pch) CheckSecret(encodedHash string, password []byte) (match bool, err error) {
	p, salt, hash, err := a.decodeHash(encodedHash)
	if err != nil {
		return false, err
	}

	otherHash := argon2.IDKey([]byte(password), salt, p.iterations, p.memory, p.parallelism, p.keyLength)
	if subtle.ConstantTimeCompare(hash, otherHash) == 1 {
		return true, nil
	}

	return false, nil
}

func (a *argon2Pch) CheckPassword(encodedHash, password string) (match bool, err error) {
	return a.CheckSecret(encodedHash, []byte(password))
}
