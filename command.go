package botmaid

import (
	"regexp"
	"strings"

	"github.com/catsworld/api"
)

// Command is a func with priority value so that we can sort some Commands to
// make them in a specific order.
type Command struct {
	Do       func(*api.Event, *Bot) bool
	Priority int
}

// CommandSlice is a slice of Command that could be sort.
type CommandSlice []Command

// Len is the length of a CommandSlice.
func (cs CommandSlice) Len() int {
	return len(cs)
}

// Swap swaps CommandSlice[i] and CommandSlice[j].
func (cs CommandSlice) Swap(i, j int) {
	cs[i], cs[j] = cs[j], cs[i]
}

// Less returns true if CommandSlice[i] is less then CommandSlice[j].
func (cs CommandSlice) Less(i, j int) bool {
	return cs[i].Priority > cs[j].Priority
}

// AddCommand adds a command into the CommandSlice.
func AddCommand(cs *[]Command, c func(*api.Event, *Bot) bool, p int) {
	*cs = append(*cs, Command{
		Do:       c,
		Priority: p,
	})
}

// SplitCommand splits a string into a slice of string.
func SplitCommand(c string) []string {
	var a []string
	p := `("[^"]+")|([^"\s]+)`
	r := regexp.MustCompile(p).FindAllString(c, -1)
	for _, v := range r {
		if len(v) > 1 {
			if (v[0] == '"' && v[len(v)-1] == '"') ||
				(v[0] == '\'' && v[len(v)-1] == '\'') ||
				(v[0] == '`' && v[len(v)-1] == '`') {
				v = v[1 : len(v)-1]
			}
		}
		if v != "" {
			a = append(a, v)
		}
	}
	return a
}

func (b *Bot) extractCommand(u *api.Event) string {
	args := SplitCommand(u.Message.Text)
	if len(args) == 0 {
		return ""
	}
	s := args[0]
	if len(args[0])-len(b.At(b.Self)) > 0 && strings.LastIndex(args[0], b.At(b.Self)) == len(args[0])-len(b.At(b.Self)) {
		s = args[0][:len(args[0])-len(b.At(b.Self))]
	}
	if strings.Index(s, "/") == 0 {
		s = strings.Replace(s, "/", "", 1)
	} else if strings.Index(s, ":") == 0 {
		s = strings.Replace(s, ":", "", 1)
	} else if strings.Index(s, "：") == 0 {
		s = strings.Replace(s, "：", "", 1)
	} else {
		return ""
	}
	return s
}

// IsCommand checks if a message is a specific command.
func (b *Bot) IsCommand(u *api.Event, c ...string) bool {
	s := b.extractCommand(u)
	for _, v := range c {
		if s == v {
			return true
		}
	}
	return false
}

// CheckBeAt checks if someone send a message without commands and at the api.
func (b *Bot) CheckBeAt(u *api.Event) bool {
	if strings.Contains(u.Message.Text, b.At(b.Self)) && b.extractCommand(u) == "" {
		return true
	}
	return false
}
