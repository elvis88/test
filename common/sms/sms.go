package sms

// The client of the sms service of Aliyun(阿理云短信服务客户端)
import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/mail"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	Request    *Request
	GatewayURL string
	Client     *http.Client
}

func New(gatewayURL string) *Client {
	client := new(Client)
	client.Request = &Request{}
	client.GatewayURL = gatewayURL
	client.Client = &http.Client{}
	return client
}

func (client *Client) Execute(accessKeyID, accessKeySecret, phoneNumbers, signName, templateCode, templateParam string) (*Response, error) {
	err := client.Request.SetParamsValue(accessKeyID, phoneNumbers, signName, templateCode, templateParam)
	if err != nil {
		return nil, err
	}
	endpoint, err := client.Request.BuildEndpoint(accessKeySecret, client.GatewayURL)
	if err != nil {
		return nil, err
	}

	request, _ := http.NewRequest("GET", endpoint, nil)
	response, err := client.Client.Do(request)
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	result := new(Response)
	err = json.Unmarshal(body, result)

	result.RawResponse = body
	return result, err
}

// SendCode
func SendCode(phoneNumbers string, code string) bool {
	if strings.Contains(phoneNumbers, "@") {
		from := mail.Address{Name: "Uranus", Address: "service@uranus.io"}
		sender, err := NewSMTPClient("smtp.exmail.qq.com:465", from, "Ct#P@ssw0rd")
		if err != nil {
			panic("Failed to send email: " + err.Error())
		}
		msg := &Message{
			Subject: "verification code",
			Content: bytes.NewBufferString(fmt.Sprintf(`Hello,<br>
			[Uranus] code: %v.<br>
			This code only works for Uranus and expires after 30 minutes.<br>
			If this is not your operation, please change your login password immediately.<br><br>
			Thank you for choosing Uranus!<br>
			Uranus official<br>`, code)),
			To: []string{phoneNumbers},
		}
		err = sender.Send(msg, false)
		fmt.Println("sms err", err)
		return err == nil
	} else {
		gatewayURL := "http://dysmsapi.aliyuncs.com/"
		accessKeyID := "LTAIbsoSIx8TvPrj"
		accessKeySecret := "AJX8O7LzNSvP5N2bzP39UjnBkR9nk2"
		signName := "中云动力科技有限公司"
		templateCode := "SMS_149421847"
		if !strings.HasPrefix(phoneNumbers, "86") {
			templateCode = "SMS_149417154"
		}
		templateParam := fmt.Sprintf("{\"code\":\"%s\"}", code)
		client := New(gatewayURL)
		result, err := client.Execute(accessKeyID, accessKeySecret, fmt.Sprintf("00%v", phoneNumbers), signName, templateCode, templateParam)
		if err != nil {
			panic("Failed to send Message: " + err.Error())
		}

		resultJSON, err := json.Marshal(result)
		if err != nil {
			panic(err)
		}
		if result.IsSuccessful() {
			fmt.Println("[SMS] A SMS is sent successfully:", phoneNumbers, string(resultJSON))
		} else {
			fmt.Println("[SMS] Failed to send a SMS:", phoneNumbers, string(resultJSON))
		}
		return result.IsSuccessful()
	}
}

// MakeCode 生成验证码
func MakeCode() (code string) {
	code = strconv.Itoa(rand.New(rand.NewSource(time.Now().UnixNano())).Intn(899999) + 100000)
	return
}

// VailMobile 验证手机号
func VailMobile(mobile string) error {
	if strings.Contains(mobile, "@") {
		return nil
	}
	if strings.Compare(strings.ToLower(mobile), "test") == 0 {
		return nil
	}
	c, err := regexp.Compile("^[0-9]+$")
	if err != nil {
		panic("regexp error")
	}
	if !c.MatchString(mobile) {
		return errors.New("incorrect phone number")
	}
	// if len(mobile) < 11 {
	// 	return errors.New("手机号码位数不正确")
	// }
	// reg, err := regexp.Compile("^1[3-8][0-9]{9}$")
	// if err != nil {
	// 	panic("regexp error")
	// }
	// if !reg.MatchString(mobile) {
	// 	return errors.New("手机号码格式不正确")
	// }
	return nil
}

// VailCode 验证验证码
func VailCode(code string) error {
	if len(code) != 6 {
		return errors.New("incorrect number of verification codes")
	}
	c, err := regexp.Compile("^[0-9]{6}$")
	if err != nil {
		panic("regexp error")
	}
	if !c.MatchString(code) {
		return errors.New("incorrect verification codes")
	}
	return nil
}
