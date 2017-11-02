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
	"regexp"
	"strings"
	"time"

	"github.com/wxdao/go-imap/imap"

	"golang.org/x/text/encoding/htmlindex"
)

var regexParenthese, _ = regexp.Compile(` \(.*\)`)

// HandlerFunc ...
type HandlerFunc func(m *Mail)

// Mail contains raw mail data and some extracted essential info from it.
type Mail struct {
	Header    mail.Header
	Result    *imap.FetchResult
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

func (d *Daemon) handleNewEmails(result map[int]*imap.FetchResult, headerOnly bool) {
	for _, fetchResult := range result {
		msg, err := mail.ReadMessage(bytes.NewReader(fetchResult.Data))
		if err != nil {
			if d.config.Debug {
				log.Println("msg:", err)
			}
			continue
		}

		messageID := msg.Header.Get("Message-ID")
		inReplyTo := msg.Header.Get("In-Reply-To")

		fromAddr, err := addressParser.Parse(msg.Header.Get("From"))
		if err != nil {
			if d.config.Debug {
				log.Println("from:", err)
			}
		}

		// remove (GMT+08:00) like suffix
		date, err := mail.ParseDate(regexParenthese.ReplaceAllString(msg.Header.Get("Date"), ""))
		if err != nil {
			if d.config.Debug {
				log.Println("date:", err)
			}
		}

		subject, err := addressParser.WordDecoder.DecodeHeader(msg.Header.Get("Subject"))
		if err != nil {
			if d.config.Debug {
				log.Println("subject:", err)
			}
			subject = msg.Header.Get("Subject")
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
			handler(&Mail{msg.Header, fetchResult, messageID, inReplyTo, fromAddr, subject, date, texts, parts})
		}
	}
}
