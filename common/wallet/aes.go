package wallet

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha1"
	"errors"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

// Encrypt 加密
func Encrypt(plaintext []byte, key []byte) []byte {
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}

	// The IV needs to be unique, but not secure. Therefore it's common to
	// include it at the beginning of the ciphertext.
	//

	// {IV} + {plaintext len}
	// ciphertext := make([]byte, aes.BlockSize+len(plaintext)+10)
	ciphertext := make([]byte, len(plaintext))

	// create iv from random stream
	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		panic(err)
	}
	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(ciphertext, plaintext)

	result := append(iv, ciphertext...)
	hmac := pbkdf2.Key([]byte(result), key, 1000, 16, sha1.New)
	result = append(result, hmac...)
	return result
}

// Decrypt 解密
func Decrypt(message []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return []byte{}, err
	}

	// The first 16 bytes are IV
	iv := message[0:16]
	em := message[16 : len(message)-16]
	hmac := message[len(message)-16:]

	hmac2 := pbkdf2.Key(message[:len(message)-16], key, 1000, 16, sha1.New)

	if bytes.Compare(hmac, hmac2) != 0 {
		return []byte{}, errors.New("Unmatched hmac")
	}
	plaintext := make([]byte, len(em))
	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(plaintext, em)
	return plaintext, nil
}

// RightPadBytes zero-pads slice to the right up to length l.
func RightPadBytes(slice []byte, l int) []byte {
	if l <= len(slice) {
		return slice[:l]
	}

	padded := make([]byte, l)
	copy(padded, slice)

	return padded
}
