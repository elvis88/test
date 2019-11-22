package main

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"time"

	"github.com/Jeffail/gabs"
	"github.com/erick785/services/common"
	"github.com/erick785/services/common/log"
)

const (
	methodGetBlockByNumber      = "BlockChain.GetBlockByHeight"
	methodGasPrice              = "Uranus.SuggestGasPrice"
	methodTxPool                = "TxPool.Content"
	methodSendRawTransaction    = "Uranus.SendRawTransaction"
	methodGetTransaction        = "BlockChain.GetTransactionByHash"
	methodGetTransactionReceipt = "BlockChain.GetTransactionReceipt"
	methodGetBalance            = "Uranus.GetBalance"
	methodGetTransactionCount   = "Uranus.GetNonce"
	methodCall                  = "Uranus.Call"
)

// RPCClient rpc
type RPCClient struct {
	RPCHost     string
	RPCUser     string
	RPCPassword string

	// 	GetRawMemPool() ([]ITransaction, error)
	// 	GetBlockByNumber(number *big.Int) (IBlock, error)
	//	GetBlockByNumberJSON(number *big.Int) (string, error)
	// 	GetGasPrice() (*big.Int, error)
	// 	SendRawTransaction(signed string) (string, error)
}

//GetRawMemPool 获取内存池交易
func (client *RPCClient) GetRawMemPool(otxs map[string]*Transaction) ([]*Transaction, error) {
	t := time.Now()
	cnt := 0
	defer func() {
		if cnt > 0 {
			log.Infof("[RPC] GetRawMemPool elpase: %s, txs: %d", time.Now().Sub(t), cnt)
		}
	}()

	request := common.NewRPCRequest("2.0", methodTxPool)
	jsonParsed, err := common.SendRPCRequst(client.RPCHost, request)
	if err != nil {
		return nil, fmt.Errorf("GetRawMemPool SendRPCRequst error --- %s", err)
	}

	if jsonParsed.Path("error").Data() != nil {
		msg, _ := jsonParsed.Path("error").Data().(string)
		return nil, fmt.Errorf("GetRawMemPool error --- %s", msg)
	}

	if /*value*/ _, ok := jsonParsed.Path("error.code").Data().(float64); ok /*&& value > 0*/ {
		msg, _ := jsonParsed.Path("error.message").Data().(string)
		return nil, fmt.Errorf("GetRawMemPool error --- %s", msg)
	}

	if jsonParsed.Path("result").Data() == nil {
		return nil, nil
	}

	txs := []*Transaction{}
	children, _ := jsonParsed.S("result", "pending").ChildrenMap()
	for _, child := range children {
		tchildren, _ := child.ChildrenMap()
		for _, tchild := range tchildren {
			tx, err := client.decodeTransactionJSON(tchild)
			if err != nil {
				return nil, err
			}
			if ttx, ok := otxs[tx.TxHash()]; ok {
				txs = append(txs, ttx)
			} else if tx != nil {
				txs = append(txs, tx)
			}
		}
	}
	cnt = len(txs)
	return txs, nil
}

// GetBlockByNumberJSON 获取指定高度的区块
func (client *RPCClient) GetBlockByNumberJSON(number *big.Int, full bool) (interface{}, error) {
	t := time.Now()
	cnt := int64(0)
	defer func() {
		log.Infof("[RPC] GetBlockByNumberJSON %s elpase: %s, txs: %d", number, time.Now().Sub(t), cnt)
	}()

	request := common.NewRPCRequest("2.0", methodGetBlockByNumber, map[string]interface{}{
		"BlockHeight": fmt.Sprintf("0x%x", number),
		"FullTX":      full,
	})
	jsonParsed, err := common.SendRPCRequst(client.RPCHost, request)
	if err != nil {
		return nil, fmt.Errorf("GetBlockByNumberJSON SendRPCRequst error --- %s", err)
	}

	if jsonParsed.Path("error").Data() != nil {
		msg, _ := jsonParsed.Path("error").Data().(string)
		return nil, fmt.Errorf("GetBlockByNumberJSON rpc error --- %s", msg)
	}

	if _, ok := jsonParsed.Path("error.code").Data().(float64); ok /*&& value > 0*/ {
		msg, _ := jsonParsed.Path("error.message").Data().(string)
		return nil, fmt.Errorf("GetBlockByNumberJSON rpc error --- %s", msg)
	}

	return jsonParsed.Path("result").Data(), nil
}

// GetBlockByNumber 获取指定高度的区块
func (client *RPCClient) GetBlockByNumber(number *big.Int, full bool) (*Block, error) {
	t := time.Now()
	cnt := int64(0)
	defer func() {
		log.Infof("[RPC] GetBlockByNumber %s elpase: %s, txs: %d", number, time.Now().Sub(t), cnt)
	}()

	request := common.NewRPCRequest("2.0", methodGetBlockByNumber, map[string]interface{}{
		"BlockHeight": fmt.Sprintf("0x%x", number),
		"FullTX":      full,
	})
	jsonParsed, err := common.SendRPCRequst(client.RPCHost, request)
	if err != nil {
		return nil, fmt.Errorf("GetBlockByNumber SendRPCRequst error --- %s", err)
	}

	if jsonParsed.Path("error").Data() != nil {
		msg, _ := jsonParsed.Path("error").Data().(string)
		return nil, fmt.Errorf("GetBlockByNumber rpc error --- %s", msg)
	}

	if _, ok := jsonParsed.Path("error.code").Data().(float64); ok /*&& value > 0*/ {
		msg, _ := jsonParsed.Path("error.message").Data().(string)
		return nil, fmt.Errorf("GetBlockByNumber rpc error --- %s", msg)
	}

	if jsonParsed.Path("result").Data() == nil {
		return nil, nil
	}
	return client.decodeBlockJSON(jsonParsed.Path("result"))
}

// GetTransaction 获取指定哈希的交易
func (client *RPCClient) GetTransaction(hash string) (*Transaction, error) {
	request := common.NewRPCRequest("2.0", methodGetTransaction, hash)
	jsonParsed, err := common.SendRPCRequst(client.RPCHost, request)
	if err != nil {
		return nil, fmt.Errorf("GetTransaction SendRPCRequst error --- %s", err)
	}

	if jsonParsed.Path("error").Data() != nil {
		msg, _ := jsonParsed.Path("error").Data().(string)
		return nil, fmt.Errorf("GetTransaction rpc error --- %s", msg)
	}

	if _, ok := jsonParsed.Path("error.code").Data().(float64); ok /*&& value > 0*/ {
		msg, _ := jsonParsed.Path("error.message").Data().(string)
		return nil, fmt.Errorf("GetTransaction rpc error --- %s", msg)
	}

	if jsonParsed.Path("result").Data() == nil {
		return nil, nil
	}
	return client.decodeTransactionJSON(jsonParsed.Path("result"))
}

// GetGasPrice 获取费率
func (client *RPCClient) GetGasPrice() (*big.Int, error) {
	request := common.NewRPCRequest("2.0", methodGasPrice)
	jsonParsed, err := common.SendRPCRequst(client.RPCHost, request)
	if err != nil {
		return big.NewInt(0), fmt.Errorf("getGasPrice SendRPCRequst error --- %s", err)
	}

	if jsonParsed.Path("error").Data() != nil {
		msg, _ := jsonParsed.Path("error").Data().(string)
		return nil, fmt.Errorf("getGasPrice error --- %s", msg)
	}

	if /*value*/ _, ok := jsonParsed.Path("error.code").Data().(float64); ok /*&& value > 0*/ {
		msg, _ := jsonParsed.Path("error.message").Data().(string)
		return nil, fmt.Errorf("getGasPrice error --- %s", msg)
	}

	r, ok := jsonParsed.Path("result").Data().(string)
	if !ok {
		return big.NewInt(0), fmt.Errorf("getGasPrice Path('result') interface error --- %s", err)
	}

	ret := new(big.Int)
	ret.UnmarshalJSON([]byte(r))
	return ret, nil
}

// SendRawTransaction 发送交易
func (client *RPCClient) SendRawTransaction(signed string) (string, error) {
	request := common.NewRPCRequest("2.0", methodSendRawTransaction, signed)
	jsonParsed, err := common.SendRPCRequst(client.RPCHost, request)
	if err != nil {
		return "", fmt.Errorf("SendRawTransaction SendRPCRequst error --- %s", err)
	}

	if jsonParsed.Path("error").Data() != nil {
		msg, _ := jsonParsed.Path("error").Data().(string)
		return "", fmt.Errorf("SendRawTransaction rpc error --- %s", msg)
	}

	if /*value*/ _, ok := jsonParsed.Path("error.code").Data().(float64); ok /*&& value > 0*/ {
		msg, _ := jsonParsed.Path("error.message").Data().(string)
		return "", fmt.Errorf("SendRawTransaction rpc error --- %s", msg)
	}

	r, ok := jsonParsed.Path("result").Data().(string)
	if !ok {
		return "", fmt.Errorf("SendRawTransaction Path('result') interface error --- %v", reflect.TypeOf(jsonParsed.Path("result").Data()))
	}

	return r, nil
}

func (client *RPCClient) decodeBlockJSON(jsonParsed *gabs.Container) (*Block, error) {
	// 	{
	// 	"difficulty": "0xf4240",
	// 	"extraData": "0x",
	// 	"gasLimit": "0x4c4b40",
	// 	"gasUsed": "0x0",
	// 	"hash": "0xd4e1dcfda3c2d0a54bea6905a149f6394ce34ba4bacafe4f29b94467f17e041c",
	// 	"height": "0x0",
	// 	"miner": "0x0000000000000000000000000000000000000000",
	// 	"nonce": "0x0000000000000001",
	// 	"parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
	// 	"receiptsRoot": "0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421",
	// 	"size": "0x1b8",
	// 	"stateRoot": "0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421",
	// 	"timestamp": "0x0",
	// 	"totalDifficulty": "0xf4240",
	// 	"transactions": [],
	// 	"transactionsRoot": "0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"
	// }

	blk := &Block{
		Transactions: make(map[string]*Transaction),
	}
	blk.ID = jsonParsed.Path("hash").Data().(string)
	blk.PrevID = jsonParsed.Path("parentHash").Data().(string)
	ret := new(big.Int)
	ret.UnmarshalJSON([]byte(jsonParsed.Path("height").Data().(string)))
	blk.Height = ret.Int64()
	ret.UnmarshalJSON([]byte(jsonParsed.Path("timestamp").Data().(string)))
	blk.Time = ret.Int64() / int64(time.Second)
	ret.UnmarshalJSON([]byte(jsonParsed.Path("gasUsed").Data().(string)))

	blk.Miner = jsonParsed.Path("miner").Data().(string)
	ret.UnmarshalJSON([]byte(jsonParsed.Path("gasLimit").Data().(string)))
	blk.GasLimit = ret.Int64()
	ret.UnmarshalJSON([]byte(jsonParsed.Path("gasUsed").Data().(string)))
	blk.GasUsed = ret.Int64()

	children, _ := jsonParsed.S("transactions").Children()
	for _, child := range children {
		tx, err := client.decodeTransactionJSON(child)
		if err != nil {
			return nil, err
		} else if tx != nil {
			tx.Height = blk.Height
			tx.Time = blk.Time
			blk.Transactions[tx.ID] = tx
		}
	}

	return blk, nil
}

func (client *RPCClient) decodeTransactionJSON(jsonParsed *gabs.Container) (*Transaction, error) {
	// 	{
	// 	"blockHash": "0x000010017413ef42e7542b3693f69e918dc8cdc18ac83c621d40d73fdda7756c",
	// 	"blockHeight": "0x8",
	// 	"from": "0x210f48c05511dd64da1f46f9163d2e2c75bba988",
	// 	"gas": "0x76c0",
	// 	"gasPrice": "0x9184e72a000",
	// 	"hash": "0x266f00801f59cf7036b62be91fa9064f6017ccf22d4fbc8196fd6ccc708bd7d6",
	// 	"input": "0xd46e8dd67c5d32be8d46e8dd67c5d32be8058bb8eb970870f072445675058bb8eb970870f072445675",
	// 	"nonce": "0x0",
	// 	"to": "0xd46e8dd67c5d32be8058bb8eb970870f07244567",
	// 	"transactionIndex": "0x0",
	// 	"value": "0x9184e72a",
	// 	"signature": "0xd9f262752b404651a581933af6aaeb2b43f682d9229eca2cbdc07e3b8b5367d97992571033155d792396424e77124fd98b79415dc3ca1c9f46bf1f9e1cbac5c900"
	// }

	tx := &Transaction{
		Fee: big.NewInt(0),
	}
	tx.ID = jsonParsed.Path("hash").Data().(string)
	tx.Time = time.Now().Unix()
	if jsonParsed.Path("blockHeight").Data() != nil {
		ret := new(big.Int)
		ret.UnmarshalJSON([]byte(jsonParsed.Path("blockHeight").Data().(string)))
		tx.Height = ret.Int64()
	}

	from := strings.ToLower(jsonParsed.Path("from").Data().(string))
	to := "UNKOWN"
	if jsonParsed.Path("tos").Data() != nil {
		if cnt, _ := jsonParsed.ArrayCount("tos"); cnt == 1 {
			to = strings.ToLower(jsonParsed.S("tos").Index(0).Data().(string))
		}
	}
	value := new(big.Int)
	value.UnmarshalJSON([]byte(jsonParsed.Path("value").Data().(string)))
	gasprice := new(big.Int)
	gasprice.UnmarshalJSON([]byte(jsonParsed.Path("gasPrice").Data().(string)))
	gasUsed := new(big.Int)
	gasUsed.UnmarshalJSON([]byte(jsonParsed.Path("gas").Data().(string)))
	tx.Signature = jsonParsed.Path("signature").Data().(string)
	var tins, touts []*InOut
	if jsonParsed.Path("blockHeight").Data() != nil {
		request := common.NewRPCRequest("2.0", methodGetTransactionReceipt, tx.ID)
		jsonParsed, err := common.SendRPCRequst(client.RPCHost, request)
		if err != nil {
			return nil, fmt.Errorf("getTransactionReceipt SendRPCRequst error --- %s", err)
		}

		if /*value*/ _, ok := jsonParsed.Path("error.code").Data().(float64); ok /*&& value > 0*/ {
			msg, _ := jsonParsed.Path("error.message").Data().(string)
			return nil, fmt.Errorf("getTransactionReceipt rpc error --- %s", msg)
		}

		if jsonParsed.Path("result").Data() == nil {
			return nil, nil
		}

		if jsonParsed.Path("result.contractAddress").Data() != nil {
			if strings.Compare(to, "UNKOWN") == 0 {
				to = strings.ToLower(jsonParsed.Path("result.contractAddress").Data().(string))
			}
		}
		if jsonParsed.Path("result.gasUsed").Data() != nil {
			gasUsed.UnmarshalJSON([]byte(jsonParsed.Path("result.gasUsed").Data().(string)))
		}
		// Token
		logs, _ := jsonParsed.S("result", "logs").Children()
		for _, log := range logs {
			if cnt, _ := log.ArrayCount("topics"); cnt != 3 {
				continue
			}
			if strings.Compare(log.S("topics").Index(0).Data().(string), "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef") != 0 {
				continue
			}
			token := strings.ToLower(log.Path("address").Data().(string))
			tokenFrom := strings.ToLower(string(append([]byte{'0', 'x'}, log.S("topics").Index(1).Data().(string)[26:]...)))
			tokenTo := strings.ToLower(string(append([]byte{'0', 'x'}, log.S("topics").Index(2).Data().(string)[26:]...)))
			tokenValue := big.NewInt(0)
			tokenValue.UnmarshalJSON([]byte(log.Path("data").Data().(string)))
			tins = append(tins, &InOut{
				Addresses: []string{fmt.Sprintf("%s-%s", tokenFrom, token)},
				Value:     new(big.Int).SetBytes(tokenValue.Bytes()),
			})
			touts = append(touts, &InOut{
				Addresses: []string{fmt.Sprintf("%s-%s", tokenTo, token)},
				Value:     new(big.Int).SetBytes(tokenValue.Bytes()),
			})
		}
	} else {
		input := jsonParsed.Path("input").Data().(string)
		// Function: transfer(address _to, uint256 _value)
		// MethodID: 0xa9059cbb
		// 0xa9059cbb000000000000000000000000
		// [0]:00000000000000000000000075186ece18d7051afb9c1aee85170c0deda23d82
		// [1]:0000000000000000000000000000000000000000000000364db9fbe6a7902000
		if len(input) > 74 && string(input[:10]) == "0xa9059cbb" {
			token := to
			tokenFrom := from
			tokenTo := strings.ToLower(string(append([]byte{'0', 'x'}, input[34:74]...)))
			tokenValue := new(big.Int)
			tokenValue.UnmarshalJSON(append([]byte{'0', 'x'}, input[74:]...))
			tins = append(tins, &InOut{
				Addresses: []string{fmt.Sprintf("%s-%s", tokenFrom, token)},
				Value:     new(big.Int).SetBytes(tokenValue.Bytes()),
			})
			touts = append(touts, &InOut{
				Addresses: []string{fmt.Sprintf("%s-%s", tokenTo, token)},
				Value:     new(big.Int).SetBytes(tokenValue.Bytes()),
			})
		}
	}

	tx.Fee = new(big.Int).Mul(gasUsed, gasprice)
	tx.Size = gasUsed.Int64()
	tx.Ins = append(tx.Ins, &InOut{
		Addresses: []string{from},
		Value:     new(big.Int).Add(value, tx.Fee),
	})
	tx.Outs = append(tx.Outs, &InOut{
		Addresses: []string{to},
		Value:     new(big.Int).SetBytes(value.Bytes()),
	})
	tx.Ins = append(tx.Ins, tins...)
	tx.Outs = append(tx.Outs, touts...)
	return tx, nil
}

func (client *RPCClient) getTransactionCount(address string, number *big.Int) (*big.Int, error) {
	h := big.NewInt(-1)
	if number != nil {
		h = number
	}
	_ = h
	request := common.NewRPCRequest("2.0", methodGetTransactionCount, map[string]interface{}{
		"Address":     address,
		"BlockHeight": "latest",
	})
	jsonParsed, err := common.SendRPCRequst(client.RPCHost, request)
	if err != nil {
		return big.NewInt(0), fmt.Errorf("getTransactionCount SendRPCRequst error --- %s", err)
	}

	if jsonParsed.Path("error").Data() != nil {
		msg, _ := jsonParsed.Path("error").Data().(string)
		return nil, fmt.Errorf("getTransactionCount error ---%s %s", address, msg)
	}

	if /*value*/ _, ok := jsonParsed.Path("error.code").Data().(float64); ok /*&& value > 0*/ {
		msg, _ := jsonParsed.Path("error.message").Data().(string)
		return nil, fmt.Errorf("getTransactionCount error ---%s %s", address, msg)
	}

	r, ok := jsonParsed.Path("result").Data().(string)
	if !ok {
		return big.NewInt(0), fmt.Errorf("getTransactionCount Path('result') interface error --- %v", reflect.TypeOf(jsonParsed.Path("result").Data()))
	}

	var ret = big.NewInt(0)
	ret.UnmarshalJSON([]byte(r))
	return ret, nil
}

func (client *RPCClient) getBalance(address string, token string, number *big.Int) (*big.Int, error) {
	h := big.NewInt(-1)
	if number != nil {
		h = number
	}
	_ = h
	request := common.NewRPCRequest("2.0", methodGetBalance, map[string]interface{}{
		"Address":     address,
		"BlockHeight": "latest",
	})
	if len(token) > 0 {
		request = common.NewRPCRequest("2.0", methodCall, map[string]interface{}{
			"To":          token,
			"Data":        fmt.Sprintf("0x70a08231000000000000000000000000%s", address[2:]),
			"BlockHeight": "latest",
		})
	}
	jsonParsed, err := common.SendRPCRequst(client.RPCHost, request)
	if err != nil {
		return big.NewInt(0), fmt.Errorf("getBalance SendRPCRequst error --- %s", err)
	}

	if jsonParsed.Path("error").Data() != nil {
		msg, _ := jsonParsed.Path("error").Data().(string)
		return nil, fmt.Errorf("getBalance error --- %s", msg)
	}

	if /*value*/ _, ok := jsonParsed.Path("error.code").Data().(float64); ok /*&& value > 0*/ {
		msg, _ := jsonParsed.Path("error.message").Data().(string)
		return nil, fmt.Errorf("getBalance error --- %s", msg)
	}

	r, ok := jsonParsed.Path("result").Data().(string)
	if !ok {
		return big.NewInt(0), fmt.Errorf("getBalance Path('result') interface error --- %v", reflect.TypeOf(jsonParsed.Path("result").Data()))
	}
	var ret = big.NewInt(0)
	ret.UnmarshalJSON([]byte(r))
	return ret, nil
}

func (client *RPCClient) GetBalanceAndNone(address string, token string) (*big.Int, *big.Int, error) {
	nonce, err := client.getTransactionCount(address, nil)
	if err != nil {
		return nil, nil, err
	}
	balance, err := client.getBalance(address, token, nil)
	if err != nil {
		return nil, nil, err
	}
	return balance, nonce, nil
}

func (client *RPCClient) GetTokenSymbol(token string) (string, error) {
	data := "0x95d89b41"
	request := common.NewRPCRequest("2.0", methodCall, map[string]interface{}{
		"To":          token,
		"Data":        data,
		"BlockHeight": "latest",
	})
	jsonParsed, err := common.SendRPCRequst(client.RPCHost, request)
	if err != nil {
		return "", fmt.Errorf("GetTokenSymbol SendRPCRequst error --- %s", err)
	}

	if /*value*/ _, ok := jsonParsed.Path("error.code").Data().(float64); ok /*&& value > 0*/ {
		msg, _ := jsonParsed.Path("error.message").Data().(string)
		return "", fmt.Errorf("GetTokenSymbol error --- %s", msg)
	}

	if jsonParsed.Path("result").Data() == nil {
		return "", nil
	}

	r, ok := jsonParsed.Path("result").Data().(string)
	if !ok {
		return "", fmt.Errorf("GetTokenSymbol Path('result') interface error --- %s", jsonParsed.String())
	}

	decoded, err := hex.DecodeString(strings.TrimLeft(r, "0x"))
	return strings.Trim(strings.TrimSpace(string(decoded)), "\x00"), nil
}

func (client *RPCClient) GetTokenName(token string) (string, error) {
	data := "0x06fdde03"
	request := common.NewRPCRequest("2.0", methodCall, map[string]interface{}{
		"To":          token,
		"Data":        data,
		"BlockHeight": "latest",
	})
	jsonParsed, err := common.SendRPCRequst(client.RPCHost, request)
	if err != nil {
		return "", fmt.Errorf("GetTokenName SendRPCRequst error --- %s", err)
	}

	if /*value*/ _, ok := jsonParsed.Path("error.code").Data().(float64); ok /*&& value > 0*/ {
		msg, _ := jsonParsed.Path("error.message").Data().(string)
		return "", fmt.Errorf("GetTokenName error --- %s", msg)
	}

	if jsonParsed.Path("result").Data() == nil {
		return "", nil
	}

	result, ok := jsonParsed.Path("result").Data().(string)
	if !ok {
		return "", fmt.Errorf("GetTokenName Path('result') interface error --- %s", jsonParsed.String())
	}

	decoded, err := hex.DecodeString(strings.TrimLeft(result, "0x"))
	return strings.Trim(strings.TrimSpace(string(decoded)), "\x00"), nil
}

func (client *RPCClient) GetTokenDecimal(token string) (*big.Int, error) {
	data := "0x313ce567"
	request := common.NewRPCRequest("2.0", methodCall, map[string]interface{}{
		"To":          token,
		"Data":        data,
		"BlockHeight": "latest",
	})
	jsonParsed, err := common.SendRPCRequst(client.RPCHost, request)
	if err != nil {
		return nil, fmt.Errorf("GetTokenDecimal SendRPCRequst error --- %s", err)
	}

	if /*value*/ _, ok := jsonParsed.Path("error.code").Data().(float64); ok /*&& value > 0*/ {
		msg, _ := jsonParsed.Path("error.message").Data().(string)
		return nil, fmt.Errorf("GetTokenDecimal error --- %s", msg)
	}

	if jsonParsed.Path("result").Data() == nil {
		return nil, nil
	}

	r, ok := jsonParsed.Path("result").Data().(string)
	if !ok {
		return nil, fmt.Errorf("GetTokenDecimal Path('result') interface error --- %s", jsonParsed.String())
	}
	var ret = big.NewInt(0)
	ret.UnmarshalJSON([]byte(r))

	return ret, nil
}
