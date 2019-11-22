package sms

import (
	"testing"
)

func TestSMS(t *testing.T) {
	number := "342529499@qq.com"
	code := MakeCode()
	SendCode(number, code)

	// fmt.Println(VailMobile(number))

	// for i := 0; i < 100; i++ {
	// 	code := MakeCode()
	// 	fmt.Println(code, VailCode(code) == nil)
	// }
}
