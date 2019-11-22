package wallet

var initSQL = `
CREATE TABLE IF NOT EXISTS t_user (
  id int(11) NOT NULL PRIMARY KEY AUTO_INCREMENT,
  s_name char(100) NOT NULL comment '用户标识',
  s_entropy longtext NOT NULL comment '用户商',
  s_meta longtext NOT NULL comment '其它信息',
  UNIQUE INDEX (s_name)
);
`
