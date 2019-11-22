package main

import "math/big"

// Block 区块信息，
type Block struct {
	ID     string `json:"hash"`       // 区块哈希
	PrevID string `json:"parentHash"` // 前区块哈希
	Height int64  `json:"height"`     // 区块号
	Time   int64  `json:"timestamp"`  // 区块时间

	GasLimit int64  `json:"gasLimit"`
	GasUsed  int64  `json:"gasUsed"`
	Miner    string `json:"miner"`

	Transactions map[string]*Transaction `json:"-"`
	addressInfos map[string]*AddressInfo `json:"-"` //相关账户信息
}

// Transaction 交易概览
type Transaction struct {
	ID        string   // 交易哈希
	Ins       []*InOut // 输入
	Outs      []*InOut // 输出
	Height    int64    // 区块号
	Time      int64    // 交易时间
	Size      int64    // 消耗 gasused
	Signature string   // 签名
	Fee       *big.Int // 消耗 eth
}

// AddressInfo 地址信息，
type AddressInfo struct {
	Amount *big.Int                //	余额
	HTxs   []string                // 相关的交易顺序
	Txs    map[string]*Transaction // 相关的交易信息
}

// Number 区块高度
func (blk *Block) Number() *big.Int {
	return big.NewInt(blk.Height)
}

// Hash 区块哈希
func (blk *Block) Hash() string {
	return blk.ID
}

// ParentHash 前区块哈希
func (blk *Block) ParentHash() string {
	return blk.PrevID
}

// TxHash 交易哈希
func (tx *Transaction) TxHash() string {
	return tx.ID
}

// InOut 输入输出信息
type InOut struct {
	Addresses []string `json:"addresses"`
	Value     *big.Int `json:"value"`
}

// TokenInfo token信息
type TokenInfo struct {
	Address string `json:"address"`
	Name    string `json:"name"`
	Symbol  string `json:"symbol"`
	Decimal int64  `json:"decimal"`
}
