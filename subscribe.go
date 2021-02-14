package botmaid

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/pflag"
)

// Broadcast sends an update to all chats in the table.
func (bm *BotMaid) Broadcast(key string, m *Message) {
	cs := bm.Redis.SMembers("subscribe_" + key).Val()

	for _, v := range cs {
		args := strings.Split(v, "|")
		botID := args[0]
		chatType := args[1]
		chatID, _ := strconv.ParseInt(args[2], 10, 0)

		(*bm.Bots[botID].API).Push(&Update{
			Message: m,
			Chat: &Chat{
				Type: chatType,
				ID:   chatID,
			},
		})
	}
}

func (bm *BotMaid) SubscribeCommandDo(u *Update, f *pflag.FlagSet) bool {
	if !bm.IsMaster(u.User) {
		bm.Reply(u, fmt.Sprintf(bm.Words["noPermission"], bm.At(u.User), "subscribe"))
		return true
	}

	if len(f.Args()) == 1 || !Contains(bm.SubEntries, f.Args()[1]) {
		bm.Reply(u, fmt.Sprintf(bm.Words["correctSubEntries"], ListToString(bm.SubEntries, bm.Words["subEntriesFormat"], bm.Words["subEntriesSeparator"], bm.Words["subEntriesAnd"])))
		return true
	}

	if len(f.Args()) == 2 {
		if bm.Redis.SIsMember("subscribe_"+f.Args()[1], u.Bot.ID+"|"+u.Chat.Type+"|"+strconv.FormatInt(u.Chat.ID, 10)).Val() {
			bm.Redis.SRem("subscribe_"+f.Args()[1], u.Bot.ID+"|"+u.Chat.Type+"|"+strconv.FormatInt(u.Chat.ID, 10))
			bm.Reply(u, fmt.Sprintf(bm.Words["unsubscribed"], f.Args()[1]))
			return true
		}

		bm.Redis.SAdd("subscribe_"+f.Args()[1], u.Bot.ID+"|"+u.Chat.Type+"|"+strconv.FormatInt(u.Chat.ID, 10))
		bm.Reply(u, fmt.Sprintf(bm.Words["subscribed"], f.Args()[1]))
		return true
	}

	return false
}
