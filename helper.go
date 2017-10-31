package mailbot

import (
	"fmt"
	"io"
	"mime"
	"net/mail"

	"github.com/satori/go.uuid"
	"golang.org/x/text/encoding/htmlindex"
)

func newWordDecoder() (dec *mime.WordDecoder) {
	dec = new(mime.WordDecoder)
	dec.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		e, err := htmlindex.Get(charset)
		if err != nil {
			return nil, err
		}
		return e.NewDecoder().Reader(input), err
	}
	return
}

func newAddressParser() (parser *mail.AddressParser) {
	parser = new(mail.AddressParser)
	parser.WordDecoder = newWordDecoder()
	return
}

func GenerateMessageID(user string) string {
	return fmt.Sprintf("<%s-%s>", uuid.NewV4().String(), user)
}
