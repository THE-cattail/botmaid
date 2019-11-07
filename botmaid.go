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
	Bots      map[string]*Bot
	Conf      *botMaidConfig
	Redis     *redis.Client
	Commands  CommandSlice
	Timers    []Timer
	HelpMenus []HelpMenu
	Words     map[string][]string
	RespTime  time.Time
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

func (bm *BotMaid) initCommand() {
	bm.AddCommand(&Command{
		SetFlag: func(u *Update) {
		},
		Do:       bm.help,
		Priority: 10000,
	})

	bm.AddCommand(&Command{
		SetFlag: func(u *Update) {
		},
		Do:       bm.help2,
		Priority: -10000,
	})

	bm.AddCommand(&Command{
		SetFlag: func(u *Update) {
			u.Message.Flag.BoolP("del", "d", false, "")
			u.Message.Flag.BoolP("add", "a", false, "")
		},
		Do: func(u *Update) bool {
			if len(u.Message.Flag.Args()) != 2 {
				return false
			}

			id, err := bm.ParseUserID(u, u.Message.Flag.Args()[1])
			if err != nil {
				Reply(u, fmt.Sprintf(random.String(bm.Words["invalidUser"]), u.Message.Flag.Args()[1]))
				return true
			}

			is := bm.Redis.SIsMember("master_"+u.Bot.ID, id).Val()

			del, _ := u.Message.Flag.GetBool("del")
			if del || is {
				bm.Redis.SRem("master_"+u.Bot.ID, id)
				Reply(u, fmt.Sprintf(random.String(bm.Words["unregMaster"]), u.Message.Flag.Args()[1]))
				return true
			}

			add, _ := u.Message.Flag.GetBool("add")
			if add || !is {
				bm.Redis.SAdd("master_"+u.Bot.ID, id)
				Reply(u, fmt.Sprintf(random.String(bm.Words["regMaster"]), u.Message.Flag.Args()[1]))
			}

			return false
		},
		Names:  []string{"master"},
		Master: true,
	})

	bm.AddCommand(&Command{
		SetFlag: func(u *Update) {
			u.Message.Flag.BoolP("del", "d", false, "")
			u.Message.Flag.BoolP("add", "a", false, "")
		},
		Do: func(u *Update) bool {
			if len(u.Message.Flag.Args()) != 2 {
				return false
			}

			id, err := bm.ParseUserID(u, u.Message.Flag.Args()[1])
			if err != nil {
				Reply(u, fmt.Sprintf(random.String(bm.Words["invalidUser"])))
				return true
			}

			is := bm.Redis.SIsMember("ban_"+u.Bot.ID, id).Val()

			del, _ := u.Message.Flag.GetBool("del")
			if del || is {
				bm.Redis.SRem("ban_"+u.Bot.ID, id)
				Reply(u, fmt.Sprintf(random.String(bm.Words["unbanUser"]), u.Message.Flag.Args()[1]))
				return true
			}

			add, _ := u.Message.Flag.GetBool("add")
			if add || !is {
				bm.Redis.SAdd("ban_"+u.Bot.ID, id)
				Reply(u, fmt.Sprintf(random.String(bm.Words["banUser"]), u.Message.Flag.Args()[1]))
			}

			return false
		},
		Names:  []string{"ban"},
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
					if u.Bot.Platform() == "Telegram" && u.User.UserName != "" {
						bm.Redis.HSet("telegramUsers", fmt.Sprintf("%v", u.User.UserName), u.User.ID)
					}

					u.Bot = b
					if u.User != nil {
						u.User.Bot = b
					}

					if !u.Time.After(bm.RespTime) {
						return
					}

					if u.Message == nil {
						return
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
						Reply(u, fmt.Sprintf(random.String(bm.Words["invalidParameters"])), u.Message.Text)
						return
					}
					u.Message.Args = args

					u.Message.Command = bm.extractCommand(u)

					for _, c := range bm.Commands {
						if !bm.IsMaster(u.User) && c.Master {
							continue
						}
						if len(c.Names) != 0 && !IsCommand(u, c.Names) {
							continue
						}
						if c.ArgsMinLen != 0 && len(u.Message.Flag.Args()) < c.ArgsMinLen {
							continue
						}
						if c.ArgsMaxLen != 0 && len(u.Message.Flag.Args()) > c.ArgsMaxLen {
							continue
						}

						u.Message.Flag = &pflag.FlagSet{}
						c.SetFlag(u)
						u.Message.Flag.Parse(args)

						if c.Do(u) {
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
		RespTime: time.Now(),
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
			fmt.Sprintf("%%v, Please use %v to call this bot.", ListToString(bm.Conf.CommandPrefix, "\"%v\"", ", ", " or ")),
		},
		"undefCommand": []string{
			"Unknown command %v.",
		},
		"unregMaster": []string{
			"The master %v has been unregistered.",
		},
		"regMaster": []string{
			"The user %v has been registered as master.",
		},
		"unbanUser": []string{
			"The user %v has been unbanned.",
		},
		"banUser": []string{
			"The user %v has been banned.",
		},
		"noPermission": []string{
			"%v, you don't have permission to use the command %v.",
		},
		"invalidParameters": []string{
			"The parameters of the command \"%v\" is invalid.",
		},
		"noHelpText": []string{
			"The command \"%v\" has no help text.",
		},
		"invalidUser": []string{
			"The user \"%v\" is invalid or not exist.",
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
