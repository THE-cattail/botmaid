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

// IsMaster checks if a user is master of the bot.
func (b *Bot) IsMaster(u User) bool {
	f, _ := b.BotMaid.Redis.SIsMember("master_"+b.ID, u.ID).Result()
	return f
}

// IsTestChat checks if a chat is a test chat of the bot.
func (b *Bot) IsTestChat(p Chat) bool {
	f, _ := b.BotMaid.Redis.SIsMember("testchat_"+b.ID, p.ID).Result()
	return f
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
		return []string{fmt.Sprintf("tg://user?id=%v", u.ID), fmt.Sprintf("@%s", u.UserName)}
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
