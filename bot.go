package botmaid

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
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

// ParseUserID parses the ID of the User in the At string.
func (bm *BotMaid) ParseUserID(u *Update, s string) (int64, error) {
	if u.Bot.Platform() == "QQ" {
		if strings.HasPrefix(s, "[CQ:at,qq=") && strings.HasSuffix(s, "]") {
			start := 10
			end := strings.LastIndex(s, "]")
			s = s[start:end]

			id, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				return 0, fmt.Errorf("Invalid At string: %v", err)
			}
			return id, nil
		}
	}

	if u.Bot.Platform() == "Telegram" {
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(s))
		if err == nil {
			s := doc.Find("a").AttrOr("href", "")
			if strings.HasPrefix(s, "tg://user?id=") {
				s = s[13:]

				id, err := strconv.ParseInt(s, 10, 64)
				if err != nil {
					return 0, fmt.Errorf("Invalid At string: %v", err)
				}
				return id, nil
			}
		}

		if strings.HasPrefix(s, "@") {
			s = bm.Redis.HGet("telegramUsers", s[1:]).Val()

			id, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				return 0, fmt.Errorf("Invalid At string: %v", err)
			}
			return id, nil
		}
	}

	return 0, errors.New("Invalid At string")
}

func (bm *BotMaid) ats(u *User) []string {
	if u.Bot.Platform() == "QQ" {
		return []string{fmt.Sprintf("[CQ:at,qq=%v]", u.ID)}
	}

	if u.Bot.Platform() == "Telegram" {
		return []string{fmt.Sprintf("<a href=\"tg://user?id=%v\">%v</a>", u.ID, u.NickName), fmt.Sprintf("@%v", u.UserName)}
	}

	return []string{""}
}

// BeAt checks if a message of an update is mentioning the bot.
func (bm *BotMaid) BeAt(u *Update) bool {
	if bm.extractCommand(u) != "" {
		return false
	}

	for _, v := range bm.ats(u.Bot.Self) {
		if strings.Contains(u.Message.Content, v) {
			return true
		}
	}

	return false
}

// At returns a string to mention someone in a message.
func (bm *BotMaid) At(u *User) string {
	return bm.ats(u)[0]
}

// Reply replies a message back.
func (bm *BotMaid) Reply(u *Update, s ...string) (*Update, error) {
	if len(s) < 1 || len(s) > 2 {
		return nil, errors.New("Invalid number of arguments")
	}

	if len(s) == 1 || s[1] == "Text" {
		return (*u.Bot.API).Push(&Update{
			Message: &Message{
				Content: s[0],
			},
			Chat: u.Chat,
		})
	}
	if s[1] == "Image" {
		return (*u.Bot.API).Push(&Update{
			Message: &Message{
				Type:    "Image",
				Content: s[0],
			},
			Chat: u.Chat,
		})
	}
	if s[1] == "Audio" {
		return (*u.Bot.API).Push(&Update{
			Message: &Message{
				Type:    "Audio",
				Content: s[0],
			},
			Chat: u.Chat,
		})
	}

	for len(bm.history[u.Chat.ID]) > 0 && time.Now().Sub(bm.history[u.Chat.ID][0]) > time.Second {
		bm.history[u.Chat.ID] = bm.history[u.Chat.ID][1:]
	}
	if len(bm.history[u.Chat.ID]) >= 5 {
		bm.Redis.SAdd(fmt.Sprintf("ban_%v", u.Bot.ID), fmt.Sprintf("%v|%v", u.Chat.ID, u.Chat.Title))
	}

	return nil, errors.New("Invalid type of message")
}

// In checks if the element is in the slice.
func In(a interface{}, s ...interface{}) bool {
	if len(s) == 1 && reflect.TypeOf(s[0]).Kind() == reflect.Slice {
		t := reflect.ValueOf(s[0])
		for i := 0; i < t.Len(); i++ {
			if t.Index(i).Interface() == a {
				return true
			}
		}
		return false
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
