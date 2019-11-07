package botmaid

import (
	"fmt"
	"strings"

	"github.com/catsworld/botmaid/random"
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
		if c.Master && !bm.IsMaster(u.User) {
			continue
		}
		if !In(hc, c.Help.Names) {
			continue
		}

		s := strings.TrimSpace(fmt.Sprintf(c.Help.Full, bm.Flags[c.Help.Menu].FlagUsages()))

		if s == "" {
			Reply(u, fmt.Sprintf(random.String(bm.Words["noHelpText"]), At(u.User), hc))
			return
		}

		Reply(u, s)
		return
	}

	if showUndef {
		Reply(u, fmt.Sprintf(random.String(bm.Words["undefCommand"]), At(u.User), hc))
	}
}
