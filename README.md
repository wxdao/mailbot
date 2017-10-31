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

	daemon.RegisterHandler(func(data []byte, messageID string, inReplyTo string, fromAddr *mail.Address, subject string, date time.Time, texts []string, parts []*mailbot.Part) {
		reply := "Not sure what you are looking for."

		text := strings.Join(texts, "\n")
		if subject == "What time is it" {
			if strings.Contains(text, "UTC") {
				reply = time.Now().UTC().String()
			} else {
				reply = time.Now().String()
			}
		}

		header := map[string]string{}
		header["Subject"] = mime.QEncoding.Encode("utf-8", "Re: "+subject)
		header["From"] = config.User
		header["Message-Id"] = mailbot.GenerateMessageID(config.User)
		header["In-Reply-To"] = messageID
		header["To"] = fromAddr.String()
		err := daemon.SendPlainTextMail(fromAddr.Address, header, reply)
		if err != nil {
			log.Println(err)
		}
	})

	log.Println(daemon.Serve())
}

```