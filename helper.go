package mailbot

import (
	"fmt"
	"io"
	"mime"
	"net/mail"

	"github.com/satori/go.uuid"
	"golang.org/x/text/encoding/htmlindex"
)

var wordDecoder = func() (dec *mime.WordDecoder) {
	dec = new(mime.WordDecoder)
	dec.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		e, err := htmlindex.Get(charset)
		if err != nil {
			return nil, err
		}
		return e.NewDecoder().Reader(input), err
	}
	return
}()

var addressParser = func() (parser *mail.AddressParser) {
	parser = new(mail.AddressParser)
	parser.WordDecoder = wordDecoder
	return
}()

// GenerateMessageID generates a unique message-id.
func GenerateMessageID(user string) string {
	return fmt.Sprintf("<%s-%s>", uuid.NewV4().String(), user)
}

// BuildMail combines header and body part into email payload.
func BuildMail(header mail.Header, body []byte) (msg []byte) {
	for k, vs := range header {
		for _, v := range vs {
			msg = append(msg, []byte(k+": "+v+"\r\n")...)

		}
	}
	msg = append(msg, []byte("\r\n")...)
	msg = append(msg, body...)
	return
}
