package sms

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
)

// Message 邮件发送数据
type Message struct {
	Subject   string            // 标题
	Content   io.Reader         // 支持html的消息主体
	To        []string          // 邮箱地址
	Extension map[string]string // 发送邮件消息体扩展项
}

// NewSMTPClient 创建基于smtp的邮件发送实例(使用PlainAuth)
// addr 服务器地址
// from 发送者
// authPwd 验证密码
// 如果创建实例发生异常，则返回错误
func NewSMTPClient(addr string, from mail.Address, authPwd string) (*SMTPClient, error) {
	smtpCli := &SMTPClient{
		addr: addr,
		from: from,
	}
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	smtpCli.auth = smtp.PlainAuth("", from.Address, authPwd, host)
	return smtpCli, nil
}

// SMTPClient 使用smtp发送邮件
type SMTPClient struct {
	addr string
	from mail.Address
	auth smtp.Auth
}

// Send 发送邮件
func (client *SMTPClient) Send(msg *Message, isMass bool) (err error) {
	if isMass {
		err = client.massSend(msg)
	} else {
		err = client.oneSend(msg)
	}
	return
}

// AsyncSend 异步发送邮件
func (client *SMTPClient) AsyncSend(msg *Message, isMass bool, handle func(err error)) error {
	go func() {
		err := client.Send(msg, isMass)
		handle(err)
	}()
	return nil
}

// oneSend 一对一按顺序发送
func (client *SMTPClient) oneSend(msg *Message) error {
	for _, addr := range msg.To {
		header := client.getHeader(msg.Subject)
		header["To"] = addr
		if msg.Extension != nil {
			for k, v := range msg.Extension {
				header[k] = v
			}
		}
		data := client.getData(header, msg.Content)
		err := client.SendMailUsingTLS(client.addr, client.auth, client.from.Address, []string{addr}, data)
		if err != nil {
			return err
		}
	}
	return nil
}

// massSend 群发邮件
func (client *SMTPClient) massSend(msg *Message) error {
	header := client.getHeader(msg.Subject)
	if msg.Extension != nil {
		for k, v := range msg.Extension {
			header[k] = v
		}
	}
	data := client.getData(header, msg.Content)
	return client.SendMailUsingTLS(client.addr, client.auth, client.from.Address, msg.To, data)
}

func (client *SMTPClient) SendMailUsingTLS(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	conn, err := tls.Dial("tcp", client.addr, nil)
	if err != nil {
		return err
	}
	defer conn.Close()

	host, _, _ := net.SplitHostPort(client.addr)
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}

	if ok, _ := c.Extension("AUTH"); ok {
		if err = c.Auth(client.auth); err != nil {
			return err
		}
	}

	if err = c.Mail(from); err != nil {
		return err
	}

	for _, addr := range to {
		if err = c.Rcpt(addr); err != nil {
			return err
		}
	}

	w, err := c.Data()
	if err != nil {
		return err
	}

	_, err = w.Write(msg)
	if err != nil {
		return err
	}

	err = w.Close()
	if err != nil {
		return err
	}
	return c.Quit()
}

func (client *SMTPClient) getHeader(subject string) map[string]string {
	header := make(map[string]string)
	header["From"] = client.from.String()
	header["Subject"] = mime.QEncoding.Encode("utf-8", subject)
	header["Mime-Version"] = "1.0"
	header["Content-Type"] = "text/html;charset=utf-8"
	header["Content-Transfer-Encoding"] = "Quoted-Printable"
	return header
}

func (client *SMTPClient) getData(header map[string]string, body io.Reader) []byte {
	buf := new(bytes.Buffer)
	for k, v := range header {
		fmt.Fprintf(buf, "%s: %s\r\n", k, v)
	}
	fmt.Fprintf(buf, "\r\n")
	io.Copy(buf, body)
	return buf.Bytes()
}
