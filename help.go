package botmaid

import (
	"fmt"

	"github.com/catsworld/api"
	"github.com/catsworld/random"
	"github.com/catsworld/slices"
)

// Help stores a full help menus that would be used in '/help' command.
type Help struct {
	HelpMenu                string
	SelfIntro, UndefCommand []string
	HelpSubMenu, HelpAlias  map[string]string
}

var (
	h *Help
)

// RegHelpCommand adds help as a high-priority command if a Help is defined and
// going to be used.
func RegHelpCommand(cs *[]Command, hs *Help) {
	AddCommand(cs, help, 10000)
	AddCommand(cs, help2, -10000)
	h = hs
}

func help(e *api.Event, b *Bot) bool {
	args := SplitCommand(e.Message.Text)
	if b.IsCommand(e, "help", "?") && len(args) == 1 {
		b.API.Push(&api.Event{
			Message: &api.Message{
				Text: fmt.Sprintf(random.String(h.SelfIntro), e.Sender.NickName) + "\n\n" + h.HelpMenu,
			},
			Place: e.Place,
		})
		return true
	}

	helpCommand := ""
	if b.IsCommand(e, "help", "?") && len(args) > 1 {
		helpCommand = args[1]
	} else if b.IsCommand(e) && len(args) > 1 && slices.In(args[1], "--help", "-?") {
		helpCommand = b.extractCommand(e)
	} else {
		return false
	}

	if _, ok := h.HelpAlias[helpCommand]; ok {
		helpCommand = h.HelpAlias[helpCommand]
	}

	if s, ok := h.HelpSubMenu[helpCommand]; ok {
		b.API.Push(&api.Event{
			Message: &api.Message{
				Text: s,
			},
			Place: e.Place,
		})
	} else {
		b.API.Push(&api.Event{
			Message: &api.Message{
				Text: fmt.Sprintf(random.String(h.UndefCommand), helpCommand),
			},
			Place: e.Place,
		})
	}
	return true
}

func help2(e *api.Event, b *Bot) bool {
	for k, v := range h.HelpAlias {
		if b.IsCommand(e, k) {
			b.API.Push(&api.Event{
				Message: &api.Message{
					Text: h.HelpSubMenu[v],
				},
				Place: e.Place,
			})
			return true
		}
	}

	for k, v := range h.HelpSubMenu {
		if b.IsCommand(e, k) {
			b.API.Push(&api.Event{
				Message: &api.Message{
					Text: v,
				},
				Place: e.Place,
			})
			return true
		}
	}

	return false
}
