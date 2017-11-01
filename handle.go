package mailbot

import (
	"bytes"
	"encoding/base64"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"net/textproto"
	"strings"
	"time"

	"golang.org/x/text/encoding/htmlindex"
)

// HandlerFunc ...
type HandlerFunc func(m *Mail)

// Mail contains raw mail data and some extracted essential info from it.
type Mail struct {
	Header    mail.Header
	Data      []byte
	MessageID string
	InReplyTo string
	FromAddr  *mail.Address
	Subject   string
	Date      time.Time
	Texts     []string
	Parts     []*Part
}

// Part is like multipart.Part but it provides raw data bytes.
type Part struct {
	Header textproto.MIMEHeader
	Data   []byte
}

func readBody(h textproto.MIMEHeader, br io.Reader) (texts []string, parts []*Part) {
	mt, params, err := mime.ParseMediaType(h.Get("Content-Type"))
	if err != nil {
		return
	}
	switch h.Get("Content-Transfer-Encoding") {
	case "base64":
		br = base64.NewDecoder(base64.RawStdEncoding, br)
	case "quoted-printable":
		br = quotedprintable.NewReader(br)
	}
	if mt == "text/plain" {
		if c, ok := params["charset"]; ok {
			e, err := htmlindex.Get(c)
			if err != nil {
				return
			}
			br = e.NewDecoder().Reader(br)
			d, _ := ioutil.ReadAll(br)
			texts = append(texts, string(d))
		}
	} else if strings.HasPrefix(mt, "multipart/") {
		mr := multipart.NewReader(br, params["boundary"])
		for {
			mp, err := mr.NextPart()
			if err != nil {
				break
			}
			ntexts, nparts := readBody(mp.Header, mp)
			texts = append(texts, ntexts...)
			parts = append(parts, nparts...)
		}
	} else {
		d, _ := ioutil.ReadAll(br)
		parts = append(parts, &Part{
			Header: h,
			Data:   d,
		})
	}
	return
}

func (d *Daemon) handleNewEmails(data map[int][]byte, headerOnly bool) {
	addressParser := newAddressParser()
	for _, mailData := range data {
		msg, err := mail.ReadMessage(bytes.NewReader(mailData))
		if err != nil {
			continue
		}

		messageID := msg.Header.Get("message-id")
		fromAddr, err := addressParser.Parse(msg.Header.Get("from"))
		if err != nil {
			continue
		}

		inReplyTo, err := addressParser.WordDecoder.DecodeHeader(msg.Header.Get("in-reply-to"))
		if err != nil {
			continue
		}

		date, err := mail.ParseDate(msg.Header.Get("date"))
		if err != nil {
			continue
		}

		subject, err := addressParser.WordDecoder.DecodeHeader(msg.Header.Get("subject"))
		if err != nil {
			continue
		}

		var texts []string
		var parts []*Part
		if !headerOnly {
			texts, parts = readBody(textproto.MIMEHeader(msg.Header), msg.Body)
		}

		if d.config.Debug {
			log.Println("received:", messageID,
				"\nin-reply-to:", inReplyTo,
				"\nfrom:", fromAddr.String(),
				"\nsubject:", subject,
				"\ndate:", date.Local(),
				"\ntexts:", texts,
			)
		}

		for _, handler := range d.handlers {
			go handler(&Mail{msg.Header, mailData, messageID, inReplyTo, fromAddr, subject, date, texts, parts})
		}
	}
}
