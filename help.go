package botmaid

import (
	"fmt"
	"sort"

	"github.com/catsworld/random"
	"github.com/catsworld/slices"
)

func (bm *BotMaid) pushHelp(hc string, u *Update, b *Bot, showUndef bool) {
	if _, ok := bm.HelpMenus[hc]; ok {
		s := ""

		for _, v := range bm.Commands {
			if v.Master && !b.IsMaster(*u.User) {
				continue
			}
			if v.Test && !b.IsTestChat(*u.Chat) {
				continue
			}
			if v.Menu == hc {
				s += v.Names[0] + v.Help + "\n"
			}
		}

		if len(s) > 0 && s[len(s)-1] == '\n' {
			s = s[:len(s)-1]
		}

		b.Reply(u, s)
		return
	}

	s := ""

	for _, c := range bm.Commands {
		if c.Master && !b.IsMaster(*u.User) {
			continue
		}
		if c.Test && !b.IsTestChat(*u.Chat) {
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

		b.Reply(u, s)
		return
	}

	if !showUndef {
		return
	}

	b.Reply(u, fmt.Sprintf(random.String(bm.Words["undefCommand"]), hc))
}

func (bm *BotMaid) help(u *Update, b *Bot) bool {
	args := SplitCommand(u.Message.Text)
	if b.IsCommand(u, "help") && len(args) == 1 {
		s := fmt.Sprintf(random.String(bm.Words["selfIntro"]), u.User.NickName) + "\n\n"

		menus := []string{}

		for k := range bm.HelpMenus {
			menus = append(menus, k)
		}

		sort.Strings(menus)

		for _, k := range menus {
			f := false

			for _, c := range bm.Commands {
				if c.Master && !b.IsMaster(*u.User) {
					continue
				}
				if c.Test && !b.IsTestChat(*u.Chat) {
					continue
				}
				if c.Menu == k {
					f = true
					break
				}
			}

			if f {
				s += k + " - " + bm.HelpMenus[k] + "\n"
			}
		}

		if len(s) > 0 && s[len(s)-1] == '\n' {
			s = s[:len(s)-1]
		}

		b.Reply(u, s)
		return true
	}

	hc := ""
	if b.IsCommand(u, "help") && len(args) == 2 {
		hc = args[1]
	} else if b.IsCommand(u) && len(args) == 2 && slices.In(args[1], "help") {
		hc = b.extractCommand(u)
	} else {
		return false
	}

	bm.pushHelp(hc, u, b, true)
	return true
}

func (bm *BotMaid) help2(u *Update, b *Bot) bool {
	if b.IsCommand(u) {
		hc := b.extractCommand(u)

		bm.pushHelp(hc, u, b, false)
		return true
	}

	return false
}
