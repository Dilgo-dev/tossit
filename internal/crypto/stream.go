package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"errors"
	"sync"
)

const ChunkSize = 64 * 1024

type Encryptor struct {
	gcm cipher.AEAD
	seq uint32
	mu  sync.Mutex
}

func NewEncryptor(key []byte) (*Encryptor, error) {
	return NewEncryptorAt(key, 0)
}

func NewEncryptorAt(key []byte, startSeq uint32) (*Encryptor, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	return &Encryptor{gcm: gcm, seq: startSeq}, nil
}

func (e *Encryptor) EncryptChunk(plaintext []byte) (uint32, []byte, error) {
	e.mu.Lock()
	seq := e.seq
	e.seq++
	e.mu.Unlock()

	nonce := nonceFromSeq(seq, e.gcm.NonceSize())
	ciphertext := e.gcm.Seal(nil, nonce, plaintext, nil)
	return seq, ciphertext, nil
}

type Decryptor struct {
	gcm cipher.AEAD
}

func NewDecryptor(key []byte) (*Decryptor, error) {
	return NewDecryptorAt(key, 0)
}

func NewDecryptorAt(key []byte, _ uint32) (*Decryptor, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	return &Decryptor{gcm: gcm}, nil
}

func (d *Decryptor) DecryptChunk(seq uint32, ciphertext []byte) ([]byte, error) {
	nonce := nonceFromSeq(seq, d.gcm.NonceSize())
	plaintext, err := d.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, errors.New("decryption failed: invalid chunk or wrong key")
	}
	return plaintext, nil
}

func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func nonceFromSeq(seq uint32, nonceSize int) []byte {
	nonce := make([]byte, nonceSize)
	binary.BigEndian.PutUint32(nonce[nonceSize-4:], seq)
	return nonce
}
