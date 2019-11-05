package botmaid

import (
	"errors"
	"fmt"
	"strings"
)

// Bot includes some information of a bot.
type Bot struct {
	ID string

	API *API

	Self *User
}

// Platform returns a string showing the platform of the bot.
func (b *Bot) Platform() string {
	switch (*b.API).(type) {
	case *APICqhttp:
		return "QQ"
	case *APITelegramBot:
		return "Telegram"
	}

	return "Unknown Platform"
}

// IsMaster checks if a user is master of the bot.
func (bm *BotMaid) IsMaster(u *User) bool {
	return bm.Redis.SIsMember("master_"+u.Bot.ID, u.ID).Val()
}

// IsBanned checks if a user has been banned.
func (bm *BotMaid) IsBanned(u *User) bool {
	return bm.Redis.SIsMember("ban_"+u.Bot.ID, u.ID).Val()
}

// BeAt checks if a message of an update is mentioning the bot.
func (bm *BotMaid) BeAt(u *Update) bool {
	switch (*u.Bot.API).(type) {
	case *APICqhttp:
		if (strings.Contains(u.Message.Text, fmt.Sprintf("[CQ:at,qq=%v]", u.Bot.Self.ID)) || strings.Contains(u.Message.Text, fmt.Sprintf("@%v", u.Bot.Self.NickName))) && bm.extractCommand(u) == "" {
			return true
		}
	case *APITelegramBot:
		if strings.Contains(u.Message.Text, fmt.Sprintf("@%v", u.Bot.Self.UserName)) && bm.extractCommand(u) == "" {
			return true
		}
	}

	return false
}

// At returns a string to mention someone in a message.
func At(u *User) []string {
	switch (*u.Bot.API).(type) {
	case *APICqhttp:
		return []string{fmt.Sprintf("[CQ:at,qq=%v]", u.ID), fmt.Sprintf("@%v", u.NickName)}
	case *APITelegramBot:
		return []string{fmt.Sprintf("tg://user?id=%v", u.ID), fmt.Sprintf("@%v", u.UserName)}
	}

	return []string{fmt.Sprintf("@%v", u.ID)}
}

// Reply replies a message back.
func Reply(u *Update, s ...string) (*Update, error) {
	if len(s) < 1 || len(s) > 2 {
		return nil, errors.New("Invalid number of arguments")
	}
	if len(s) == 1 || s[1] == "Text" {
		return (*u.Bot.API).Push(&Update{
			Message: &Message{
				Text: s[0],
			},
			Chat: u.Chat,
		})
	}
	if s[1] == "Image" {
		return (*u.Bot.API).Push(&Update{
			Message: &Message{
				Image: s[0],
			},
			Chat: u.Chat,
		})
	}
	if s[1] == "Audio" {
		return (*u.Bot.API).Push(&Update{
			Message: &Message{
				Audio: s[0],
			},
			Chat: u.Chat,
		})
	}
	return nil, errors.New("Invalid type of message")
}

// In checks if the element is in the slice.
func In(a interface{}, s ...interface{}) bool {
	if len(s) == 1 {
		if _, ok := s[0].([]interface{}); ok {
			for _, v := range s[0].([]interface{}) {
				if v == a {
					return true
				}
			}
			return false
		}
	}

	for _, v := range s {
		if v == a {
			return true
		}
	}
	return false
}

// ListToString convert the list to a string.
func ListToString(list []string, format string, separator string, andWord string) string {
	if len(list) < 1 {
		return ""
	}
	if len(list) == 1 {
		return fmt.Sprintf(format, list[0])
	}
	ret := fmt.Sprintf(format, list[0])
	for i := 1; i < len(list)-1; i++ {
		ret += separator + fmt.Sprintf(format, list[i])
	}
	ret += andWord + fmt.Sprintf(format, list[len(list)-1])
	return ret
}
