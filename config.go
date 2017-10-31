package mailbot

// Config ...
type Config struct {
	IMAPAddress string
	IMAPUseTLS  bool

	SMTPAddress string
	SMTPUseTLS  bool

	User string
	Pass string

	// IgnoreExisting determines whether to ignore existing unseen mails.
	IgnoreExisting bool
	// MarkSeen determines whether to mark fetched mails seen.
	MarkSeen bool
	// UnseenOnly determines whether to look for unseen mails only.
	UnseenOnly bool

	Debug bool
}
