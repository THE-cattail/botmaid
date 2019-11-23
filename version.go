package botmaid

import (
	"fmt"

	"github.com/spf13/pflag"
)

func (bm *BotMaid) getLog() string {
	log := ""
	l := bm.Redis.LRange("log_"+bm.Redis.Get("version").Val(), 0, -1).Val()
	for i := range l {
		log += fmt.Sprintf("\n%v. %v", i+1, l[i])
	}

	return fmt.Sprintf(bm.Words["fmtLog"], bm.Redis.Get("version").Val(), log)
}

// VersionCommandDo is a Do func for a version command.
func (bm *BotMaid) VersionCommandDo(u *Update, f *pflag.FlagSet) bool {
	log, _ := f.GetBool("log")
	if log {
		bm.Reply(u, bm.getLog())
		return true
	}

	bm.Reply(u, fmt.Sprintf(bm.Words["fmtVersion"], bm.Redis.Get("version").Val()))
	return true
}

// VersionCommandHelpSetFlag is a SetFlag func for a version command.
func (bm *BotMaid) VersionCommandHelpSetFlag(f *pflag.FlagSet) {
	f.BoolP("log", "l", false, bm.Words["versionLogHelp"])
}

// VersetCommandDo is a Do func for a verset command.
func (bm *BotMaid) VersetCommandDo(u *Update, f *pflag.FlagSet) bool {
	if !bm.IsMaster(u.User) {
		bm.Reply(u, fmt.Sprintf(bm.Words["noPermission"], bm.At(u.User), "verset"))
		return true
	}

	broadcast, _ := f.GetBool("broadcast")
	if broadcast {
		bm.Broadcast("log", &Message{
			Content: bm.Words["upgraded"] + bm.getLog(),
		})
		return true
	}

	flag := false
	v := bm.Redis.Get("version").Val()

	ver, _ := f.GetString("ver")
	if ver != "" {
		v = ver
	}

	if len(f.Args()) == 2 {
		bm.Redis.Set("version", f.Args()[1], 0)
		bm.Reply(u, fmt.Sprintf(bm.Words["versionSet"], f.Args()[1]))
		flag = true
	}

	log, _ := f.GetString("log")
	if log != "" {
		bm.Redis.RPush("log_"+v, log)
		bm.Reply(u, fmt.Sprintf(bm.Words["logAdded"], log))
		flag = true
	}

	return flag
}

// VersetCommandHelpSetFlag is a SetFlag func for a verset command.
func (bm *BotMaid) VersetCommandHelpSetFlag(f *pflag.FlagSet) {
	f.String("ver", "", bm.Words["versetVerHelp"])
	f.String("log", "", bm.Words["versetLogHelp"])
	f.Bool("broadcast", false, bm.Words["versetBroadcastHelp"])
}
