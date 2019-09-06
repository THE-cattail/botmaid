package botmaid

import (
	"strconv"
	"strings"
)

// Broadcast sends an update to all chats in the table.
func (bm *BotMaid) Broadcast(key string, m *Message) {
	cs := bm.Redis.SMembers("broadcast_" + key).Val()

	for _, v := range cs {
		args := strings.Split(v, "|")
		botID := args[0]
		chatType := args[1]
		chatID, _ := strconv.ParseInt(args[2], 10, 0)

		bm.Bots[botID].API.Push(Update{
			Message: m,
			Chat: &Chat{
				Type: chatType,
				ID:   chatID,
			},
		})
	}
}

// SwitchBroadcast switches the broadcast on/off of a chat.
func (bm *BotMaid) SwitchBroadcast(key string, c *Chat, b *Bot) {
	f := bm.Redis.SIsMember("broadcast_"+key, b.ID+"|"+c.Type+"|"+strconv.FormatInt(c.ID, 10)).Val()
	if f {
		bm.Redis.SRem("broadcast_"+key, b.ID+"|"+c.Type+"|"+strconv.FormatInt(c.ID, 10))
	} else {
		bm.Redis.SAdd("broadcast_"+key, b.ID+"|"+c.Type+"|"+strconv.FormatInt(c.ID, 10))
	}
}
