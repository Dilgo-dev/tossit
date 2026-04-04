package crypto

import (
	"errors"

	"github.com/schollz/pake/v3"
	"golang.org/x/crypto/argon2"
)

const keyLen = 32

var kdfSalt = []byte("tossit-v1-key-derivation")

func SenderKeyExchange(sendMsg func([]byte) error, recvMsg func() ([]byte, error), password string) ([]byte, error) {
	p, err := pake.InitCurve([]byte(password), 0, "siec")
	if err != nil {
		return nil, err
	}

	if err := sendMsg(p.Bytes()); err != nil {
		return nil, err
	}

	incoming, err := recvMsg()
	if err != nil {
		return nil, err
	}
	if err := p.Update(incoming); err != nil {
		return nil, err
	}

	sessionKey, err := p.SessionKey()
	if err != nil {
		return nil, errors.New("key exchange failed")
	}

	return deriveKey(sessionKey), nil
}

func ReceiverKeyExchange(sendMsg func([]byte) error, recvMsg func() ([]byte, error), password string) ([]byte, error) {
	p, err := pake.InitCurve([]byte(password), 1, "siec")
	if err != nil {
		return nil, err
	}

	incoming, err := recvMsg()
	if err != nil {
		return nil, err
	}
	if err := p.Update(incoming); err != nil {
		return nil, err
	}

	if err := sendMsg(p.Bytes()); err != nil {
		return nil, err
	}

	sessionKey, err := p.SessionKey()
	if err != nil {
		return nil, errors.New("key exchange failed")
	}

	return deriveKey(sessionKey), nil
}

func DeriveKeyFromCode(code string, password string) []byte {
	input := code
	if password != "" {
		input = code + ":" + password
	}
	return argon2.IDKey([]byte(input), kdfSalt, 1, 64*1024, 4, keyLen)
}

func deriveKey(sessionKey []byte) []byte {
	return argon2.IDKey(sessionKey, kdfSalt, 1, 64*1024, 4, keyLen)
}
