package botmaid

import (
	"fmt"
	"strings"
)

// Bot includes some information of a bot.
type Bot struct {
	ID string

	API API

	Self *User

	BotMaid *BotMaid
}

type dbMaster struct {
	ID       int64
	BotID    string
	UserName string
}

type dbTestChat struct {
	ID       int64
	BotID    string
	ChatType string
	ChatID   int64
}

// IsMaster checks if a user is master of the bot.
func (b *Bot) IsMaster(u User) bool {
	m := dbMaster{}
	err := b.BotMaid.DB.QueryRow("SELECT * FROM masters WHERE bot_id = $1 AND username = $2", b.ID, u.UserName).Scan(&m.ID, &m.BotID, &m.UserName)
	if err != nil {
		return false
	}
	return true
}

// IsTestChat checks if a chat is a test chat of the bot.
func (b *Bot) IsTestChat(p Chat) bool {
	t := dbTestChat{}
	err := b.BotMaid.DB.QueryRow("SELECT * FROM testchats WHERE bot_id = $1 AND chat_type = $2 AND chat_id = $3", b.ID, p.Type, p.ID).Scan(&t.ID, &t.BotID, &t.ChatType, &t.ChatID)
	if err != nil {
		return false
	}
	return true
}

// Platform returns a string showing the platform of the bot.
func (b *Bot) Platform() string {
	switch b.API.(type) {
	case *CoolqHTTPAPI:
		return "QQ"
	case *TelegramBotAPI:
		return "Telegram"
	}

	return "Unknown Platform"
}

// At returns a string to mention someone in a message.
func (b *Bot) At(u *User) []string {
	switch b.API.(type) {
	case *CoolqHTTPAPI:
		return []string{fmt.Sprintf("[CQ:at,qq=%s]", u.UserName), fmt.Sprintf("@%s", u.NickName)}
	case *TelegramBotAPI:
		return []string{fmt.Sprintf("@%s", u.UserName)}
	}

	return []string{fmt.Sprintf("@%s", u.UserName)}
}

// BeAt checks if a message of an update is mentioning the bot.
func (b *Bot) BeAt(u *Update) bool {
	switch b.API.(type) {
	case *CoolqHTTPAPI:
		if (strings.Contains(u.Message.Text, fmt.Sprintf("[CQ:at,qq=%s]", b.Self.UserName)) || strings.Contains(u.Message.Text, fmt.Sprintf("@%s", b.Self.NickName))) && b.extractCommand(u) == "" {
			return true
		}
	case *TelegramBotAPI:
		if strings.Contains(u.Message.Text, fmt.Sprintf("@%s", b.Self.UserName)) && b.extractCommand(u) == "" {
			return true
		}
	}

	return false
}

// UserNameFromAt returns the UserName of the user in the mention query.
func (b *Bot) UserNameFromAt(s string) string {
	switch b.API.(type) {
	case *CoolqHTTPAPI:
		if fmt.Sprintf("[CQ:at,qq=%s]", s[10:len(s)-1]) == s {
			return s[10 : len(s)-1]
		}
	case *TelegramBotAPI:
		if fmt.Sprintf("@%s", s[1:]) == s {
			return s[1:]
		}
	}

	return ""
}

// SendBack sends a text back to the origin chat.
func (b *Bot) SendBack(u *Update, t string) (Update, error) {
	return b.API.Send(Update{
		Message: &Message{
			Text: t,
		},
		Chat: u.Chat,
	})
}

// SendBackImage sends a image back to the origin chat.
func (b *Bot) SendBackImage(u *Update, t string) (Update, error) {
	return b.API.Send(Update{
		Message: &Message{
			Image: t,
		},
		Chat: u.Chat,
	})
}

// SendBackAudio sends a audio back to the origin chat.
func (b *Bot) SendBackAudio(u *Update, t string) (Update, error) {
	return b.API.Send(Update{
		Message: &Message{
			Audio: t,
		},
		Chat: u.Chat,
	})
}
