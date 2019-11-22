package wallet

import (
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"sync"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil/hdkeychain"
	bip39 "github.com/tyler-smith/go-bip39"
)

type Wallet struct {
	Name      string                 //用户名
	HexEntory string                 //用户商
	Meta      map[string]interface{} //用户其它信息

	masterKey *hdkeychain.ExtendedKey   //HD root key
	paths     map[string]DerivationPath //HD path (address -> path)
	addresses map[uint32]string         //HD address (type -> address)
	sync.RWMutex
}

func NewWallet(name string, hexEntory string, meta map[string]interface{}) (*Wallet, error) {
	wallet := &Wallet{
		Name:      name,
		HexEntory: hexEntory,
		Meta:      meta,
		paths:     make(map[string]DerivationPath),
		addresses: make(map[uint32]string),
	}
	//商
	entropy, err := hex.DecodeString(hexEntory)
	if err != nil {
		return nil, err
	}
	//助记词
	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return nil, err
	}
	//种子
	seed := bip39.NewSeed(mnemonic, "")
	//根 key
	masterKey, err := hdkeychain.NewMaster(seed, &chaincfg.MainNetParams)
	if err != nil {
		return nil, err
	}
	wallet.masterKey = masterKey
	return wallet, nil
}

// DerivePrivateKey derives the private key of the derivation path.
func (wallet *Wallet) DerivePrivateKey(path DerivationPath) (*ecdsa.PrivateKey, error) {
	var err error
	key := wallet.masterKey
	for _, n := range path {
		key, err = key.Child(n)
		if err != nil {
			return nil, err
		}
	}

	privateKey, err := key.ECPrivKey()
	if err != nil {
		return nil, err
	}
	privateKeyECDSA := privateKey.ToECDSA()
	return privateKeyECDSA, nil
}

// DerivePublicKey derives the public key of the derivation path.
func (wallet *Wallet) DerivePublicKey(path DerivationPath) (*ecdsa.PublicKey, error) {
	privateKeyECDSA, err := wallet.DerivePrivateKey(path)
	if err != nil {
		return nil, err
	}

	publicKey := privateKeyECDSA.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("failed to get public key")
	}
	return publicKeyECDSA, nil
}
