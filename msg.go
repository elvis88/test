package main

const (
	codeOk = iota
	codeRequest
	codeAuthorize
	codePhoneValidate
	codeWallet
	codeDB
	codeRPC
	codeSMS
	codeSMSValidate
	codeSMSExpire
	codeAddrValidate
	codeOrder
	codeHash
)

var msgs = []string{
	"ok",
	"incorrect request parameters",
	"unauthorized",
	"invalidate phone or mail",
	"wallet err has occured",
	"db err has occured",
	"rpc err has occured",
	"send sms code failed",
	"invalidate sms code",
	"expire sms code",
	"invalidate address",
	"order empty",
	"hash empty",
}
