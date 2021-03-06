package mailbot

import (
	"crypto/tls"
	"errors"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/wxdao/go-imap/imap"
)

var (
	// ErrInterrupted means the loop is interrupted by signal.
	ErrInterrupted = errors.New("interrupted")
)

// Daemon ...
type Daemon struct {
	config      *Config
	client      *imap.Client
	handlers    []HandlerFunc
	smallestSeq int
}

// Serve serves.
func (d *Daemon) Serve() (err error) {
	if d.config.IMAPUseTLS {
		d.client, err = imap.DialTLS(d.config.IMAPAddress, &tls.Config{
			ServerName: strings.Split(d.config.IMAPAddress, ":")[0],
		})
		if err != nil {
			return
		}
	} else {
		d.client, err = imap.Dial(d.config.IMAPAddress)
		if err != nil {
			return
		}
		// try STARTTLS
		err = d.client.StartTLS(strings.Split(d.config.IMAPAddress, ":")[0])
	}
	defer d.client.Close()

	if d.config.Debug {
		d.client.Debug = os.Stderr
	}

	_, err = d.client.Capability()
	if err != nil {
		return
	}

	interrupted := make(chan os.Signal, 1)
	signal.Notify(interrupted, os.Interrupt, os.Kill)
	updated := make(chan int, 10)
	d.client.UpdateCallback = func() {
		updated <- 1
	}

	err = d.client.Login(d.config.User, d.config.Pass)
	if err != nil {
		return
	}

	info, err := d.client.Select("INBOX")
	if err != nil {
		return
	}
	if d.config.IgnoreExisting {
		d.smallestSeq = info.Exists + 1
	} else {
		d.smallestSeq = 1
	}

	wait := make(chan int, 10)

	for {
		var criteria string
		if d.config.UnseenOnly {
			criteria = strconv.Itoa(d.smallestSeq) + ":* UNSEEN"
		} else {
			criteria = strconv.Itoa(d.smallestSeq) + ":*"
		}
		seqs, err := d.client.Search(criteria)
		if err != nil {
			return err
		}
		for _, seq := range seqs {
			result, err := d.client.FetchRFC822([]int{seq}, !d.config.MarkSeen)
			if err != nil {
				return err
			}
			d.smallestSeq = seqs[len(seqs)-1]
			wait <- 1
			go func() {
				d.handleNewEmails(result, false)
				<-wait
			}()
		}
		go d.client.Idle()
		select {
		case <-updated:
			d.client.Done()
		case <-time.After(time.Minute * 5):
			d.client.Done()
		case <-interrupted:
			return ErrInterrupted
		}
	}
}

// RegisterHandler registers a handler.
func (d *Daemon) RegisterHandler(fun HandlerFunc) {
	d.handlers = append(d.handlers, fun)
}

// NewDaemon ...
func NewDaemon(config *Config) *Daemon {
	return &Daemon{
		config: config,
	}
}
