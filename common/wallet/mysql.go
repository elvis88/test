package wallet

import (
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	// mysql

	_ "github.com/go-sql-driver/mysql"
	bip39 "github.com/tyler-smith/go-bip39"
)

//Mysql implement mysql
type Mysql struct {
	DBName string
	DBUser string
	DBPWD  string
	DBHost string
	db     *sql.DB
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

// GetWallet find wallet
func (mysql *Mysql) GetWallet(name string) (*Wallet, error) {
	if strings.Compare(strings.ToLower(name), "test") == 0 {
		mnemonic := "abandon amount liar amount expire adjust cage candy arch gather drum buyer"
		entropy, _ := bip39.EntropyFromMnemonic(mnemonic)
		return NewWallet("test", hex.EncodeToString(entropy), nil)
	}

	sqlStr := fmt.Sprintf("SELECT s_entropy, s_meta FROM t_user where s_name='%s'", name)
	var entropy, meta string
	row := mysql.db.QueryRow(sqlStr)
	err := row.Scan(&entropy, &meta)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	ometa := make(map[string]interface{})
	json.Unmarshal([]byte(meta), &ometa)
	ebts, _ := hex.DecodeString(entropy)
	bts, err := Decrypt([]byte(ebts), RightPadBytes([]byte(name), 16))
	if err != nil {
		return nil, err
	}
	return NewWallet(name, string(bts), ometa)
}

// InsertOrGetWallet find wallet
func (mysql *Mysql) InsertOrGetWallet(name string) (*Wallet, error) {
	wallet, err := mysql.GetWallet(name)
	if wallet != nil || err != nil {
		return wallet, err
	}
	if wallet, err := NewWallet(name, NewHexEntropy(), nil); err != nil {
		return nil, err
	} else if err := mysql.UpdateWallet(wallet); err != nil {
		return nil, err
	} else {
		return wallet, nil
	}
}

// UpdateWalletName update wallet name
func (mysql *Mysql) UpdateWalletName(name, newname string) error {
	wallet, err := mysql.GetWallet(name)
	if wallet == nil || err != nil {
		return err
	}
	meta, _ := json.Marshal(wallet.Meta)
	sqlStr := fmt.Sprintf("UPDATE t_user set s_name='%s', s_entropy='%s', s_meta='%s' where s_name='%s'", newname, hex.EncodeToString(Encrypt([]byte(wallet.HexEntory), RightPadBytes([]byte(newname), 16))), string(meta), wallet.Name)
	return mysql.execSQL(sqlStr)
}

// UpdateWallet insert or update wallet
func (mysql *Mysql) UpdateWallet(wallet *Wallet) error {
	meta, _ := json.Marshal(wallet.Meta)
	sqlStr := fmt.Sprintf("REPLACE INTO t_user(s_name, s_entropy, s_meta) values('%s','%s','%s')", wallet.Name, hex.EncodeToString(Encrypt([]byte(wallet.HexEntory), RightPadBytes([]byte(wallet.Name), 16))), string(meta))
	return mysql.execSQL(sqlStr)
}

// GetWallet find wallet
func (mysql *Mysql) GetWallets() ([]*Wallet, error) {
	mnemonic := "abandon amount liar amount expire adjust cage candy arch gather drum buyer"
	entropy, _ := bip39.EntropyFromMnemonic(mnemonic)
	twlt, _ := NewWallet("test", hex.EncodeToString(entropy), nil)

	sqlStr := fmt.Sprintf("SELECT s_name,s_entropy, s_meta FROM t_user")
	rows, err := mysql.db.Query(sqlStr)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	wlts := []*Wallet{}
	wlts = append(wlts, twlt)
	for rows.Next() {
		var name, entropy, meta string
		err := rows.Scan(&name, &entropy, &meta)
		if err != nil {
			return nil, err
		}
		ometa := make(map[string]interface{})
		json.Unmarshal([]byte(meta), &ometa)
		ebts, _ := hex.DecodeString(entropy)
		bts, err := Decrypt([]byte(ebts), RightPadBytes([]byte(name), 16))
		if err != nil {
			continue
		}
		wlt, err := NewWallet(name, string(bts), ometa)
		if err != nil {
			continue
		}
		wlts = append(wlts, wlt)
	}
	return wlts, nil
}
