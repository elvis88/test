package main

import (
	"container/list"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/big"
	"net/url"
	"strings"
	"sync"
	"time"

	// mysql

	"github.com/erick785/services/common/log"
	_ "github.com/go-sql-driver/mysql"
)

var (
	confirmed = 300
)

//Mysql implement mysql
type Mysql struct {
	DBName string
	DBUser string
	DBPWD  string
	DBHost string
	db     *sql.DB
	RPC    *RPCClient

	writeBlockChan chan *list.Element // 已可安全写入db
	memBlocks      *list.List         // 缓存10个块， 未安全，易回滚
	memBlocksRW    sync.RWMutex
	pendingBlock   *Block // 内存池
	pendingBlockRW sync.RWMutex
	elemChan       *list.Element

	tokenChan chan string
}

// Open open a db and create tables if necessary.
func (mysql *Mysql) Open() error {
	connStr := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8&loc=%s&parseTime=true",
		mysql.DBUser, mysql.DBPWD, mysql.DBHost, mysql.DBName, url.QueryEscape("Asia/Shanghai"))

	db, err := sql.Open("mysql", connStr)
	if err != nil {
		return err
	}
	db.SetMaxOpenConns(2000)
	db.SetMaxIdleConns(2000)
	db.SetConnMaxLifetime(60 * time.Second)
	mysql.db = db

	if err := mysql.execSQL(initSQL); err != nil {
		db.Close()
		return err
	}

	mysql.memBlocks = list.New()
	mysql.writeBlockChan = make(chan *list.Element, 100)
	mysql.tokenChan = make(chan string, 100)
	go func() {
		for {
			select {
			case elem := <-mysql.writeBlockChan:
				blk := elem.Value.(*Block)
				t := time.Now()
				if err := mysql.execSQL(mysql.getSQL(blk)); err != nil {
					panic(err)
				}
				log.Infof("[MYSQL] write block %d, elpase %s", blk.Height, time.Now().Sub(t))
				mysql.memBlocksRW.Lock()
				mysql.memBlocks.Remove(elem)
				mysql.memBlocksRW.Unlock()
			case token := <-mysql.tokenChan:
				if _, err := mysql.InsertOrUpdateTokenInfo(token); err != nil {
					log.Errorf("[MYSQL] insert or update token %s - %s", token, err)
				}
			}
		}
	}()
	return nil
}

// Close close db
func (mysql *Mysql) Close() error {
	return mysql.db.Close()
}

func (mysql *Mysql) execSQL(sqlStr string) error {
	sqlStrs := strings.Split(sqlStr, ";")
	tx, err := mysql.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if tx != nil {
			tx.Rollback()
		}
	}()
	for _, sqlStr := range sqlStrs {
		sqlStr = strings.TrimSpace(sqlStr)
		if len(sqlStr) != 0 {
			if _, err := tx.Exec(fmt.Sprintf("%s;", sqlStr)); err != nil {
				return fmt.Errorf("%s - %s", sqlStr, err)
			}
		}
	}
	err = tx.Commit()
	if err == nil {
		tx = nil
	}
	return err
}

func (mysql *Mysql) getSQL(blk *Block) string {
	isMonitorAddresses := func(addresses []string) bool {
		for _, address := range addresses {
			list := strings.Split(address, "-")
			if mysql.IsMonitorAddress(list[0]) {
				return true
			}
		}
		return false
	}
	//blockchain
	sqlStr := fmt.Sprintf("REPLACE INTO t_blockchain(id, i_height, i_created, s_hash, s_prevhash) values(1, %d, %d, '%s', '%s');",
		blk.Height, blk.Time, blk.ID, blk.PrevID)
	//tx
	for _, tx := range blk.Transactions {
		addresses := []string{}
		for _, in := range tx.Ins {
			addresses = append(addresses, in.Addresses...)
		}
		for _, out := range tx.Outs {
			addresses = append(addresses, out.Addresses...)
		}
		if !isMonitorAddresses(addresses) {
			continue
		}
		ins, _ := json.Marshal(tx.Ins)
		outs, _ := json.Marshal(tx.Outs)
		sqlStr += fmt.Sprintf("INSERT INTO t_transaction(s_hash, s_ins, s_outs, i_created, i_height, s_fee, i_size) values("+
			"'%s','%s','%s', %d, %d, '%s', %d);",
			tx.ID, ins, outs, tx.Time, tx.Height, tx.Fee, tx.Size)
	}
	//address
	for address, addressInfo := range blk.addressInfos {
		if !isMonitorAddresses([]string{address}) {
			continue
		}
		sqlStr += fmt.Sprintf("REPLACE INTO t_address(s_address, s_value) values('%s', '%s');", address, addressInfo.Amount)
		for _, hash := range addressInfo.HTxs {
			sqlStr += fmt.Sprintf("REPLACE INTO t_history(s_address, s_hash) values('%s', '%s');", address, hash)
		}
	}
	return sqlStr
}

//InsertBlock 新增区块
func (mysql *Mysql) InsertBlock(blk *Block) error {
	t := time.Now()
	defer func() {
		log.Infof("[MYSQL] insert block %d elpase %s", blk.Height, time.Now().Sub(t))
	}()

	blk.addressInfos = make(map[string]*AddressInfo)
	for _, tx := range blk.Transactions {
		if err := mysql.insertTx(tx, blk); err != nil {
			return err
		}
	}

	mysql.memBlocksRW.Lock()
	mysql.memBlocks.PushBack(blk)
	cnt := mysql.memBlocks.Len()
	if mysql.elemChan == nil {
		mysql.elemChan = mysql.memBlocks.Front()
	}
	mysql.memBlocksRW.Unlock()

	if cnt > confirmed {
		mysql.writeBlockChan <- mysql.elemChan
		mysql.elemChan = mysql.elemChan.Next()
	}
	return nil
}

//InsertPendingTxs 新增内存池
func (mysql *Mysql) InsertPendingTxs(txs []*Transaction) error {
	pendingBlock := &Block{}
	t := time.Now()
	defer func() {
		log.Infof("[MYSQL] insert pending block %d elpase %s", pendingBlock.Height, time.Now().Sub(t))
	}()

	pendingBlock.addressInfos = make(map[string]*AddressInfo)
	for _, tx := range txs {
		if err := mysql.insertTx(tx, pendingBlock); err != nil {
			return err
		}
	}

	mysql.pendingBlockRW.Lock()
	mysql.pendingBlock = pendingBlock
	mysql.pendingBlockRW.Unlock()
	return nil
}

// DeleteBlock 删除区块
func (mysql *Mysql) DeleteBlock(blk *Block) error {
	mysql.memBlocksRW.Lock()
	if elem := mysql.memBlocks.Back(); elem != nil {
		lblk := elem.Value.(*Block)
		if lblk.Height != blk.Height {
			panic(fmt.Sprintf("mismatch height %d %d", lblk.Height, blk.Height))
		}
		mysql.memBlocks.Remove(elem)
	} else {
		panic("uncompleted")
	}
	mysql.memBlocksRW.Unlock()
	return nil
}

// GetBlockChainFromDB 获取最新区块
func (mysql *Mysql) GetBlockChainFromDB() (*Block, error) {
	sqlstr := "SELECT i_height, i_created, s_hash, s_prevhash FROM t_blockchain"
	blk := &Block{}
	row := mysql.db.QueryRow(sqlstr)
	err := row.Scan(&blk.Height, &blk.Time, &blk.ID, &blk.PrevID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return blk, nil
}

// GetBlockChain 获取最新区块
func (mysql *Mysql) GetBlockChain() (*Block, error) {
	//从缓存区块中查找
	mysql.memBlocksRW.RLock()
	if elem := mysql.memBlocks.Back(); elem != nil {
		blk := elem.Value.(*Block)
		mysql.memBlocksRW.RUnlock()
		return blk, nil
	}
	mysql.memBlocksRW.RUnlock()
	return mysql.GetBlockChainFromDB()
}

// GetTransactionsByAddressFromDB 获取指定地址的交易
func (mysql *Mysql) GetTransactionsByAddressFromDB(addr string, skip int64, num int64) ([]*Transaction, error) {
	if num == 0 {
		return nil, nil
	}
	sqlStrH := fmt.Sprintf("SELECT s_hash FROM t_history where s_address='%s' order by id desc limit %d, %d", addr, skip, num)
	rowsH, err := mysql.db.Query(sqlStrH)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer rowsH.Close()

	hashes := []string{}
	for rowsH.Next() {
		var hash string
		err := rowsH.Scan(&hash)
		if err != nil {
			return nil, err
		}
		hashes = append(hashes, fmt.Sprintf("'%s'", hash))
	}

	if len(hashes) == 0 {
		return nil, nil
	}

	sqlStr := fmt.Sprintf("SELECT s_hash, s_ins, s_outs, i_created, i_height, s_fee, i_size FROM t_transaction where s_hash in(%s) order by id desc;",
		strings.Join(hashes, ","))
	rows, err := mysql.db.Query(sqlStr)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	txs := []*Transaction{}
	for rows.Next() {
		tx := &Transaction{
			Fee: big.NewInt(0),
		}
		var ins, outs, fee string
		err := rows.Scan(&tx.ID, &ins, &outs, &tx.Time, &tx.Height, &fee, &tx.Size)
		if err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(ins), &tx.Ins)
		json.Unmarshal([]byte(outs), &tx.Outs)
		tx.Fee.SetString(fee, 10)
		txs = append(txs, tx)
	}
	return txs, nil
}

// GetTransactionsByAddress 获取指定地址的交易
func (mysql *Mysql) GetTransactionsByAddress(addr string, pagenum int64, pagesize int64) ([]*Transaction, error) {
	skip := pagenum * pagesize
	txs := []*Transaction{}
	//从内存池区块中查找
	mysql.pendingBlockRW.RLock()
	if mysql.pendingBlock != nil {
		if addrInfo, ok := mysql.pendingBlock.addressInfos[addr]; ok {
			for cnt := len(addrInfo.HTxs); cnt > 0; cnt-- {
				if skip == 0 {
					txs = append(txs, addrInfo.Txs[addrInfo.HTxs[cnt-1]])
					if pagenum--; pagenum == 0 {
						mysql.pendingBlockRW.RUnlock()
						return txs, nil
					}
				} else {
					skip--
				}
			}
		}
	}
	mysql.pendingBlockRW.RUnlock()

	//从缓存区块中查找
	mysql.memBlocksRW.RLock()
	for elem := mysql.memBlocks.Back(); elem != nil; elem = elem.Prev() {
		blk := elem.Value.(*Block)
		if addrInfo, ok := blk.addressInfos[addr]; ok {
			for cnt := len(addrInfo.HTxs); cnt > 0; cnt-- {
				if skip == 0 {
					txs = append(txs, addrInfo.Txs[addrInfo.HTxs[cnt-1]])
					if pagenum--; pagenum == 0 {
						mysql.memBlocksRW.RUnlock()
						return txs, nil
					}
				} else {
					skip--
				}
			}
		}
	}
	mysql.memBlocksRW.RUnlock()

	ttxs, err := mysql.GetTransactionsByAddressFromDB(addr, skip, pagenum-int64(len(txs)))
	return append(txs, ttxs...), err
}

// GetAccountByAddress 获取指定地址的余额
func (mysql *Mysql) GetAccountByAddress(addr string, pending bool) (*AddressInfo, error) {
	if pending {
		//从内存池区块中查找
		if mysql.pendingBlock != nil {
			mysql.pendingBlockRW.RLock()
			if addrInfo, ok := mysql.pendingBlock.addressInfos[addr]; ok {
				mysql.pendingBlockRW.RUnlock()
				return addrInfo, nil
			}
			mysql.pendingBlockRW.RUnlock()
		}
	}

	//从缓存区块中查找
	mysql.memBlocksRW.RLock()
	for elem := mysql.memBlocks.Back(); elem != nil; elem = elem.Prev() {
		blk := elem.Value.(*Block)
		if addrInfo, ok := blk.addressInfos[addr]; ok {
			mysql.memBlocksRW.RUnlock()
			return addrInfo, nil
		}
	}
	mysql.memBlocksRW.RUnlock()

	return mysql.GetAccountByAddressFromDB(addr)
}

// GetAccountByAddressFromDB  获取指定地址的信息
func (mysql *Mysql) GetAccountByAddressFromDB(addr string) (*AddressInfo, error) {
	sqlStr := fmt.Sprintf("SELECT s_value FROM t_address where s_address='%s'", addr)
	addrInfo := &AddressInfo{
		Amount: big.NewInt(0),
	}
	row := mysql.db.QueryRow(sqlStr)
	var amount string
	err := row.Scan(&amount)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	addrInfo.Amount.SetString(amount, 10)
	return addrInfo, nil
}

// AddMonitorAddress 新增监控地址
func (mysql *Mysql) AddMonitorAddress(address string) error {
	if addressInfo, err := mysql.GetAccountByAddressFromDB(strings.ToLower(address)); addressInfo != nil || err != nil {
		return err
	}
	sqlStr := fmt.Sprintf("INSERT INTO t_address(s_address, s_value) values('%s', '%s');", address, big.NewInt(0))
	return mysql.execSQL(sqlStr)
}

// IsMonitorAddress 新增监控地址
func (mysql *Mysql) IsMonitorAddress(address string) bool {
	return true
	// addressInfo, _ := mysql.GetAccountByAddressFromDB(strings.ToLower(address))
	// return addressInfo != nil
}

func (mysql *Mysql) insertTx(tx *Transaction, blk *Block) error {
	for _, in := range tx.Ins {
		for _, address := range in.Addresses {
			addressInfo, ok := blk.addressInfos[address]
			if !ok {
				addressInfo = &AddressInfo{
					Amount: big.NewInt(0),
					Txs:    make(map[string]*Transaction),
				}
				if addrInfo, _ := mysql.GetAccountByAddress(address, false); addrInfo != nil {
					addressInfo.Amount.SetBytes(addrInfo.Amount.Bytes())
				}
				blk.addressInfos[address] = addressInfo
			}
			//花掉
			addressInfo.Amount = new(big.Int).Sub(addressInfo.Amount, in.Value)
			//新增历史记录
			if _, ok := addressInfo.Txs[tx.ID]; !ok {
				addressInfo.Txs[tx.ID] = tx
				addressInfo.HTxs = append(addressInfo.HTxs, tx.ID)
			}
			if addrs := strings.Split(address, "-"); len(addrs) == 2 {
				mysql.tokenChan <- addrs[1]
			}
		}
	}

	for _, out := range tx.Outs {
		for _, address := range out.Addresses {
			addressInfo, ok := blk.addressInfos[address]
			if !ok {
				addressInfo = &AddressInfo{
					Amount: big.NewInt(0),
					Txs:    make(map[string]*Transaction),
				}
				if addrInfo, _ := mysql.GetAccountByAddress(address, false); addrInfo != nil {
					addressInfo.Amount.SetBytes(addrInfo.Amount.Bytes())
				}
				blk.addressInfos[address] = addressInfo
			}
			//新增
			addressInfo.Amount = new(big.Int).Add(addressInfo.Amount, out.Value)
			//新增历史记录
			if _, ok := addressInfo.Txs[tx.ID]; !ok {
				addressInfo.Txs[tx.ID] = tx
				addressInfo.HTxs = append(addressInfo.HTxs, tx.ID)
			}
		}
	}
	return nil
}

// GetTokenInfo 获取token信息
func (mysql *Mysql) GetTokenInfo(token string) (*TokenInfo, error) {
	sqlStr := fmt.Sprintf("SELECT s_address, s_name, s_symbol, i_decimal FROM t_tokeninfo where s_address='%s'", token)
	tokenInfo := &TokenInfo{}
	row := mysql.db.QueryRow(sqlStr)
	err := row.Scan(&tokenInfo.Address, &tokenInfo.Name, &tokenInfo.Symbol, &tokenInfo.Decimal)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return tokenInfo, nil

}

// InsertOrUpdateTokenInfo 更新token信息
func (mysql *Mysql) InsertOrUpdateTokenInfo(token string) (*TokenInfo, error) {
	if tokenInfo, err := mysql.GetTokenInfo(token); tokenInfo != nil || err != nil {
		return tokenInfo, err
	}

	tokenInfo := &TokenInfo{}
	tokenInfo.Address = token
	name, err := mysql.RPC.GetTokenName(token)
	if err != nil {
		return nil, err
	}
	tokenInfo.Name = name
	symbol, err := mysql.RPC.GetTokenSymbol(token)
	if err != nil {
		return nil, err
	}
	tokenInfo.Symbol = symbol
	decimal, err := mysql.RPC.GetTokenDecimal(token)
	if err != nil {
		return nil, err
	}
	tokenInfo.Decimal = decimal.Int64()

	if len(tokenInfo.Name) > 0 && tokenInfo.Name[0] < 32 {
		tokenInfo.Name = tokenInfo.Name[1:]
	}

	if len(tokenInfo.Symbol) > 0 && tokenInfo.Symbol[0] < 32 {
		tokenInfo.Symbol = tokenInfo.Symbol[1:]
	}

	sqlStr := fmt.Sprintf("INSERT INTO t_tokeninfo(s_address, s_name, s_symbol, i_decimal) values('%s','%s','%s',%d)",
		tokenInfo.Address, Escape(tokenInfo.Name), Escape(tokenInfo.Symbol), tokenInfo.Decimal)

	return tokenInfo, mysql.execSQL(sqlStr)
}

// GetTokenInfosByAddress 获取token信息列表
func (mysql *Mysql) GetTokenInfosByAddress(addr string) ([]*TokenInfo, error) {
	sqlStr := fmt.Sprintf("SELECT s_address FROM t_address where s_address like '%s_%%'", addr)

	rows, err := mysql.db.Query(sqlStr)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tokenInfos := []*TokenInfo{}
	for rows.Next() {
		var address string
		err := rows.Scan(&address)
		if err != nil {
			return nil, err
		}
		token := strings.Replace(address, fmt.Sprintf("%s-", addr), "", -1)
		tokenInfo, err := mysql.InsertOrUpdateTokenInfo(token)
		if err != nil {
			log.Errorf("gettoken info ---- %s", err)
			continue
		}
		tokenInfos = append(tokenInfos, tokenInfo)
	}
	return tokenInfos, nil
}

func Escape(sql string) string {
	dest := make([]byte, 0, 2*len(sql))
	var escape byte
	for i := 0; i < len(sql); i++ {
		c := sql[i]

		escape = 0

		switch c {
		case 0: /* Must be escaped for 'mysql' */
			escape = '0'
			break
		case '\n': /* Must be escaped for logs */
			escape = 'n'
			break
		case '\r':
			escape = 'r'
			break
		case '\\':
			escape = '\\'
			break
		case '\'':
			escape = '\''
			break
		case '"': /* Better safe than sorry */
			escape = '"'
			break
		case '\032': /* This gives problems on Win32 */
			escape = 'Z'
		}

		if escape != 0 {
			dest = append(dest, '\\', escape)
		} else {
			dest = append(dest, c)
		}
	}

	return string(dest)
}
