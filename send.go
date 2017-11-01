package mailbot

import (
	"crypto/tls"
	"encoding/base64"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
)

type unencryptedAuth struct {
	smtp.Auth
}

func (a unencryptedAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	s := *server
	s.TLS = true
	return a.Auth.Start(&s)
}

// SendMail sends an email.
func (d *Daemon) SendMail(header mail.Header, body []byte) (err error) {
	var conn net.Conn
	tlsConfig := &tls.Config{
		ServerName: strings.Split(d.config.SMTPAddress, ":")[0],
	}
	if d.config.SMTPUseTLS {
		conn, err = tls.Dial("tcp", d.config.SMTPAddress, tlsConfig)
	} else {
		conn, err = net.Dial("tcp", d.config.SMTPAddress)
	}
	if err != nil {
		return
	}
	smtpClient, err := smtp.NewClient(conn, tlsConfig.ServerName)
	if err != nil {
		return
	}
	defer smtpClient.Close()

	if !d.config.SMTPUseTLS {
		// try STARTTLS
		if ok, _ := smtpClient.Extension("STARTTLS"); ok {
			err = smtpClient.StartTLS(tlsConfig)
			if err != nil {
				return
			}
		}
	}

	err = smtpClient.Auth(unencryptedAuth{smtp.PlainAuth("", d.config.User, d.config.Pass, tlsConfig.ServerName)})
	if err != nil {
		return
	}
	err = smtpClient.Mail(d.config.User)
	if err != nil {
		return
	}

	tos, _ := addressParser.ParseList(header.Get("To"))
	for _, address := range tos {
		err = smtpClient.Rcpt(address.Address)
		if err != nil {
			return
		}
	}

	ccs, _ := addressParser.ParseList(header.Get("Cc"))
	for _, address := range ccs {
		err = smtpClient.Rcpt(address.Address)
		if err != nil {
			return
		}
	}

	bccs, _ := addressParser.ParseList(header.Get("Bcc"))
	for _, address := range bccs {
		err = smtpClient.Rcpt(address.Address)
		if err != nil {
			return
		}
	}
	delete(header, "Bcc")

	w, err := smtpClient.Data()
	if err != nil {
		return
	}

	for k, vs := range header {
		for _, v := range vs {
			_, err = w.Write([]byte(k + ": " + v + "\r\n"))
			if err != nil {
				return
			}
		}
	}
	_, err = w.Write([]byte("\r\n"))
	if err != nil {
		return
	}

	_, err = w.Write(body)
	if err != nil {
		return
	}
	err = w.Close()
	if err != nil {
		return
	}
	err = smtpClient.Quit()
	return
}

// SendPlainTextMail sends a simple text email.
func (d *Daemon) SendPlainTextMail(header mail.Header, text string) (err error) {
	header["Content-Transfer-Encoding"] = []string{"base64"}
	header["Content-Type"] = []string{"text/plain; charset=utf-8"}

	body := []byte(base64.StdEncoding.EncodeToString([]byte(text)))
	if err != nil {
		return
	}

	err = d.SendMail(header, body)
	return
}
