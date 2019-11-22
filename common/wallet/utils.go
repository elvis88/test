package wallet

import (
	"encoding/hex"

	bip39 "github.com/tyler-smith/go-bip39"
)

func NewHexEntropy() string {
	entropy, err := bip39.NewEntropy(128)
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(entropy)
}
