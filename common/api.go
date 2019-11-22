package common

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"math/big"

	"github.com/erick785/uranus/common/utils"
)

// var (
// 	// OKCode Ok
// 	OKCode = 2000
// 	//RequestCode Api request error
// 	RequestCode = 20001
// 	//ExecuteCode Api execute error
// 	ExecuteCode = 20002
// 	//UnauthorizedCode unauthorized
// 	UnauthorizedCode = 2003
// )

//APIRespone Api respone data
type APIRespone struct {
	Data    interface{} `json:"data"`
	ErrCode int         `json:"errCode"`
	ErrMsg  string      `json:"errMsg"`
	Hash    string      `json:"hash"`
}

//MD5 calc hash
func (api *APIRespone) MD5() string {
	bts, _ := json.Marshal(api.Data)
	r := md5.Sum(bts)
	return hex.EncodeToString(r[:])
}

//HistoryInfo 历史交易信息
type HistoryInfo struct {
	Hash          string   `json:"hash"`          // 交易哈希
	From          string   `json:"from"`          // 发起者
	To            string   `json:"to"`            // 接受者（合约地址）
	Value         *big.Int `json:"value"`         // 金额
	TValue        *big.Int `json:"tvalue"`        // 金额实际变动
	Fee           *big.Int `json:"fee"`           // 手续费
	Size          int64    `json:"size"`          // gas used
	Time          int64    `json:"time"`          // 交易时间
	Height        int64    `json:"height"`        // 区块号
	Confirmations int64    `json:"confirmations"` // 确认数
	Signature     string   `json:"signature"`     // 签名
	Status        int      `json:"status"`        // 状态码
}

// BlockInfo 区块信息
type BlockInfo struct {
	PreviousHash utils.Hash    `json:"previousHash"`
	Miner        utils.Address `json:"miner"`
	Height       *big.Int      `json:"height"`
	GasLimit     uint64        `json:"gasLimit"`
	GasUsed      uint64        `json:"gasUsed" `
	TimeStamp    *big.Int      `json:"timestamp"`
	ExtraData    []byte        `json:"extraData"`
}
