package sms

// The library of the sms service of Aliyun(阿理云短信服务库)
import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base32"
	"encoding/base64"
	"errors"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/pborman/uuid"
)

// Response
type Response struct {
	// The raw response from server.
	RawResponse []byte `json:"-"`
	/* Response body */
	RequestId string `json:"RequestId"`
	Code      string `json:"Code"`
	Message   string `json:"Message"`
	BizId     string `json:"BizId"`
}

func (m *Response) IsSuccessful() bool {
	return m.Code == "OK"
}

// Request
type Request struct {
	//system parameters
	AccessKeyID      string
	Timestamp        string
	Format           string
	SignatureMethod  string
	SignatureVersion string
	SignatureNonce   string
	Signature        string

	//business parameters
	Action          string
	Version         string
	RegionID        string
	PhoneNumbers    string
	SignName        string
	TemplateCode    string
	TemplateParam   string
	SmsUpExtendCode string
	OutID           string
}

var encoding = base32.NewEncoding("ybndrfg8ejkmcpqxot1uwisza345h897")

// NewID uuid
func NewID() string {
	var b bytes.Buffer
	encoder := base32.NewEncoder(encoding, &b)
	encoder.Write(uuid.NewRandom())
	encoder.Close()
	b.Truncate(26)
	return b.String()
}

// SetParamsValue init
func (req *Request) SetParamsValue(accessKeyID, phoneNumbers, signName, templateCode, templateParam string) error {
	req.AccessKeyID = accessKeyID
	now := time.Now()
	local, err := time.LoadLocation("GMT")
	if err != nil {
		return err
	}
	req.Timestamp = now.In(local).Format("2006-01-02T15:04:05Z")
	req.Format = "json"
	req.SignatureMethod = "HMAC-SHA1"
	req.SignatureVersion = "1.0"
	req.SignatureNonce = NewID()

	req.Action = "SendSms"
	req.Version = "2017-05-25"
	req.RegionID = "cn-hangzhou"
	req.PhoneNumbers = phoneNumbers
	req.SignName = signName
	req.TemplateCode = templateCode
	req.TemplateParam = templateParam
	req.SmsUpExtendCode = "90999"
	req.OutID = "abcdefg"
	return nil
}

// IsValid valid
func (req *Request) IsValid() error {
	if len(req.AccessKeyID) == 0 {
		return errors.New("AccessKeyId required")
	}

	if len(req.PhoneNumbers) == 0 {
		return errors.New("PhoneNumbers required")
	}

	if len(req.SignName) == 0 {
		return errors.New("SignName required")
	}

	if len(req.TemplateCode) == 0 {
		return errors.New("TemplateCode required")
	}

	if len(req.TemplateParam) == 0 {
		return errors.New("TemplateParam required")
	}

	return nil
}

// BuildEndpoint
func (req *Request) BuildEndpoint(accessKeySecret, gatewayURL string) (string, error) {
	var err error
	if err = req.IsValid(); err != nil {
		return "", err
	}
	// common params
	systemParams := make(map[string]string)
	systemParams["SignatureMethod"] = req.SignatureMethod
	systemParams["SignatureNonce"] = req.SignatureNonce
	systemParams["AccessKeyId"] = req.AccessKeyID
	systemParams["SignatureVersion"] = req.SignatureVersion
	systemParams["Timestamp"] = req.Timestamp
	systemParams["Format"] = req.Format

	// business params
	businessParams := make(map[string]string)
	businessParams["Action"] = req.Action
	businessParams["Version"] = req.Version
	businessParams["RegionId"] = req.RegionID
	businessParams["PhoneNumbers"] = req.PhoneNumbers
	businessParams["SignName"] = req.SignName
	businessParams["TemplateParam"] = req.TemplateParam
	businessParams["TemplateCode"] = req.TemplateCode
	businessParams["SmsUpExtendCode"] = req.SmsUpExtendCode
	businessParams["OutId"] = req.OutID
	// generate signature and sorted  query
	sortQueryString, signature := generateQueryStringAndSignature(businessParams, systemParams, accessKeySecret)
	return gatewayURL + "?Signature=" + signature + sortQueryString, nil
}

func generateQueryStringAndSignature(businessParams map[string]string, systemParams map[string]string, accessKeySecret string) (string, string) {
	keys := make([]string, 0)
	allParams := make(map[string]string)
	for key, value := range businessParams {
		keys = append(keys, key)
		allParams[key] = value
	}

	for key, value := range systemParams {
		keys = append(keys, key)
		allParams[key] = value
	}

	sort.Strings(keys)

	sortQueryStringTmp := ""
	for _, key := range keys {
		rstkey := specialURLEncode(key)
		rstval := specialURLEncode(allParams[key])
		sortQueryStringTmp = sortQueryStringTmp + "&" + rstkey + "=" + rstval
	}

	sortQueryString := strings.Replace(sortQueryStringTmp, "&", "", 1)
	stringToSign := "GET" + "&" + specialURLEncode("/") + "&" + specialURLEncode(sortQueryString)

	sign := sign(accessKeySecret+"&", stringToSign)
	signature := specialURLEncode(sign)
	return sortQueryStringTmp, signature
}

func specialURLEncode(value string) string {
	rstValue := url.QueryEscape(value)
	rstValue = strings.Replace(rstValue, "+", "%20", -1)
	rstValue = strings.Replace(rstValue, "*", "%2A", -1)
	rstValue = strings.Replace(rstValue, "%7E", "~", -1)
	return rstValue
}

func sign(accessKeySecret, sortquerystring string) string {
	h := hmac.New(sha1.New, []byte(accessKeySecret))
	h.Write([]byte(sortquerystring))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}
