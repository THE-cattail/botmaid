package botmaid

import (
	"errors"
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
	ID     int64
	BotID  string
	UserID int64
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
	err := b.BotMaid.DB.QueryRow("SELECT * FROM masters WHERE bot_id = $1 AND user_id = $2", b.ID, u.ID).Scan(&m.ID, &m.BotID, &m.UserID)
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
		return []string{fmt.Sprintf("[CQ:at,qq=%v]", u.ID), fmt.Sprintf("@%s", u.NickName)}
	case *TelegramBotAPI:
		return []string{fmt.Sprintf("tg://user?id=%v", u.ID)}
	}

	return []string{fmt.Sprintf("@%v", u.ID)}
}

// BeAt checks if a message of an update is mentioning the bot.
func (b *Bot) BeAt(u *Update) bool {
	switch b.API.(type) {
	case *CoolqHTTPAPI:
		if (strings.Contains(u.Message.Text, fmt.Sprintf("[CQ:at,qq=%v]", b.Self.ID)) || strings.Contains(u.Message.Text, fmt.Sprintf("@%s", b.Self.NickName))) && b.extractCommand(u) == "" {
			return true
		}
	case *TelegramBotAPI:
		if strings.Contains(u.Message.Text, fmt.Sprintf("@%s", b.Self.UserName)) && b.extractCommand(u) == "" {
			return true
		}
	}

	return false
}

// Reply replies a message back.
func (b *Bot) Reply(u *Update, s ...string) (Update, error) {
	if len(s) < 1 || len(s) > 2 {
		return Update{}, errors.New("Invalid number of arguments")
	}
	if len(s) == 1 || s[1] == "Text" {
		return b.API.Push(Update{
			Message: &Message{
				Text: s[0],
			},
			Chat: u.Chat,
		})
	}
	if s[1] == "Image" {
		return b.API.Push(Update{
			Message: &Message{
				Image: s[0],
			},
			Chat: u.Chat,
		})
	}
	if s[1] == "Audio" {
		return b.API.Push(Update{
			Message: &Message{
				Audio: s[0],
			},
			Chat: u.Chat,
		})
	}
	return Update{}, errors.New("Invalid type of message")
}
