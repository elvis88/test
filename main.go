package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math/big"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/erick785/services/common"
	"github.com/erick785/services/common/log"
	"github.com/erick785/services/common/sms"
	"github.com/erick785/services/common/wallet"
	gin "gopkg.in/gin-gonic/gin.v1"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	// listen 端口
	listenport := flag.Int("listenport", 8080, "api listen port")
	// Log 级别
	level := flag.String("level", "info", "log level, debug | info | warn | error | fatal | panic")
	// DB
	wdbname := flag.String("wdbname", "wallet", "db name")
	dbname := flag.String("dbname", "uranus", "db name")
	dbhost := flag.String("dbhost", "127.0.0.1:3306", "db host, ip:port")
	dbuser := flag.String("dbuser", "root", "db user")
	dbpassword := flag.String("dbpassword", "root", "db password")

	// RPC
	rpchost := flag.String("rpchost", "http://127.0.0.1:8000", "rpc host, http://ip:port")
	rpcuser := flag.String("rpcuser", "", "rpc user")
	rpcpassword := flag.String("rpcpassword", "", "rpc password")

	// white list
	whitelist := strings.Split(*flag.String("whitelist", "", "white list"), ",")

	// skiplist
	skiplist := strings.Split(*flag.String("skiplist", "test", "white list"), ",")

	flag.Parse()
	log.SetLevel(strings.ToLower(*level))

	inlist := func(list []string, elem string) bool {
		for _, l := range list {
			if strings.Compare(l, elem) == 0 {
				return true
			}
		}
		return false
	}

	// coindb
	db := &Mysql{
		DBName: *dbname,
		DBHost: *dbhost,
		DBUser: *dbuser,
		DBPWD:  *dbpassword,
		RPC: &RPCClient{
			RPCHost:     *rpchost,
			RPCUser:     *rpcuser,
			RPCPassword: *rpcpassword,
		},
	}
	if err := db.Open(); err != nil {
		panic(err)
	}
	defer db.Close()

	// Wallet
	wltdb := &wallet.Mysql{
		DBName: strings.ToLower(*wdbname),
		DBHost: *dbhost,
		DBUser: *dbuser,
		DBPWD:  *dbpassword,
	}
	if err := wltdb.Open(); err != nil {
		panic(err)
	}
	//初始化监控地址
	wlts, _ := wltdb.GetWallets()
	for _, wlt := range wlts {
		pub, err := wlt.DerivePublicKey(ParseDerivationPath(COINTYPE))
		if err != nil {
			log.Errorf("[Wallet] DerivePublicKey(%s) error:%v", wlt.Name, err)
		}
		address := ToAddress(pub)
		if err := db.AddMonitorAddress(address); err != nil {
			log.Errorf("[Wallet] AddMonitorAddress(%s) error:%v", address, err)
		}
	}

	// Scanning
	go Scanning(context.Background(), db, db.RPC, big.NewInt(0))

	router := gin.Default()
	router.POST("/changeprimarykey", func(c *gin.Context) {
		respone := &common.APIRespone{
			ErrCode: codeOk,
		}
		req := &ChangePrimaryKeyRequest{}
		if err := c.BindJSON(&req); err != nil {
			log.Errorf("[changePrimaryKey] %v BindJSON err %v", req.Phone, err)
			respone.ErrCode = codeRequest
		} else if len(req.Phone) == 0 || len(req.NewPhone) == 0 {
			log.Errorf("[changePrimaryKey] empty %v or %v", req.Phone, req.NewPhone)
			respone.ErrCode = codePhoneValidate
		} else if err := wltdb.UpdateWalletName(req.Phone, req.NewPhone); err != nil {
			log.Errorf("[changePrimaryKey] %v -> %v UpdateWalletName err %v", req.Phone, req.NewPhone, err)
			respone.ErrCode = codeWallet
		}
		respone.Data = "change success"
		respone.ErrMsg = msgs[respone.ErrCode]
		respone.Hash = respone.MD5()
		c.JSON(http.StatusOK, respone)
	})
	router.POST("/getaddressinfo", func(c *gin.Context) {
		respone := &common.APIRespone{
			ErrCode: codeOk,
		}
		req := &AddressInfoRequest{}
		if err := c.BindJSON(&req); err != nil {
			log.Errorf("[getaddressinfo] %v BindJSON err %v", req.Phone, err)
			respone.ErrCode = codeRequest
		} else if err := sms.VailMobile(req.Phone); err != nil {
			log.Errorf("[getaddressinfo] %v VailMobile err %v", req.Phone, err)
			respone.ErrCode = codePhoneValidate
		} else if wlt, err := wltdb.InsertOrGetWallet(req.Phone); err != nil {
			log.Errorf("[getaddressinfo] %v InsertOrGetWallet err %v", req.Phone, err)
			respone.ErrCode = codeWallet
		} else if pub, err := wlt.DerivePublicKey(ParseDerivationPath(COINTYPE)); err != nil {
			log.Errorf("[getaddressinfo] %v DerivePublicKey err %v", req.Phone, err)
			respone.ErrCode = codeWallet
		} else {
			addressInfo := &AddressInfoRespone{
				Address:      ToAddress(pub),
				TokenAddress: req.TokenAddress,
				Coin:         "urac",
				Decimal:      18,
			}
			if len(addressInfo.TokenAddress) > 0 {
				if tokenInfo, err := db.InsertOrUpdateTokenInfo(strings.ToLower(addressInfo.TokenAddress)); err != nil {
					log.Errorf("[getaddressinfo] %v InsertOrUpdateTokenInfo err %v", req.Phone, err)
					respone.ErrCode = codeDB
				} else {
					addressInfo.Coin = tokenInfo.Symbol
					addressInfo.Decimal = uint32(tokenInfo.Decimal)
				}
			}
			if amount, err := db.GetAmount(strings.ToLower(addressInfo.Address), strings.ToLower(addressInfo.TokenAddress)); err != nil {
				log.Errorf("[getaddressinfo] %v GetAmount err %v %v", req.Phone, req.TokenAddress, err)
				respone.ErrCode = codeDB
			} else {
				addressInfo.Amount = amount
			}
			if gasprice, err := db.GetGasPrice(); err != nil {
				log.Errorf("[getaddressinfo] %v GetGasPrice err %v %v", req.Phone, req.TokenAddress, err)
				respone.ErrCode = codeDB
			} else {
				addressInfo.GasPrice = gasprice
			}
			respone.Data = addressInfo
		}
		respone.ErrMsg = msgs[respone.ErrCode]
		respone.Hash = respone.MD5()
		c.JSON(http.StatusOK, respone)
	})
	router.POST("/gethistoryinfo", func(c *gin.Context) {
		respone := &common.APIRespone{
			ErrCode: codeOk,
		}
		req := &HistoryInfoRequest{
			PageNum:  0,
			PageSize: 20,
		}
		if err := c.BindJSON(&req); err != nil {
			log.Errorf("[gethistoryinfo] %v BindJSON err %v", req.Phone, err)
			respone.ErrCode = codeRequest
		} else if err := sms.VailMobile(req.Phone); err != nil {
			log.Errorf("[gethistoryinfo] %v VailMobile err %v", req.Phone, err)
			respone.ErrCode = codePhoneValidate
		} else if wlt, err := wltdb.InsertOrGetWallet(req.Phone); err != nil {
			log.Errorf("[gethistoryinfo] %v InsertOrGetWallet err %v", req.Phone, err)
			respone.ErrCode = codeWallet
		} else if pub, err := wlt.DerivePublicKey(ParseDerivationPath(COINTYPE)); err != nil {
			log.Errorf("[gethistoryinfo] %v DerivePublicKey err %v", req.Phone, err)
			respone.ErrCode = codeWallet
		} else if htxs, err := db.GetHistory(strings.ToLower(ToAddress(pub)), strings.ToLower(req.TokenAddress), req.PageSize, req.PageNum); err != nil {
			log.Errorf("[gethistoryinfo] %v GetHistory err %v", req.Phone, err)
			respone.ErrCode = codeDB
		} else {
			respone.Data = htxs
		}
		respone.ErrMsg = msgs[respone.ErrCode]
		respone.Hash = respone.MD5()
		c.JSON(http.StatusOK, respone)
	})
	router.POST("/getblkinfo", func(c *gin.Context) {
		respone := &common.APIRespone{
			ErrCode: codeOk,
		}
		req := &BlkInfoRequest{}
		if err := c.BindJSON(&req); err != nil {
			log.Errorf("[getblkinfo] %v BindJSON err %v", req.Height, err)
			respone.ErrCode = codeRequest
		} else if curBlock, err := db.GetBlockChain(); err != nil {
			log.Errorf("[getblkinfo] %v GetBlockChain err %v", req.Height, err)
			respone.ErrCode = codeDB
		} else {
			if req.Height < 0 {
				req.Height = curBlock.Height
			}
			if blk, err := db.RPC.GetBlockByNumberJSON(big.NewInt(req.Height), true); err != nil {
				log.Errorf("[getblkinfo] %v GetBlockByNumber err %v", req.Height, err)
				respone.ErrCode = codeRPC
			} else {
				respone.Data = blk
			}
		}
		respone.ErrMsg = msgs[respone.ErrCode]
		respone.Hash = respone.MD5()
		c.JSON(http.StatusOK, respone)
	})
	router.POST("/gettxinfo", func(c *gin.Context) {
		respone := &common.APIRespone{
			ErrCode: codeOk,
		}
		req := &TxInfoRequest{}
		if err := c.BindJSON(&req); err != nil {
			log.Errorf("[gettxinfo] %v BindJSON err %v", req.Phone, err)
			respone.ErrCode = codeRequest
		} else if len(req.Hash) == 0 {
			respone.ErrCode = codeHash
		} else if curBlock, err := db.GetBlockChain(); err != nil {
			log.Errorf("[gettxinfo] %v GetBlockChain err %v", req.Phone, err)
			respone.ErrCode = codeDB
		} else if tx, err := db.RPC.GetTransaction(req.Hash); err != nil {
			log.Errorf("[gettxinfo] %v GetTransaction err %v", req.Phone, err)
			respone.ErrCode = codeRPC
		} else {
			tokenAddress := ""
			ttx := &common.HistoryInfo{
				Hash:      tx.ID,
				Time:      tx.Time,
				Height:    tx.Height,
				Fee:       new(big.Int).SetBytes(tx.Fee.Bytes()),
				Size:      tx.Size,
				Signature: tx.Signature,
			}
			if tx.Height > 0 {
				ttx.Confirmations = curBlock.Height - tx.Height + 1
			}
			if ttx.Confirmations > 6 {
				ttx.Status = 1
			}
			var ins []*InOut
			ivalue := big.NewInt(0)
			for _, in := range tx.Ins {
				if strings.Contains(strings.Join(in.Addresses, ","), "-") {
					continue
				}
				ivalue = new(big.Int).Add(ivalue, in.Value)
				ins = append(ins, in)
			}
			var outs []*InOut
			ovalue := big.NewInt(0)
			for _, out := range tx.Outs {
				if strings.Contains(strings.Join(out.Addresses, ","), "-") {
					continue
				}
				ovalue = new(big.Int).Add(ovalue, out.Value)
				outs = append(outs, out)
			}
			ttx.Value = ivalue
			ttx.TValue = new(big.Int).Sub(ivalue, ovalue)
			if ttx.TValue.Sign() < 0 {
				ttx.TValue = new(big.Int).Abs(ttx.TValue)
			}
			if len(ins) == 1 && len(outs) == 1 {
				ttx.From = strings.Replace(string(ins[0].Addresses[0]), fmt.Sprintf("-%s", tokenAddress), "", -1)
				ttx.To = strings.Replace(string(outs[0].Addresses[0]), fmt.Sprintf("-%s", tokenAddress), "", -1)
			} else {
				insStr, _ := json.Marshal(ins)
				outsStr, _ := json.Marshal(outs)
				ttx.From = strings.Replace(string(insStr), fmt.Sprintf("-%s", tokenAddress), "", -1)
				ttx.To = strings.Replace(string(outsStr), fmt.Sprintf("-%s", tokenAddress), "", -1)
			}
			respone.Data = ttx
		}
		respone.ErrMsg = msgs[respone.ErrCode]
		respone.Hash = respone.MD5()
		c.JSON(http.StatusOK, respone)
	})
	router.POST("/confirm", func(c *gin.Context) {
		respone := &common.APIRespone{
			ErrCode: codeOk,
		}
		req := &ConfirmRequest{}
		if err := c.BindJSON(&req); err != nil {
			log.Errorf("[confirm] %v BindJSON err %v", req.Phone, err)
			respone.ErrCode = codeRequest
		} else if err := sms.VailMobile(req.Phone); err != nil {
			log.Errorf("[confirm] %v VailMobile err %v", req.Phone, err)
			respone.ErrCode = codePhoneValidate
		} else if inlist(whitelist, req.Phone) {
			token := &Token{
				Phone:      req.Phone,
				SendTxCode: "123456",
				SendTxTime: time.Now().Unix(),
			}
			getsessions(c).Set(req.Phone, token)
			respone.Data = "send code successfully"
		} else if code := sms.MakeCode(); sms.SendCode(req.Phone, code) {
			token := &Token{
				Phone:      req.Phone,
				SendTxCode: code,
				SendTxTime: time.Now().Unix(),
			}
			getsessions(c).Set(req.Phone, token)
			respone.Data = "send code successfully"
		} else {
			respone.ErrCode = codeSMS
		}
		respone.ErrMsg = msgs[respone.ErrCode]
		respone.Hash = respone.MD5()
		c.JSON(http.StatusOK, respone)
	})
	router.POST("/send", func(c *gin.Context) {
		respone := &common.APIRespone{
			ErrCode: codeOk,
		}
		req := &SendRequest{}
		if err := c.BindJSON(&req); err != nil {
			log.Errorf("[send] %v BindJSON err %v", req.Phone, err)
			respone.ErrCode = codeRequest
		} else if skip := inlist(skiplist, req.Phone); false {

		} else if err := sms.VailMobile(req.Phone); err != nil {
			log.Errorf("[send] %v VailMobile err %v", req.Phone, err)
			respone.ErrCode = codePhoneValidate
		} else if err := sms.VailCode(req.Code); !skip && err != nil {
			log.Errorf("[send] %v VailCode err %v", req.Phone, err)
			respone.ErrCode = codeSMSValidate
		} else if len(req.Order) == 0 {
			respone.ErrCode = codeOrder
		} else if token, ok := getsessions(c).Get(req.Phone).(*Token); !skip && !ok {
			respone.ErrCode = codeSMSValidate
		} else if !skip && strings.Compare(token.SendTxCode, req.Code) != 0 {
			respone.ErrCode = codeSMSValidate
		} else if !skip && time.Now().Sub(time.Unix(token.SendTxTime, 0)) > 600*time.Second {
			respone.ErrCode = codeSMSExpire
		} else if req.TokenAddress != "" && !ValidAddress(req.TokenAddress) {
			log.Errorf("[send] %v invalide token address %v", req.Phone, req.TokenAddress)
			respone.ErrCode = codeAddrValidate
		} else if wlt, err := wltdb.InsertOrGetWallet(req.Phone); err != nil {
			log.Errorf("[send] %v InsertOrGetWallet err %v", req.Phone, err)
			respone.ErrCode = codeWallet
		} else if pub, err := wlt.DerivePublicKey(ParseDerivationPath(COINTYPE)); err != nil {
			log.Errorf("[send] %v DerivePublicKey err %v", req.Phone, err)
			respone.ErrCode = codeWallet
		} else if privateKey, err := wlt.DerivePrivateKey(ParseDerivationPath(COINTYPE)); err != nil {
			log.Errorf("[send] %v DerivePrivateKey err %v", req.Phone, err)
			respone.ErrCode = codeWallet
		} else {
			getsessions(c).Delete(req.Phone)
			from := ToAddress(pub)
			if len(req.TokenAddress) > 0 {
				//TODO
			} else {
				res := map[string]string{}
				amount, nonce, _ := db.RPC.GetBalanceAndNone(strings.ToLower(from), strings.ToLower(req.TokenAddress))
				for _, order := range req.Order {
					if order.Gas == 0 {
						order.Gas = 21000
					}
					if order.GasPrice.Cmp(big.NewInt(0)) == 0 {
						gasprice, err := db.RPC.GetGasPrice()
						if err != nil {
							res[order.ID] = err.Error()
							continue
						}
						gasprice.Sub(gasprice, new(big.Int).SetBytes(gasprice.Bytes()).Mod(new(big.Int).SetBytes(gasprice.Bytes()), big.NewInt(1e9)))
						order.GasPrice = *gasprice
					}
					if amount.Cmp(new(big.Int).Add(&order.Value, new(big.Int).Mul(&order.GasPrice, new(big.Int).SetInt64(order.Gas)))) < 0 {
						res[order.ID] = fmt.Sprintf("not sufficient funds %v < %v", amount, new(big.Int).Mul(&order.GasPrice, new(big.Int).SetInt64(order.Gas)))
					} else if signedhash, err := CreateTx(privateKey, nonce.Uint64(), order.To, &order.Value, uint64(order.Gas), &order.GasPrice, nil); err != nil {
						res[order.ID] = err.Error()
					} else if hash, err := db.RPC.SendRawTransaction(fmt.Sprintf("0x%s", signedhash)); err != nil {
						res[order.ID] = err.Error()
					} else {
						res[order.ID] = hash
					}
					amount = new(big.Int).Sub(amount, new(big.Int).Add(&order.Value, new(big.Int).Mul(&order.GasPrice, new(big.Int).SetInt64(order.Gas))))
					nonce = new(big.Int).Add(nonce, big.NewInt(1))
				}
				respone.Data = res
			}
		}
		respone.ErrMsg = msgs[respone.ErrCode]
		respone.Hash = respone.MD5()
		c.JSON(http.StatusOK, respone)
	})
	router.POST("/getfee", func(c *gin.Context) {
		respone := &common.APIRespone{
			ErrCode: codeOk,
		}
		req := &AddressInfoRequest{}
		if err := c.BindJSON(&req); err != nil {
			log.Errorf("[getfee] %v BindJSON err %v", req.Phone, err)
			respone.ErrCode = codeRequest
		} else if gasprice, err := db.GetGasPrice(); err != nil {
			log.Errorf("[getfee] %v GetGasPrice err %v", req.Phone, err)
			respone.ErrCode = codeRPC
		} else {
			gasprice.Sub(gasprice, new(big.Int).SetBytes(gasprice.Bytes()).Mod(new(big.Int).SetBytes(gasprice.Bytes()), big.NewInt(1e9)))
			fee := &Fee{
				Gas:      21000,
				GasPrice: *gasprice,
			}
			if len(req.TokenAddress) > 0 {
				// TODO
			}
			respone.Data = fee
		}
		respone.ErrMsg = msgs[respone.ErrCode]
		respone.Hash = respone.MD5()
		c.JSON(http.StatusOK, respone)
	})
	if err := router.Run(fmt.Sprintf(":%d", *listenport)); err != nil {
		panic(err)
	}
}

// ChangePrimaryKeyRequest 修改
type ChangePrimaryKeyRequest struct {
	Phone    string `json:"phone" binding:"required"`
	NewPhone string `json:"new_phone" binding:"required"`
}

//AddressInfoRequest 地址请求
type AddressInfoRequest struct {
	Phone        string `json:"phone" binding:"required"`
	TokenAddress string `json:"token_address"` //token 地址
}

//HistoryInfoRequest 历史请求
type HistoryInfoRequest struct {
	Phone        string `json:"phone" binding:"required"`
	TokenAddress string `json:"token_address"`
	PageNum      int64  `json:"page_num"`
	PageSize     int64  `json:"page_size"`
}

//BlkInfoRequest 区块请求
type BlkInfoRequest struct {
	Height int64 `json:"height" binding:"required"`
}

//TxInfoRequest 交易请求
type TxInfoRequest struct {
	Phone        string `json:"phone" binding:"required"`
	TokenAddress string `json:"token_address"`
	Hash         string `json:"hash"`
}

// ConfirmRequest 发送交易验证码
type ConfirmRequest struct {
	Phone string `json:"phone" binding:"required"`
}

// SendRequest 发送交易
type SendRequest struct {
	Phone        string   `json:"phone" binding:"required"`
	TokenAddress string   `json:"token_address"` //token 地址
	Code         string   `json:"code"`          //验证码
	Order        []*Order `json:"order"`         //订单列表
}

// Order 订单
type Order struct {
	ID       string  `json:"id"`        //id
	To       string  `json:"to"`        //接收方
	Value    big.Int `json:"value"`     //接收金额
	Gas      int64   `json:"gas"`       //手续费
	GasPrice big.Int `json:"gas_price"` //手续费
}

//AddressInfoRespone 地址信息
type AddressInfoRespone struct {
	Address      string   `json:"address"`       //拥有的地址
	TokenAddress string   `json:"token_address"` //token 地址
	Amount       *big.Int `json:"amount"`        //账户余额
	GasPrice     *big.Int `json:"gas_price"`     //费率
	Coin         string   `json:"coin"`          //拥有的币名
	Decimal      uint32   `json:"decimal"`       //拥有的币类型
}

type Fee struct {
	Gas      int64   `json:"gas"`       //手续费
	GasPrice big.Int `json:"gas_price"` //手续费
}
