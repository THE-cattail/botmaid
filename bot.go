package botmaid

import (
	"fmt"
	"strings"

	"github.com/catsworld/api"
	"github.com/catsworld/cqhttp"
	"github.com/catsworld/tgbot"
)

// Bot includes some information of a bot.
type Bot struct {
	ID string

	API api.API

	Self *api.User

	BotMaid *BotMaid
}

type dbMaster struct {
	ID       int64
	BotID    string
	UserName string
}

type dbTestPlace struct {
	ID        int64
	BotID     string
	PlaceType string
	PlaceID   int64
}

// IsMaster checks if a user is master of the bot.
func (b *Bot) IsMaster(u api.User) bool {
	m := dbMaster{}
	err := b.BotMaid.DB.QueryRow("SELECT * FROM masters WHERE bot_id = $1 AND username = $2", b.ID, u.UserName).Scan(&m.ID, &m.BotID, &m.UserName)
	if err != nil {
		return false
	}
	return true
}

// IsTestPlace checks if a place is test place of the bot.
func (b *Bot) IsTestPlace(p api.Place) bool {
	t := dbTestPlace{}
	err := b.BotMaid.DB.QueryRow("SELECT * FROM testplaces WHERE bot_id = $1 AND place_type = $2 AND place_id = $3", b.ID, p.Type, p.ID).Scan(&t.ID, &t.BotID, &t.PlaceType, &t.PlaceID)
	if err != nil {
		return false
	}
	return true
}

// Platform returns a string showing the platform of the bot.
func (b *Bot) Platform() string {
	switch b.API.(type) {
	case *cqhttp.API:
		return "QQ"
	case *tgbot.API:
		return "Telegram"
	}

	return "Unknown Platform"
}

// At returns a string to mention someone in a message.
func (b *Bot) At(u *api.User) []string {
	switch b.API.(type) {
	case *cqhttp.API:
		return []string{fmt.Sprintf("[CQ:at,qq=%s]", u.UserName), fmt.Sprintf("@%s", u.NickName)}
	case *tgbot.API:
		return []string{fmt.Sprintf("@%s", u.UserName)}
	}

	return []string{fmt.Sprintf("@%s", u.UserName)}
}

// BeAt checks if a message of an event is mentioning the bot.
func (b *Bot) BeAt(e *api.Event) bool {
	switch b.API.(type) {
	case *cqhttp.API:
		if (strings.Contains(e.Message.Text, fmt.Sprintf("[CQ:at,qq=%s]", b.Self.UserName)) || strings.Contains(e.Message.Text, fmt.Sprintf("@%s", b.Self.NickName))) && b.extractCommand(e) == "" {
			return true
		}
	case *tgbot.API:
		if strings.Contains(e.Message.Text, fmt.Sprintf("@%s", b.Self.UserName)) && b.extractCommand(e) == "" {
			return true
		}
	}

	return false
}

// UserNameFromAt returns the UserName of the user in the mention query.
func (b *Bot) UserNameFromAt(s string) string {
	switch b.API.(type) {
	case *cqhttp.API:
		if fmt.Sprintf("[CQ:at,qq=%s]", s[10:len(s)-1]) == s {
			return s[10 : len(s)-1]
		}
	case *tgbot.API:
		if fmt.Sprintf("@%s", s[1:]) == s {
			return s[1:]
		}
	}

	return ""
}

// PushBack pushes a message back to the origin place.
func (b *Bot) PushBack(e *api.Event, m *api.Message) (api.Event, error) {
	return b.API.Push(api.Event{
		Message: m,
		Place:   e.Place,
	})
}
