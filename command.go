package botmaid

import (
	"regexp"
	"strings"
)

// Command is a func with priority value so that we can sort some Commands to
// make them in a specific order.
type Command struct {
	Do       func(*Update, *Bot) bool
	Priority int

	Menu, Help   string
	Names        []string
	Master, Test bool
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

// AddCommand adds a command into the []Command.
func (bm *BotMaid) AddCommand(c Command) {
	bm.Commands = append(bm.Commands, c)
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

// GetArgument returns a slice with a command and an argument.
func GetArgument(c string) []string {
	args := SplitCommand(c)

	t := strings.Index(c, args[0]) + len(args[0]) + 1

	ret := ""
	if t < len(c) {
		ret = c[t:]
	}
	return []string{args[0], ret}
}

func (b *Bot) extractCommand(u *Update) string {
	args := SplitCommand(u.Message.Text)
	if len(args) == 0 {
		return ""
	}
	s := args[0]
	for _, v := range b.At(b.Self) {
		if len(args[0])-len(v) > 0 && strings.LastIndex(args[0], v) == len(args[0])-len(v) {
			s = args[0][:len(args[0])-len(v)]
			break
		}
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
func (b *Bot) IsCommand(u *Update, c ...string) bool {
	s := b.extractCommand(u)

	if len(c) == 0 && s != "" {
		return true
	}

	for _, v := range c {
		if s == v {
			return true
		}
	}
	return false
}
