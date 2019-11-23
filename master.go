package botmaid

import (
	"fmt"

	"github.com/spf13/pflag"
)

func (bm *BotMaid) MasterCommandDo(u *Update, f *pflag.FlagSet) bool {
	if !bm.IsMaster(u.User) {
		bm.Reply(u, fmt.Sprintf(bm.Words["noPermission"], bm.At(u.User), "master"))
		return true
	}

	if len(f.Args()) != 2 {
		return false
	}

	id, err := (*u.Bot.API).ParseUserID(u, f.Args()[1])
	if err != nil {
		bm.Reply(u, fmt.Sprintf(bm.Words["invalidUser"], bm.At(u.User), f.Args()[1]))
		return true
	}

	is := bm.Redis.SIsMember("master_"+u.Bot.ID, id).Val()

	if is {
		bm.Redis.SRem("master_"+u.Bot.ID, id)
		bm.Reply(u, fmt.Sprintf(bm.Words["unregMaster"], bm.At(u.User), f.Args()[1]))
		return true
	}

	bm.Redis.SAdd("master_"+u.Bot.ID, id)
	bm.Reply(u, fmt.Sprintf(bm.Words["regMaster"], bm.At(u.User), f.Args()[1]))
	return true
}
