# Mailbot

Mailbot receives emails with IMAP and dispatch them to handlers registerd by user.

[GoDoc](https://godoc.org/github.com/wxdao/mailbot)

## Example

A program that detects mail with subject "What time is it" and replies the time.

```go
package main

import (
	"log"
	"mime"
	"net/mail"
	"strings"
	"time"

	"github.com/wxdao/mailbot"
)

func main() {
	config := &mailbot.Config{
		IMAPAddress: "imap.mail.com:993",
		IMAPUseTLS:  true,

		SMTPAddress: "smtp.mail.com:994",
        SMTPUseTLS:  true,

        IgnoreExisting: false,
        MarkSeen: true,
        UnseenOnly: true,

		User: "bot@mail.com",
		Pass: "I'm a bot.",
	}

	daemon := mailbot.NewDaemon(config)

	daemon.RegisterHandler(func(m *mailbot.Mail) {
		reply := "Not sure what you are looking for."

		text := strings.Join(m.Texts, "\n")
		if strings.Contains(strings.ToLower(m.Subject), "what time is it") {
			if strings.Contains(text, "UTC") {
				reply = time.Now().UTC().String()
			} else {
				reply = time.Now().String()
			}
		}

		header := mail.Header{}
		header["Subject"] = []string{mime.QEncoding.Encode("utf-8", "Re: "+m.Subject)}
		header["From"] = []string{config.User}
		header["Message-Id"] = []string{mailbot.GenerateMessageID(config.User)}
		header["Message-Id"] = []string{mailbot.GenerateMessageID(config.User)}
		header["In-Reply-To"] = []string{m.MessageID}
		header["To"] = []string{m.FromAddr.String()}
		if replyTo := m.Header.Get("Reply-To"); replyTo != "" {
			header["To"] = []string{replyTo}
		}
		err := daemon.SendPlainTextMail(header, reply)
		if err != nil {
			log.Println(err)
		}
	})

	for {
		if err := daemon.Serve(); err == mailbot.ErrInterrupted {
			break
		}
		log.Println(err)
	}
}

```