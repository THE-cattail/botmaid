package botmaid

import (
	"fmt"
	"sort"

	"github.com/catsworld/api"
	"github.com/catsworld/random"
	"github.com/catsworld/slices"
)

func (bm *BotMaid) pushHelp(hc string, e *api.Event, b *Bot, showUndef bool) {
	if _, ok := bm.HelpMenus[hc]; ok {
		s := ""

		for _, v := range bm.Commands {
			if v.Master && !b.IsMaster(*e.Sender) {
				continue
			}
			if v.Test && !b.IsTestPlace(*e.Place) {
				continue
			}
			if v.Menu == hc {
				s += v.Names[0] + v.Help + "\n"
			}
		}

		if len(s) > 0 && s[len(s)-1] == '\n' {
			s = s[:len(s)-1]
		}

		b.API.Push(api.Event{
			Message: &api.Message{
				Text: s,
			},
			Place: e.Place,
		})
		return
	}

	s := ""

	for _, c := range bm.Commands {
		if c.Master && !b.IsMaster(*e.Sender) {
			continue
		}
		if c.Test && !b.IsTestPlace(*e.Place) {
			continue
		}
		for _, n := range c.Names {
			if n == hc {
				s += n + c.Help
				break
			}
		}
	}
	if s != "" {

		if s[len(s)-1] == '\n' {
			s = s[:len(s)-1]
		}

		b.API.Push(api.Event{
			Message: &api.Message{
				Text: s,
			},
			Place: e.Place,
		})
		return
	}

	if !showUndef {
		return
	}

	b.API.Push(api.Event{
		Message: &api.Message{
			Text: fmt.Sprintf(random.String(bm.Words["undefCommand"]), hc),
		},
		Place: e.Place,
	})
}

func (bm *BotMaid) help(e *api.Event, b *Bot) bool {
	args := SplitCommand(e.Message.Text)
	if b.IsCommand(e, "help") && len(args) == 1 {
		s := fmt.Sprintf(random.String(bm.Words["selfIntro"]), e.Sender.NickName) + "\n\n"

		menus := []string{}

		for k := range bm.HelpMenus {
			menus = append(menus, k)
		}

		sort.Strings(menus)

		for _, k := range menus {
			f := false

			for _, c := range bm.Commands {
				if c.Master && !b.IsMaster(*e.Sender) {
					continue
				}
				if c.Test && !b.IsTestPlace(*e.Place) {
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

		b.API.Push(api.Event{
			Message: &api.Message{
				Text: s,
			},
			Place: e.Place,
		})
		return true
	}

	hc := ""
	if b.IsCommand(e, "help") && len(args) == 2 {
		hc = args[1]
	} else if b.IsCommand(e) && len(args) == 2 && slices.In(args[1], "help") {
		hc = b.extractCommand(e)
	} else {
		return false
	}

	bm.pushHelp(hc, e, b, true)
	return true
}

func (bm *BotMaid) help2(e *api.Event, b *Bot) bool {
	if b.IsCommand(e) {
		hc := b.extractCommand(e)

		bm.pushHelp(hc, e, b, false)
		return true
	}

	return false
}
