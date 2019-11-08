// Package botmaid is a package for managing bots.
package botmaid

import (
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis"
	"github.com/google/shlex"
	"github.com/pelletier/go-toml"
	"github.com/spf13/pflag"

	"github.com/catsworld/botmaid/random"
)

type botmaidRedisConfig struct {
	Address  string
	Password string
	Database int
}

type botMaidConfig struct {
	Redis         botmaidRedisConfig
	Log           bool
	CommandPrefix []string
}

// BotMaid includes a slice of Bot and some methods to use them.
type BotMaid struct {
	Bots map[string]*Bot

	Conf *botMaidConfig

	Redis *redis.Client

	Commands CommandSlice
	Timers   []*Timer
	Helps    []*Help
	Flags    map[string]*pflag.FlagSet

	Words map[string][]string

	respTime time.Time
}

func (bm *BotMaid) readBotConfig(conf *toml.Tree, section string) error {
	botType := conf.Get(section + ".Type").(string)

	b := &Bot{
		ID:  section,
		API: new(API),
	}

	if botType == "QQ" {
		q := &APICqhttp{}

		if s, ok := conf.Get(section + ".AccessToken").(string); ok {
			q.AccessToken = s
		}
		if s, ok := conf.Get(section + ".Secret").(string); ok {
			q.Secret = s
		}
		if s, ok := conf.Get(section + ".APIEndpoint").(string); ok {
			q.APIEndpoint = s
		}

		for {
			m, err := q.API("get_login_info", map[string]interface{}{})
			if err != nil {
				if bm.Conf.Log {
					log.Printf("Init botmaid: %v, retrying...\n", err)
				}
				time.Sleep(time.Second * 3)
				continue
			}

			u := m.(map[string]interface{})
			b.Self = &User{
				ID:       int64(u["user_id"].(float64)),
				UserName: strconv.FormatInt(int64(u["user_id"].(float64)), 10),
				NickName: u["nickname"].(string),
				Bot:      b,
			}

			break
		}

		*b.API = q
	} else if botType == "Telegram" {
		t := &APITelegramBot{}

		if s, ok := conf.Get(section + ".Token").(string); ok {
			t.Token = s
		}

		for {
			m, err := t.API("getMe", map[string]interface{}{})
			if err != nil {
				if bm.Conf.Log {
					log.Printf("Init botmaid: %v, retrying...\n", err)
				}
				time.Sleep(time.Second * 3)
				continue
			}

			u := m.(map[string]interface{})
			b.Self = &User{
				ID:       int64(u["id"].(float64)),
				NickName: u["first_name"].(string),
				Bot:      b,
			}
			if u["last_name"] != nil {
				b.Self.NickName += " " + u["last_name"].(string)
			}
			if u["username"] != nil {
				b.Self.UserName = u["username"].(string)
			}

			break
		}
		*b.API = t
	} else {
		return fmt.Errorf("Init botmaid: Unknown type of %v", section)
	}

	if ms, ok := conf.Get(section + ".Master").([]interface{}); ok {
		for _, v := range ms {
			if id, ok := v.(int64); ok {
				bm.Redis.SAdd("master_"+b.ID, id)
			}
		}
	}

	bm.Bots[section] = b
	return nil
}

// Splits the string `s` on whitespace into an initial substring up to
// `i` runes in length and the remainder. Will go `slop` over `i` if
// that encompasses the entire string (which allows the caller to
// avoid short orphan words on the final line).
func wrapN(i, slop int, s string) (string, string) {
	if i+slop > len(s) {
		return s, ""
	}

	w := strings.LastIndexAny(s[:i], " \t\n")
	if w <= 0 {
		return s, ""
	}
	nlPos := strings.LastIndex(s[:i], "\n")
	if nlPos > 0 && nlPos < w {
		return s[:nlPos], s[nlPos+1:]
	}
	return s[:w], s[w+1:]
}

// Wraps the string `s` to a maximum width `w` with leading indent
// `i`. The first line is not indented (this is assumed to be done by
// caller). Pass `w` == 0 to do no wrapping
func wrap(i, w int, s string) string {
	if w == 0 {
		return strings.Replace(s, "\n", "\n"+strings.Repeat(" ", i), -1)
	}

	// space between indent i and end of line width w into which
	// we should wrap the text.
	wrap := w - i

	var r, l string

	// Not enough space for sensible wrapping. Wrap as a block on
	// the next line instead.
	if wrap < 24 {
		i = 16
		wrap = w - i
		r += "\n" + strings.Repeat(" ", i)
	}
	// If still not enough space then don't even try to wrap.
	if wrap < 24 {
		return strings.Replace(s, "\n", r, -1)
	}

	// Try to avoid short orphan words on the final line, by
	// allowing wrapN to go a bit over if that would fit in the
	// remainder of the line.
	slop := 5
	wrap = wrap - slop

	// Handle first line, which is indented by the caller (or the
	// special case above)
	l, s = wrapN(wrap, slop, s)
	r = r + strings.Replace(l, "\n", "\n"+strings.Repeat(" ", i), -1)

	// Now wrap the rest
	for s != "" {
		var t string

		t, s = wrapN(wrap, slop, s)
		r = r + "\n" + strings.Repeat(" ", i) + strings.Replace(t, "\n", "\n"+strings.Repeat(" ", i), -1)
	}

	return r

}

func (bm *BotMaid) initCommand() {
	bm.AddCommand(&Command{
		Do: func(u *Update, f *pflag.FlagSet) bool {
			if len(f.Args()) == 1 {
				helps := []string{}

				maxlen := 0
				for _, v := range bm.Commands {
					if v.Help == nil || v.Help.Menu == "" {
						continue
					}

					line := "  " + v.Help.Menu + "\x00"

					if len(line) > maxlen {
						maxlen = len(line)
					}

					line += v.Help.Help

					helps = append(helps, line)
				}

				for i := range helps {
					sidx := strings.Index(helps[i], "\x00")
					spacing := strings.Repeat(" ", maxlen-sidx)
					helps[i] = fmt.Sprint(helps[i][:sidx], spacing, wrap(maxlen+2, 0, helps[i][sidx+1:]))
				}

				sort.Strings(helps)

				s := ""
				for _, v := range helps {
					s += "\n" + v
				}

				Reply(u, fmt.Sprintf(random.String(bm.Words["selfIntro"]), u.Bot.Self.NickName, s))
				return true
			}

			if len(f.Args()) == 2 {
				bm.pushHelp(u, f.Args()[1], true)
				return true
			}

			return false
		},
		Help: &Help{
			Menu:  "help",
			Help:  random.String(bm.Words["helpHelp"]),
			Names: []string{"help"},
			Full: `Usage: help [COMMAND]

%v`,
		},
		Priority: 10000,
	})

	bm.AddCommand(&Command{
		Do: func(u *Update, f *pflag.FlagSet) bool {
			if IsCommand(u) {
				for _, c := range bm.Commands {
					if c.Help != nil && len(c.Help.Names) != 0 && !IsCommand(u, c.Help.Names) {
						continue
					}

					if !bm.IsMaster(u.User) && c.Master {
						Reply(u, fmt.Sprintf(random.String(bm.Words["noPermission"]), At(u.User), u.Message.Command))
						return true
					}
				}

				bm.pushHelp(u, u.Message.Command, false)
				return true
			}

			return false
		},
		Priority: -10000,
	})

	bm.AddCommand(&Command{
		Do: func(u *Update, f *pflag.FlagSet) bool {
			if len(f.Args()) != 2 {
				return false
			}

			id, err := bm.ParseUserID(u, f.Args()[1])
			if err != nil {
				Reply(u, fmt.Sprintf(random.String(bm.Words["invalidUser"]), At(u.User), f.Args()[1]))
				return true
			}

			is := bm.Redis.SIsMember("master_"+u.Bot.ID, id).Val()

			if is {
				bm.Redis.SRem("master_"+u.Bot.ID, id)
				Reply(u, fmt.Sprintf(random.String(bm.Words["unregMaster"]), At(u.User), f.Args()[1]))
				return true
			}

			bm.Redis.SAdd("master_"+u.Bot.ID, id)
			Reply(u, fmt.Sprintf(random.String(bm.Words["regMaster"]), f.Args()[1]))
			return true
		},
		Help: &Help{
			Menu:  "master",
			Help:  random.String(bm.Words["masterHelp"]),
			Names: []string{"master"},
			Full: `Usage: master USER

%v`,
		},
		Master: true,
	})

	bm.AddCommand(&Command{
		Do: func(u *Update, f *pflag.FlagSet) bool {
			if len(f.Args()) != 2 {
				return false
			}

			id, err := bm.ParseUserID(u, f.Args()[1])
			if err != nil {
				Reply(u, fmt.Sprintf(random.String(bm.Words["invalidUser"]), At(u.User), f.Args()[1]))
				return true
			}

			is := bm.Redis.SIsMember("ban_"+u.Bot.ID, id).Val()

			if is {
				bm.Redis.SRem("ban_"+u.Bot.ID, id)
				Reply(u, fmt.Sprintf(random.String(bm.Words["unbanUser"]), At(u.User), f.Args()[1]))
				return true
			}

			bm.Redis.SAdd("ban_"+u.Bot.ID, id)
			Reply(u, fmt.Sprintf(random.String(bm.Words["banUser"]), At(u.User), f.Args()[1]))
			return true
		},
		Help: &Help{
			Menu:  "ban",
			Help:  random.String(bm.Words["banHelp"]),
			Names: []string{"ban"},
			Full: `Usage: ban USER

%v`,
		},
		Master: true,
	})

	sort.Stable(CommandSlice(bm.Commands))
}

func (bm *BotMaid) startBot() {
	for _, b := range bm.Bots {
		bot := b
		go func(b *Bot) {
			updates, errors := (*b.API).Pull(&PullConfig{
				Limit:            100,
				Timeout:          60,
				RetryWaitingTime: time.Second * 3,
			})

			if bm.Conf.Log {
				go func() {
					for err := range errors {
						log.Printf("Bot running: %v.\n", err)
					}
				}()
				log.Printf("[%v] %v (%v) has been loaded. Begin to get updates.\n", b.ID, b.Self.NickName, b.Platform())
			}

			for u := range updates {
				up := u
				go func(u *Update) {
					if u.Message == nil {
						return
					}

					if !u.Time.After(bm.respTime) {
						return
					}

					if b.Platform() == "Telegram" && u.User != nil && u.User.UserName != "" {
						bm.Redis.HSet("telegramUsers", fmt.Sprintf("%v", u.User.UserName), u.User.ID)
					}

					u.Bot = b
					if u.User != nil {
						u.User.Bot = b
					}

					if bm.IsBanned(u.User) {
						return
					}

					if bm.Conf.Log {
						logText := u.Message.Text
						if u.User != nil {
							logText = u.User.NickName + ": " + logText
						}
						if u.Chat != nil && u.Chat.Title != "" {
							logText = "[" + u.Chat.Title + "]" + logText
						}
						log.Println(logText)
					}

					args, err := shlex.Split(u.Message.Text)
					if err != nil {
						Reply(u, fmt.Sprintf(random.String(bm.Words["invalidParameters"])), At(u.User), u.Message.Text)
						return
					}
					u.Message.Args = args

					u.Message.Command = bm.extractCommand(u)

					for _, c := range bm.Commands {
						if c.Help != nil && c.Help.Menu != "" {
							bm.Flags[c.Help.Menu].Parse(u.Message.Args)
						}
					}

					for _, c := range bm.Commands {
						if c.Help != nil && len(c.Help.Names) != 0 && !IsCommand(u, c.Help.Names) {
							continue
						}

						if !bm.IsMaster(u.User) && c.Master {
							continue
						}

						if ((c.Help == nil || c.Help.Menu == "") && c.Do(u, nil)) || c.Do(u, bm.Flags[c.Help.Menu]) {
							break
						}
					}
				}(up)
			}
		}(bot)
	}
}

// New creates a BotMaid.
func New(configFile string) (*BotMaid, error) {
	bm := &BotMaid{
		Bots: map[string]*Bot{},
		Conf: &botMaidConfig{
			Log: true,
		},
		Flags:    map[string]*pflag.FlagSet{},
		respTime: time.Now(),
	}

	conf, err := toml.LoadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("Init botmaid: Read config: %v", err)
	}

	if f, ok := conf.Get("Log.Log").(bool); ok {
		bm.Conf.Log = f
	}

	if ss, ok := conf.Get("Command.Prefix").([]interface{}); ok {
		for _, v := range ss {
			if s, ok := v.(string); ok {
				bm.Conf.CommandPrefix = append(bm.Conf.CommandPrefix, s)
			}
		}
	} else {
		bm.Conf.CommandPrefix = []string{"/"}
	}

	if conf.Has("Redis") {
		bm.Conf.Redis.Address = "127.0.0.1"
		if s, ok := conf.Get("Redis.Address").(string); ok {
			bm.Conf.Redis.Address = s
		}
		if s, ok := conf.Get("Redis.Password").(string); ok {
			bm.Conf.Redis.Password = s
		}
		if a, ok := conf.Get("Redis.Database").(int64); ok {
			bm.Conf.Redis.Database = int(a)
		}
	}

	if bm.Conf.Redis.Address != "" {
		bm.Redis = redis.NewClient(&redis.Options{
			Addr:     bm.Conf.Redis.Address,
			Password: bm.Conf.Redis.Password,
			DB:       bm.Conf.Redis.Database,
		})
	}

	for _, v := range conf.Keys() {
		if strings.HasPrefix(v, "Bot_") {
			if conf.Get(v) == nil {
				break
			}

			if conf.Get(v+".Type") == nil {
				return nil, fmt.Errorf("Init botmaid: Missing type of %v", v)
			}

			err := bm.readBotConfig(conf, v)
			if err != nil {
				return nil, fmt.Errorf("Read config: %v", err)
			}
		}
	}

	bm.Words = map[string][]string{
		"selfIntro": []string{
			fmt.Sprintf(`%%v is a bot.

Usage:

        %v(%v)*COMMAND* [ARGUMENTS]

The commands are:
%%v

Use "help [COMMAND] for more information about a command."`, bm.Conf.CommandPrefix[0], ListToString(bm.Conf.CommandPrefix[1:], "%v", ", ", " or ")),
		},
		"undefCommand": []string{
			"%v, the command \"%v\" is unknown, please check the spelling or the \"help\" command of this bot and retry.",
		},
		"unregMaster": []string{
			"%v, the master %v has been unregistered.",
		},
		"regMaster": []string{
			"%v, the user %v has been registered as master.",
		},
		"unbanUser": []string{
			"%v, the user %v has been unbanned.",
		},
		"banUser": []string{
			"%v, the user %v has been banned.",
		},
		"noPermission": []string{
			"%v, you don't have permission to use the command \"%v\".",
		},
		"invalidParameters": []string{
			"%v, the parameters of the command \"%v\" is invalid.",
		},
		"noHelpText": []string{
			"%v, the command \"%v\" has no help text.",
		},
		"invalidUser": []string{
			"%v, the user \"%v\" is invalid or not exist.",
		},
		"helpHelp": []string{
			"display help menus",
		},
		"masterHelp": []string{
			"add/remove masters",
		},
		"banHelp": []string{
			"ban/unban users",
		},
	}

	return bm, nil
}

// Start starts the BotMaid.
func (bm *BotMaid) Start() error {
	err := bm.Redis.Ping().Err()
	if err != nil {
		return fmt.Errorf("Init botmaid: Connect Redis: %v", err)
	}

	bm.initCommand()
	bm.startBot()
	bm.loadTimers()

	select {}
}
