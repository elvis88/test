package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Jeffail/gabs"
)

var (
	httpClient = &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 100,
		},
		Timeout: time.Second * 500,
	}
)

type RPCRequest struct {
	Jsonrpc string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

func NewRPCRequest(jsonrpc string, method string, param ...interface{}) *RPCRequest {
	r := new(RPCRequest)
	r.Jsonrpc = jsonrpc
	r.Method = method
	r.Params = make([]interface{}, 0)
	r.Params = append(r.Params, param...)
	return r
}

func SendRPCRequst(host string, rpcRequest *RPCRequest) (*gabs.Container, error) {
	var buff bytes.Buffer
	if err := json.NewEncoder(&buff).Encode(rpcRequest); err != nil {
		return nil, fmt.Errorf("SendRPCRequst EncodeRequest error --- %s", err)
	}

	req, _ := http.NewRequest("POST", host, &buff)
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("SendRPCRequst Post %s error --- %s(%s)", host, err, buff.String())
	}
	defer resp.Body.Close()
	jsonParsed, err := gabs.ParseJSONBuffer(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("SendRPCRequst ParseJSONBuffer error --- %s(%s)(%s)", err, buff.String(), jsonParsed.String())
	}
	return jsonParsed, nil
}

func SendRPCRequstWithAuth(host string, username string, password string, rpcRequest *RPCRequest) (*gabs.Container, error) {
	var buff bytes.Buffer
	if err := json.NewEncoder(&buff).Encode(rpcRequest); err != nil {
		return nil, fmt.Errorf("SendRPCRequst EncodeRequest error --- %s", err)
	}

	req, _ := http.NewRequest("POST", host, &buff)
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(username, password)
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("SendRPCRequst Post %s error --- %s(%s)", host, err, buff.String())
	}
	defer resp.Body.Close()

	// if resp.StatusCode != 200 {
	// 	return nil, fmt.Errorf("SendRPCRequst Post %s error --- status code %d(%s)", host, resp.StatusCode, buff.String())
	// }

	jsonParsed, err := gabs.ParseJSONBuffer(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("SendRPCRequst ParseJSONBuffer error --- %s(%s)(%s)", err, buff.String(), jsonParsed.String())
	}
	return jsonParsed, nil
}
