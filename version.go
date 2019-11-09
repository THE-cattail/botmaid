package botmaid

import (
	"fmt"

	"github.com/catsworld/botmaid/random"
	"github.com/spf13/pflag"
)

func (bm *BotMaid) getLog() string {
	log := ""
	l := bm.Redis.LRange("log_"+bm.Redis.Get("version").Val(), 0, -1).Val()
	for i := range l {
		log += fmt.Sprintf("\n%v. %v", i+1, l[i])
	}

	return fmt.Sprintf(random.String(bm.Words["fmtLog"]), bm.Redis.Get("version").Val(), log)
}

func (bm *BotMaid) VersionCommandDo(u *Update, f *pflag.FlagSet) bool {
	log, _ := f.GetBool("log")
	if log {
		Reply(u, bm.getLog())
		return true
	}

	Reply(u, fmt.Sprintf(random.String(bm.Words["fmtVersion"]), bm.Redis.Get("version").Val()))
	return true
}

func (bm *BotMaid) VersionCommandHelpSetFlag(f *pflag.FlagSet) {
	f.BoolP("log", "l", false, random.String(bm.Words["versionLogHelp"]))
}

func (bm *BotMaid) VersionMasterCommandDo(u *Update, f *pflag.FlagSet) bool {
	broadcast, _ := f.GetBool("broadcast")
	if broadcast {
		bm.Broadcast("log", &Message{
			Text: bm.getLog(),
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
		Reply(u, random.String(bm.Words["versionSet"]))
		flag = true
	}

	log, _ := f.GetString("log")
	if log != "" {
		bm.Redis.RPush("log_"+v, log)
		Reply(u, random.String(bm.Words["logAdded"]))
		flag = true
	}

	return flag
}

func (bm *BotMaid) VersionMasterCommandHelpSetFlag(f *pflag.FlagSet) {
	f.String("ver", "", random.String(bm.Words["versionMasterVerHelp"]))
	f.String("log", "", random.String(bm.Words["versionMasterLogHelp"]))
	f.Bool("broadcast", false, random.String(bm.Words["versionMasterBroadcastHelp"]))
}
