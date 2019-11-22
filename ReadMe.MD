### 1.1 功能描述
获取账户详情

### 1.2 请求说明
> 请求方式：POST <br>
请求URL ：[getaddressinfo](#) 

### 1.3 请求参数
字段       |字段类型       |字段说明
------------|-----------|-----------
phone           |string        | 手机号
token_address   |string        | token合约地址(可选,指定特定的合约地址)
```json  
{
    "phone":"test",
    "token_address":""
}
```

### 1.4 返回结果
字段       |字段类型        |字段说明
------------|-----------|-----------
data       |object          |账户详情
errCode    |int             |错误状态码
errMsg     |string          |错误描述
###### 账户详情
字段       |字段类型        |字段说明
------------|-----------|-----------
address     |string         |账户地址
token_address   |string     |token合约地址
amount      |bigint         |余额
gas_price   |bigint         |推荐燃料单价
decimal     |int            |币种精度
coin        |string         |币种名称
###### 错误状态码  
状态码       |说明
------------|-----------
2000       |成功
2001       |请求参数有误
2002       |后台执行有误
```json  
{
    "data": {
        "address": "0x83f1caAdaBeEC2945b73087F803d404F054Cc2B7",
        "token_address": "",
        "amount": 357000000000000000000,
        "gas_price": 18000000000,
        "coin": "urac",
        "decimal": 18
    },
    "errCode": 2000,
    "errMsg": "",
    "hash": "9cd5ea9a0afa618292d3dc65ced46584"
}
```

### 2.1 功能描述
获取历史交易信息。

### 2.2 请求说明
> 请求方式：POST <br>
请求URL ：[gethistoryinfo](#)

### 2.3 请求参数
字段       |字段类型       |字段说明
------------|-----------|-----------
phone         |string         | 手机号
token_address |string         |token合约地址(可选,指定特定的合约地址)
page_num      |int            |页码(默认0)
page_size     |int            |个数(默认20)
```json  
{
    "phone":"test",
    "token_address":"",
    "page_num":0,
    "page_size": 20
}
```

### 2.4 返回结果
字段       |字段类型        |字段说明
------------|-----------|-----------
data       |array           |交易详情数组
errCode    |int             |错误状态码
errMsg     |string          |错误描述
###### 交易详情
字段       |字段类型        |字段说明
------------|-----------|-----------
hash        |string         |交易哈希
confirmations |int          |交易确认数
timestamp   |int64          |交易时间
from        |string         |交易发送方
to          |string         |交易接收方
value       |bigint         |交易金额
fee         |bigint         |交易手续费
signature   |string         |交易签名
status      |int            |交易状态码(0 确认中 1 已确认)
size        |int            |交易燃料消费
height      |int            |交易高度
tvalue      |bigint         |金额变动

###### 错误状态码  
状态码       |说明
------------|-----------
2000       |成功
2001       |请求参数有误
2002       |后台执行有误
```json  
{
  "data":  [
    ],
  "errCode": "2000",
  "errMsg": ""
}
```

错误状态码  
状态码       |说明
------------|-----------
2000       |成功！
2001       |请求参数有误
2002       |后台执行有误

### 3.1 功能描述
获取交易验证码

### 3.2 请求说明
> 请求方式：POST <br>
请求URL ：[confirm](#) 

### 3.3 请求参数
字段       |字段类型       |字段说明
------------|-----------|-----------
phone        |string        | 手机号
```json  
{
  "phone":"test"
}
```

### 3.4 返回结果
字段       |字段类型        |字段说明
------------|-----------|-----------
data       |string          |状态描述
errCode    |int             |错误状态码
errMsg     |string          |错误描述
```json  
{
  "data": "ok",
  "errCode": "2000",
  "errMsg": ""
}
```

### 4.1 功能描述
发送交易

### 4.2 请求说明
> 请求方式：POST <br>
请求URL ：[send](#) 

### 4.3 请求参数
字段       |字段类型       |字段说明
------------|-----------|-----------
phone        |string       | 手机号
token_address  |string     |token合约地址(可选,指定特定的合约地址)
code        |string        |验证码
order       |array          |订单列表

###### 订单详情
字段       |字段类型       |字段说明
------------|-----------|-----------
id          |string         |订单号
to          |string        |接收方
value       |string        |接收金额
gas         |int           |燃料大小（有默认值）
gas_price   |string        |燃料单价(有默认值)
```json  
{
	"phone": "test",
	"token_address": "",
	"code": "123456",
	"order": [{
		"id": "0000000000000001",
		"to": "0x83f1caAdaBeEC2945b73087F803d404F054Cc2B7",
		"value": 1000000,
		"gas": 21000,
		"gas_price": 9000000000
	}]
}
```

### 4.4 返回结果
字段       |字段类型        |字段说明
------------|-----------|-----------
data       |map          |交易订单-哈希
errCode    |int             |错误状态码
errMsg     |string          |错误描述
###### 错误状态码  
状态码       |说明
------------|-----------
2000       |成功
2001       |请求参数有误
2002       |后台执行有误
```json  
{
  "data": ["0000000000000001":"0xb31ef3f08551c0b8c763fbfcf1ec84b18158119222980813f2b1085732d87fde"],
  "errCode": "2000",
  "errMsg": ""
}
```

### 5.1 功能描述
根据哈希获取交易详情。

### 5.2 请求说明
> 请求方式：POST <br>
请求URL ：[gettxinfo](#)

### 5.3 请求参数
字段       |字段类型       |字段说明
------------|-----------|-----------
hash         |string         | 交易哈希
```json  
{
    "hash":"0xb31ef3f08551c0b8c763fbfcf1ec84b18158119222980813f2b1085732d87fde"
}
```

### 5.4 返回结果
字段       |字段类型        |字段说明
------------|-----------|-----------
data       |object           |交易详情
errCode    |int             |错误状态码
errMsg     |string          |错误描述
###### 交易详情
字段       |字段类型        |字段说明
------------|-----------|-----------
hash        |string         |交易哈希
confirmations |int          |交易确认数
timestamp   |int64          |交易时间
from        |string         |交易发送方
to          |string         |交易接收方
value       |bigint         |交易金额
fee         |bigint         |交易手续费
signature   |string         |交易签名
status      |int            |交易状态码(0 确认中 1 已确认)
size        |int            |交易燃料消费
height      |int            |交易高度
tvalue      |bigint         |金额变动
###### 错误状态码  
状态码       |说明
------------|-----------
2000       |成功
2001       |请求参数有误
2002       |后台执行有误
```json  
{
  "data":  {
    },
  "errCode": "2000",
  "errMsg": ""
}
```