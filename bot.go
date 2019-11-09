package botmaid

import (
	"errors"
	"fmt"
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

func (bm *BotMaid) antiReplyLoop(u *Update) {
	for len(bm.history[u.Chat.ID]) > 0 && time.Now().Sub(bm.history[u.Chat.ID][0]) > time.Second {
		bm.history[u.Chat.ID] = bm.history[u.Chat.ID][1:]
	}
	if len(bm.history[u.Chat.ID]) >= 5 {
		bm.Redis.SAdd(fmt.Sprintf("ban_%v", u.Bot.ID), fmt.Sprintf("%v|%v", u.Chat.ID, u.Chat.Title))
	}
}

// Reply replies a message back.
func (bm *BotMaid) Reply(u *Update, s string) (*Update, error) {
	bm.antiReplyLoop(u)

	return (*u.Bot.API).Push(&Update{
		Message: &Message{
			Content: s,
		},
		Chat: u.Chat,
	})
}

func (bm *BotMaid) ReplyType(u *Update, s, t string) (*Update, error) {
	bm.antiReplyLoop(u)

	if Contains([]string{"Image", "Audio", "Sticker"}, t) {
		return (*u.Bot.API).Push(&Update{
			Message: &Message{
				Type:    t,
				Content: s,
			},
			Chat: u.Chat,
		})
	}

	return nil, errors.New("Invalid type of message")
}
