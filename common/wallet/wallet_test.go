package wallet

import (
	"encoding/hex"
	"fmt"
	"testing"

	bip39 "github.com/tyler-smith/go-bip39"
)

func TestWallet(t *testing.T) {
	mnemonic := "abandon amount liar amount expire adjust cage candy arch gather drum buyer"
	entropy, err := bip39.EntropyFromMnemonic(mnemonic)
	if err != nil {
		panic(err)
	}

	fmt.Println(hex.EncodeToString(entropy))

	w, err := NewWallet("test", hex.EncodeToString(entropy), nil)
	if err != nil {
		panic(err)
	}

	_ = w

	// 12 words
	entropy, _ = bip39.NewEntropy(128)
	fmt.Println(bip39.NewMnemonic(entropy))
	// 24 words
	entropy, _ = bip39.NewEntropy(256)
	fmt.Println(bip39.NewMnemonic(entropy))
}
