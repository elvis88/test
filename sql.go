package main

var initSQL = `
CREATE TABLE IF NOT EXISTS t_blockchain (
  id int(11) NOT NULL PRIMARY KEY AUTO_INCREMENT,
  i_height int(11) NOT NULL comment '区块高度',
  i_created int(11) NOT NULL comment '区块时间',
  s_hash char(100) NOT NULL comment '区块哈希',
  s_prevhash char(100) comment '前区块哈希'
);

CREATE TABLE IF NOT EXISTS t_transaction (
  id int(11) NOT NULL PRIMARY KEY AUTO_INCREMENT,
  s_hash char(100) NOT NULL comment '交易哈希',
  s_ins longtext NOT NULL comment '交易输入',
  s_outs longtext NOT NULL comment '交易输出',
  s_fee char(100) NOT NULL comment '交易手续费',
  i_size int(11) NOT NULL comment '交易大小',
  i_created int(11) NOT NULL comment '交易入账时间',
  i_height int(11) NOT NULL comment '交易所在区块高度'
);

CREATE TABLE IF NOT EXISTS t_address (
  id int(11) NOT NULL PRIMARY KEY AUTO_INCREMENT,
  s_address char(100) NOT NULL comment '账户地址',
  s_value char(100) NOT NULL comment '账户金额',
  UNIQUE INDEX (s_address)
);

CREATE TABLE IF NOT EXISTS t_tokeninfo (
	id INTEGER(11) PRIMARY KEY AUTO_INCREMENT,
	s_address char(100) NOT NULL,
	s_name longtext NOT NULL,
	s_symbol longtext NOT NULL,
	i_decimal INTEGER(11) NOT NULL,
	UNIQUE INDEX(s_address)
);

CREATE TABLE IF NOT EXISTS t_history (
  id int(11) NOT NULL PRIMARY KEY AUTO_INCREMENT,
  s_address char(100) NOT NULL comment '账户地址',
  s_hash char(100) NOT NULL comment '交易哈希',
  INDEX (s_address),
  UNIQUE (s_address, s_hash)
);
`
