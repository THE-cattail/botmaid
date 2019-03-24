package botmaid

import "time"

// API is an interface including some common behaviors for APIs.
//
// GetUpdates always gets updates and errors into the channels with a given config.
//
// Push always pushes an update and returns it back if existing.
//
// Delete always deletes a specific update.
type API interface {
	GetUpdates(GetUpdatesConfig) (UpdateChannel, ErrorChannel)
	Push(Update) (Update, error)
}

// Update is a struct for an update of APIs.
type Update struct {
	ID int64

	Type string

	Time time.Time

	Chat *Chat

	User *User

	Message *Message
}

// UpdateChannel is a channel for saving updates.
type UpdateChannel chan Update

// ErrorChannel is a channel for saving errors.
type ErrorChannel chan error

// GetUpdatesConfig is a struct for getting updates.
//
// Limit decides the number of updates pulled once.
// Timeout decides the timeout of long polling.
// RetryWaitingTime decides decides the time waiting after pulling an error.
type GetUpdatesConfig struct {
	Limit            int
	Timeout          int
	RetryWaitingTime time.Duration
}

// Message is a struct for a message of an update.
type Message struct {
	ID int64

	Text  string
	Image string
	Audio string

	Args []string
}

// Chat is a struct for a chat.
type Chat struct {
	ID int64

	Type string

	Title string
}

// User is a struct for a user.
type User struct {
	ID       int64
	UserName string

	NickName string
}
