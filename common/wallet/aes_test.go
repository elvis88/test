package wallet

import (
	"fmt"
	"testing"
)

func TestAes(t *testing.T) {
	bts := Encrypt([]byte("test"), RightPadBytes([]byte("password"), 16))
	c, err := Decrypt(bts, RightPadBytes([]byte("password"), 16))
	fmt.Println(string(c), err)
}
