package botmaid

import (
	"fmt"
	"sort"

	"github.com/catsworld/random"
)

// HelpMenu describes the menu item of the help.
type HelpMenu struct {
	Menu, Help string
	Names      []string
}

func (bm *BotMaid) pushHelp(hc string, u *Update, showUndef bool) {
	for _, v := range bm.HelpMenus {
		if hc == v.Menu || In(hc, v.Names) {
			s := ""

			for _, v := range bm.Commands {
				if v.Master && !u.Bot.IsMaster(u.User) {
					continue
				}
				if v.Menu == hc {
					s += v.Names[0] + v.Help + "\n"
				}
			}

			if len(s) > 0 && s[len(s)-1] == '\n' {
				s = s[:len(s)-1]
			}

			u.Bot.Reply(u, s)
			return
		}
	}

	s := ""

	for _, c := range bm.Commands {
		if c.Master && !u.Bot.IsMaster(u.User) {
			continue
		}
		for _, n := range c.Names {
			if n == hc {
				s += n + c.Help + "\n"
				break
			}
		}
	}

	if s != "" {

		if s[len(s)-1] == '\n' {
			s = s[:len(s)-1]
		}

		u.Bot.Reply(u, s)
		return
	}

	if !showUndef {
		return
	}

	u.Bot.Reply(u, fmt.Sprintf(random.String(bm.Words["undefCommand"]), hc))
}

func (bm *BotMaid) help(u *Update) bool {
	if u.Bot.IsCommand(u, []string{"help"}) && len(u.Message.Args) == 1 {
		s := fmt.Sprintf(random.String(bm.Words["selfIntro"]), u.User.NickName) + "\n\n"

		menus := []string{}

		for _, v := range bm.HelpMenus {
			menus = append(menus, v.Menu)
		}

		sort.Strings(menus)

		for _, k := range menus {
			f := false

			for _, c := range bm.Commands {
				if c.Master && !u.Bot.IsMaster(u.User) {
					continue
				}
				if c.Menu == k {
					f = true
					break
				}
			}

			if f {
				for _, v := range bm.HelpMenus {
					if k == v.Menu || In(k, v.Names) {
						s += k + " - " + v.Help + "\n"
						break
					}
				}
			}
		}

		if len(s) > 0 && s[len(s)-1] == '\n' {
			s = s[:len(s)-1]
		}

		u.Bot.Reply(u, s)
		return true
	}

	hc := ""
	if u.Bot.IsCommand(u, []string{"help"}) && len(u.Message.Args) == 2 {
		hc = u.Message.Args[1]
	} else if u.Bot.IsCommand(u, []string{}) && len(u.Message.Args) == 2 && In(u.Message.Args[1], "help") {
		hc = u.Bot.extractCommand(u)
	} else {
		return false
	}

	bm.pushHelp(hc, u, true)
	return true
}

func (bm *BotMaid) help2(u *Update) bool {
	if u.Bot.IsCommand(u, []string{}) {
		hc := u.Bot.extractCommand(u)

		bm.pushHelp(hc, u, false)
		return true
	}

	return false
}
