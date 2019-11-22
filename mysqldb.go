package main

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/erick785/services/common"
)

func (mysql *Mysql) GetAmount(address string, tokenAddress string) (*big.Int, error) {
	return mysql.RPC.getBalance(address, tokenAddress, nil)
}

func (mysql *Mysql) GetNonce(address string) (*big.Int, error) {
	return mysql.RPC.getTransactionCount(address, nil)
}

func (mysql *Mysql) GetGasPrice() (*big.Int, error) {
	return mysql.RPC.GetGasPrice()
}

func (mysql *Mysql) GetHistory(address string, tokenAddress string, pagesize int64, pagenum int64) ([]*common.HistoryInfo, error) {
	key := address
	if len(tokenAddress) > 0 {
		key = fmt.Sprintf("%s-%s", address, tokenAddress)
	}
	txs, err := mysql.GetTransactionsByAddress(key, pagesize, pagenum)
	if err != nil {
		return nil, err
	}
	curBlock, _ := mysql.GetBlockChain()
	htxs := []*common.HistoryInfo{}
	for _, tx := range txs {
		htx := &common.HistoryInfo{
			Hash:      tx.ID,
			Time:      tx.Time,
			Height:    tx.Height,
			Fee:       new(big.Int).SetBytes(tx.Fee.Bytes()),
			Size:      tx.Size,
			Signature: tx.Signature,
		}
		if tx.Height > 0 {
			htx.Confirmations = curBlock.Height - tx.Height + 1
		}
		if htx.Confirmations > 6 {
			htx.Status = 1
		}
		var ins []*InOut
		ivalue := big.NewInt(0)
		for _, in := range tx.Ins {
			if len(tokenAddress) == 0 {
				if strings.Contains(strings.Join(in.Addresses, ","), "-") {
					continue
				}
			} else {
				if !strings.Contains(strings.Join(in.Addresses, ","), fmt.Sprintf("-%s", tokenAddress)) {
					continue
				}
			}
			//支出金额
			if strings.Contains(strings.Join(in.Addresses, ","), key) {
				ivalue = new(big.Int).Add(ivalue, in.Value)
			}
			ins = append(ins, in)
		}
		var outs []*InOut
		ovalue := big.NewInt(0)
		for _, out := range tx.Outs {
			if len(tokenAddress) == 0 {
				if strings.Contains(strings.Join(out.Addresses, ","), "-") {
					continue
				}
			} else {
				if !strings.Contains(strings.Join(out.Addresses, ","), fmt.Sprintf("-%s", tokenAddress)) {
					continue
				}
			}
			//收入金额
			if strings.Contains(strings.Join(out.Addresses, ","), key) {
				ovalue = new(big.Int).Add(ovalue, out.Value)
			}
			outs = append(outs, out)
		}
		htx.Value = ivalue
		htx.TValue = new(big.Int).Sub(ivalue, ovalue)
		if htx.TValue.Sign() < 0 {
			htx.TValue = new(big.Int).Abs(htx.TValue)
		}
		if len(ins) == 1 && len(outs) == 1 {
			htx.From = strings.Replace(string(ins[0].Addresses[0]), fmt.Sprintf("-%s", tokenAddress), "", -1)
			htx.To = strings.Replace(string(outs[0].Addresses[0]), fmt.Sprintf("-%s", tokenAddress), "", -1)
		} else {
			insStr, _ := json.Marshal(ins)
			outsStr, _ := json.Marshal(outs)
			htx.From = strings.Replace(string(insStr), fmt.Sprintf("-%s", tokenAddress), "", -1)
			htx.To = strings.Replace(string(outsStr), fmt.Sprintf("-%s", tokenAddress), "", -1)
		}
		htxs = append(htxs, htx)
	}
	return htxs, nil
}
