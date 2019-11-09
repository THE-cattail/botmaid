package botmaid

import (
	"fmt"
	"strings"

	"github.com/spf13/pflag"
)

// Help describes the menu item of the help.
type Help struct {
	Menu, Help, Full string

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
		if !In(hc, c.Help.Names) {
			continue
		}

		if c.Master && !bm.IsMaster(u.User) {
			Reply(u, fmt.Sprintf(bm.Words["noPermission"], At(u.User), hc))
			return
		}

		lines := strings.Split(bm.Flags[c.Help.Menu].FlagUsages(), "\n")
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

			if i != 0 {
				s += "\n"
			}

			s += "  " + lines[i]
		}
		s = strings.TrimSpace(fmt.Sprintf(c.Help.Full, s))

		if s == "" {
			Reply(u, fmt.Sprintf(bm.Words["noHelpText"], At(u.User), hc))
			return
		}

		Reply(u, s)
		return
	}

	if showUndef {
		Reply(u, fmt.Sprintf(bm.Words["undefCommand"], At(u.User), hc))
	}
}
