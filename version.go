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

	logBM := ""
	l = bm.Redis.LRange("logBM_"+bm.Redis.Get("version").Val(), 0, -1).Val()
	for i := range l {
		logBM += fmt.Sprintf("\n%v. %v", i+1, l[i])
	}

	return fmt.Sprintf(random.String(bm.Words["fmtLog"]), bm.Redis.Get("version").Val(), log, logBM)
}

func (bm *BotMaid) VersionCommandDo(u *Update, f *pflag.FlagSet) bool {
	if len(f.Args()) == 1 {
		Reply(u, fmt.Sprintf(random.String(bm.Words["fmtVersion"]), bm.Redis.Get("version").Val()))
		return true
	}

	if len(f.Args()) == 2 && In(f.Args()[1], "log") {
		Reply(u, bm.getLog())
		return true
	}

	return false
}

func (bm *BotMaid) VersionMasterCommandDo(u *Update, f *pflag.FlagSet) bool {
	if len(f.Args()) == 2 {
		log, _ := f.GetString("log")
		if log != "" {
			bm.Redis.RPush("log_"+bm.Redis.Get("version").Val(), f.Args())
			Reply(u, random.String(bm.Words["logAdded"]))
			return true
		}

		logBM, _ := f.GetString("logbm")
		if logBM != "" {
			bm.Redis.RPush("logBM_"+bm.Redis.Get("version").Val(), f.Args())
			Reply(u, random.String(bm.Words["logBMAdded"]))
			return true
		}

		bm.Redis.Set("version", f.Args()[1], 0)
		return true
	}

	broadcast, _ := f.GetBool("broadcast")
	if broadcast {
		bm.Broadcast("log", &Message{
			Text: bm.getLog(),
		})
		return true
	}

	return false
}

func (bm *BotMaid) VersionMasterCommandHelpSetFlag(f *pflag.FlagSet) {
	f.String("log", "", random.String(bm.Words["versionMasterLogHelp"]))
	f.String("logbm", "", random.String(bm.Words["versionMasterLogHelp"]))
	f.Bool("broadcast", false, random.String(bm.Words["versionMasterBroadcastHelp"]))
}
