package botmaid

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/pflag"
)

// Help describes the menu item of the help.
type Help struct {
	Menu, Help, Usage, Comment string

	Names []string

	SetFlag func(*pflag.FlagSet)
}

func (bm *BotMaid) pushHelp(u *Update, hc string, showUndef bool) {
	for _, c := range bm.Commands {
		if c.Help == nil {
			continue
		}
		if c.Help.Menu == "" {
			continue
		}
		if !Contains(c.Help.Names, hc) {
			continue
		}

		lines := strings.Split(u.Message.Flags[c.Help.Menu].FlagUsages(), "\n")
		s := ""

		for i := range lines {
			lines[i] = strings.TrimSpace(lines[i])

			for strings.Contains(lines[i], "  ") {
				lines[i] = strings.ReplaceAll(lines[i], "  ", "\n")
			}
			for strings.Contains(lines[i], "\n ") {
				lines[i] = strings.ReplaceAll(lines[i], "\n ", "\n")
			}
			lines[i] = strings.Replace(lines[i], "\n", "  ", 1)
			lines[i] = strings.ReplaceAll(lines[i], "\n", "")

			s += "\n  " + lines[i]
		}
		s = strings.TrimSpace(c.Help.Usage + "\n" + s + c.Help.Comment)

		if s == "" {
			bm.Reply(u, fmt.Sprintf(bm.Words["noHelpText"], bm.At(u.User), hc))
			return
		}

		bm.Reply(u, s)
		return
	}

	if showUndef {
		bm.Reply(u, fmt.Sprintf(bm.Words["undefCommand"], bm.At(u.User), hc))
	}
}

func (bm *BotMaid) HelpCommandDo(u *Update, f *pflag.FlagSet) bool {
	if len(f.Args()) == 1 {
		helps := []string{}

		for _, c := range bm.Commands {
			if c.Help == nil || c.Help.Menu == "" {
				continue
			}

			helps = append(helps, fmt.Sprintf("  %v  %v", c.Help.Menu, c.Help.Help))
		}

		sort.Strings(helps)

		s := ""
		for _, v := range helps {
			s += "\n" + v
		}

		bm.Reply(u, fmt.Sprintf(bm.Words["selfIntro"], u.Bot.Self.NickName, s))
		return true
	}

	if len(f.Args()) == 2 {
		bm.pushHelp(u, f.Args()[1], true)
		return true
	}

	return false
}

func (bm *BotMaid) HelpRespCommandDo(u *Update, f *pflag.FlagSet) bool {
	if u.Message.Command != "" {
		for _, c := range bm.Commands {
			if c.Help != nil && len(c.Help.Names) != 0 && !Contains(c.Help.Names, u.Message.Command) {
				continue
			}
		}

		bm.pushHelp(u, u.Message.Command, false)
		return true
	}

	return false
}
