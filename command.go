package botmaid

import (
	"strings"

	"github.com/spf13/pflag"
)

// Command is a func with priority value so that we can sort some Commands to make them in a specific order.
type Command struct {
	Do func(*Update, *pflag.FlagSet) bool

	Priority int

	Help *Help

	Master bool
}

// CommandSlice is a slice of Command that could be sort.
type CommandSlice []*Command

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
func (bm *BotMaid) AddCommand(c *Command) {
	if c.Do == nil {
		c.Do = func(_ *Update, _ *pflag.FlagSet) bool {
			return false
		}
	}

	bm.Commands = append(bm.Commands, c)
}

func (bm *BotMaid) extractCommand(u *Update) string {
	if len(u.Message.Args) < 1 {
		return ""
	}

	s := u.Message.Args[0]
	for _, v := range bm.ats(u.Bot.Self) {
		if len(u.Message.Args[0])-len(v) > 0 && strings.LastIndex(u.Message.Args[0], v) == len(u.Message.Args[0])-len(v) {
			s = u.Message.Args[0][:len(u.Message.Args[0])-len(v)]
			break
		}
	}

	f := false
	for _, v := range bm.Conf.CommandPrefix {
		if strings.HasPrefix(s, v) {
			s = strings.Replace(s, v, "", 1)
			f = true
			break
		}
	}

	if !f {
		return ""
	}

	return s
}

// IsCommand checks if a message is a specific command.
func IsCommand(u *Update, c ...interface{}) bool {
	if len(c) == 0 && u.Message.Command != "" {
		return true
	}

	if len(c) == 1 {
		if _, ok := c[0].([]string); ok {
			for _, v := range c[0].([]string) {
				if u.Message.Command == v {
					return true
				}
			}
			return false
		}
	}

	for _, v := range c {
		if u.Message.Command == v {
			return true
		}
	}
	return false
}
