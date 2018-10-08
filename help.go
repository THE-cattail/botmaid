package botmaid

import (
	"github.com/catsworld/api"
	"github.com/catsworld/slices"
)

// Help stores a full help menus that would be used in '/help' command.
type Help struct {
	SelfIntro, HelpMenu, UndefCommand string
	HelpSubMenu                       map[string]string
}

var (
	h *Help
)

// RegHelpCommand adds help as a high-priority command if a Help is defined and
// going to be used.
func RegHelpCommand(cs *[]Command, hs *Help) {
	AddCommand(cs, help, 100)
	h = hs
}

func help(e *api.Event, b *Bot) bool {
	args := SplitCommand(e.Message.Text)
	if b.IsCommand(e, "help", "?") && len(args) == 1 {
		b.API.Push(&api.Event{
			Message: &api.Message{
				Text: h.SelfIntro + "\n\n" + h.HelpMenu,
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
				Text: h.UndefCommand,
			},
			Place: e.Place,
		})
	}
	return true
}
