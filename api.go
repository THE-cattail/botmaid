package botmaid

import (
	"time"
)

// API is an interface including some common behaviors for APIs.
type API interface {
	Pull(*PullConfig) (UpdateChannel, ErrorChannel)
	Push(*Update) (*Update, error)

	Platform() string
	ParseUserID(u *Update, s string) (int64, error)
	ats(u *User) []string
}

// Update is a struct for an update of APIs.
type Update struct {
	ID   int64
	Type string

	Chat    *Chat
	User    *User
	Message *Message

	Time time.Time

	Bot *Bot
}

// UpdateChannel is a channel for saving updates.
type UpdateChannel chan *Update

// ErrorChannel is a channel for saving errors.
type ErrorChannel chan error

// PullConfig is a struct for pulling.
//
// Limit decides the number of updates pulled once.
// Timeout decides the timeout of long polling.
// RetryWaitingTime decides decides the time waiting after pulling an error.
type PullConfig struct {
	Limit            int
	Timeout          int
	RetryWaitingTime time.Duration
}

// Message is a struct for a message of an update.
type Message struct {
	ID   int64
	Type string

	Content string

	Args    []string
	Command string
}

// Chat is a struct for a chat.
type Chat struct {
	ID   int64
	Type string

	Title string

	Bot *Bot
}

// User is a struct for a user.
type User struct {
	ID int64

	UserName string
	NickName string

	Bot *Bot
}
