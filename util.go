package main

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/erick785/services/common/wallet"
	"github.com/erick785/uranus/common/crypto"
	"github.com/erick785/uranus/common/rlp"
	"github.com/erick785/uranus/common/utils"
	"github.com/erick785/uranus/core/types"
)

var (
	COINTYPE uint32 = 60
)

// ToAddress
func ToAddress(pubKey *ecdsa.PublicKey) string {
	return crypto.PubkeyToAddress(*pubKey).String()
}

// ValidAddress
func ValidAddress(addr string) bool {
	return utils.IsHexAddr(addr)
}

func ParseDerivationPath(coinType uint32) wallet.DerivationPath {
	path, err := wallet.ParseDerivationPath(fmt.Sprintf("m/44'/%d'/0'/0/0", coinType))
	if err != nil {
		panic(err)
	}
	return path
}

func CreateTx(privKey *ecdsa.PrivateKey, nonce uint64, to string, value *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte) (string, error) {
	tto := utils.HexToAddress(to)
	tx := types.NewTransaction(types.Binary, nonce, value, gasLimit, gasPrice, data, []*utils.Address{&tto}...)
	if err := tx.SignTx(types.Signer{}, privKey); err != nil {
		return "", err
	}

	txb, err := rlp.EncodeToBytes(tx)
	if err != nil {
		return "", err
	}

	return utils.BytesToHex(txb), nil
}
