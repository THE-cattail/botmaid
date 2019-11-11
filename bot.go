package botmaid

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Bot includes some information of a bot.
type Bot struct {
	ID string

	API *API

	Self *User

	BotMaid *BotMaid
}

// IsMaster checks if a user is master of the bot.
func (bm *BotMaid) IsMaster(u *User) bool {
	return bm.Redis.SIsMember("master_"+u.Update.Bot.ID, u.ID).Val()
}

// IsBanned checks if a user has been banned.
func (bm *BotMaid) IsBanned(c *Chat) bool {
	return bm.Redis.SIsMember("ban_"+c.Update.Bot.ID, c.ID).Val()
}

// At returns a string to mention someone in a message.
func (bm *BotMaid) At(u *User) string {
	return (*u.Update.Bot.API).ats(u)[0]
}

// BeAt checks if a message of an update is mentioning the bot.
func (bm *BotMaid) BeAt(u *Update) bool {
	if bm.extractCommand(u) != "" {
		return false
	}

	for _, v := range (*u.Bot.API).ats(u.Bot.Self) {
		if strings.Contains(u.Message.Content, v) {
			return true
		}
	}

	return false
}

func (bm *BotMaid) antiReplyLoop(u *Update) {
	now := time.Now()
	for len(bm.history[u.Chat.ID]) > 0 && now.Sub(bm.history[u.Chat.ID][0]) > time.Second {
		bm.history[u.Chat.ID] = bm.history[u.Chat.ID][1:]
	}
	bm.history[u.Chat.ID] = append(bm.history[u.Chat.ID], now)
	if len(bm.history[u.Chat.ID]) >= 5 {
		bm.Redis.SAdd(fmt.Sprintf("ban_%v", u.Bot.ID), u.Chat.ID)
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

// Reply replies a message back with a type.
func (bm *BotMaid) ReplyType(u *Update, s, t string) (*Update, error) {
	bm.antiReplyLoop(u)

	if Contains([]string{"", "Text", "Image", "Audio", "Sticker"}, t) {
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

func (bm *BotMaid) Delete(u *Update) (*Update, error) {
	uu := *u
	uu.Type = "Delete"
	return (*u.Bot.API).Push(&uu)
}
