package main

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/erick785/services/common/log"
)

// Scanning sync new blocks and new pending txs from main blockchain.
func Scanning(ctx context.Context, db *Mysql, rpc *RPCClient, startHeight *big.Int) {
	//最新区块
	var curBlock *Block
	//初始化 回滚
	for {
		//获取本地最新区块
		block, err := db.GetBlockChain()
		if err != nil {
			log.Errorf("[Scanning] GetBlockChain --- %s", err)
			log.Panic(err)
		}
		curBlock = block

		//本地没有区块可回滚
		if block == nil {
			break
		}

		//获取节点最新区块
		block, err = rpc.GetBlockByNumber(curBlock.Number(), true)
		if err != nil {
			log.Errorf("[Scanning] GetBlockByNumber %s --- %s", curBlock.Number(), err)
			log.Panic(err)
		}

		//区块相同，无需回滚
		if strings.Compare(block.Hash(), curBlock.Hash()) == 0 {
			break
		}

		fmt.Println(block.Hash(), block.Number(), curBlock.Hash(), curBlock.Number())

		//区块回滚
		log.Infof("[Scanning] RollBack Block: height: %s hash: %s", curBlock.Number(), curBlock.Hash())
		if err := db.DeleteBlock(curBlock); err != nil {
			log.Errorf("[Scanning] DeleteBlock %s --- %s", curBlock.Number(), err)
			log.Panic(err)
		}
	}

	//开始高度 取最大
	fromNumber := big.NewInt(0)
	if curBlock != nil {
		fromNumber = new(big.Int).Add(curBlock.Number(), big.NewInt(1))
	}
	if startHeight != nil && startHeight.Cmp(fromNumber) > 0 {
		fromNumber = startHeight
	}

	log.Infof("[Scanning] FromNumber:%d ===>", fromNumber)
	pendingTxs := map[string]*Transaction{}
	for {
		select {
		case <-ctx.Done():
			break
		default:
		}

		block, err := rpc.GetBlockByNumber(fromNumber, true)
		if err != nil && !strings.Contains(err.Error(), "not found") {
			log.Errorf("[Scanning] GetBlockByNumber %s--- %s", fromNumber, err)
			continue
		}
		if block == nil {
			txs, err := rpc.GetRawMemPool(pendingTxs)
			if err != nil {
				log.Errorf("[Scanning] GetRawMemPool --- %s", err)
				continue
			}
			if err := db.InsertPendingTxs(txs); err != nil {
				log.Errorf("[Scanning] InsertPendingTxs --- %s", err)
				continue
			}
			pendingTxs = map[string]*Transaction{}
			for _, tx := range txs {
				pendingTxs[tx.TxHash()] = tx
			}

			time.Sleep(time.Second)
			continue
		}

		if curBlock != nil && strings.Compare(curBlock.Hash(), block.ParentHash()) != 0 {
			log.Warnf("[Scanning] RollBack Block: height: %s hash: %s", curBlock.Number(), curBlock.Hash())
			if err := db.DeleteBlock(curBlock); err != nil {
				log.Errorf("[Scanning] DeleteBlock %s--- %s", curBlock.Number(), err)
				continue
			}

			block, err := db.GetBlockChain()
			if err != nil {
				log.Errorf("[Scanning] GetBlockChain --- %s", err)
				continue
			}
			curBlock = block
		} else {
			curBlock = block
			if err := db.InsertBlock(curBlock); err != nil {
				log.Errorf("[Scanning] InsertBlock %s --- %s", curBlock.Number(), err)
				continue
			}
		}
		if curBlock != nil {
			fromNumber = new(big.Int).Add(curBlock.Number(), big.NewInt(1))
		} else {
			fromNumber = big.NewInt(0)
		}
	}
}
