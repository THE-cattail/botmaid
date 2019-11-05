// Package botmaid is a package for managing bots.
package botmaid

import (
	"errors"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis"
	"github.com/google/shlex"
	"github.com/pelletier/go-toml"
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
		ID:      section,
		BotMaid: bm,
		API:     new(API),
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
		Do:       bm.help,
		Priority: 10000,
	})
	bm.AddCommand(&Command{
		Do:       bm.help2,
		Priority: -10000,
	})
	bm.AddCommand(&Command{
		Do: func(u *Update) bool {
			if u.Bot.BotMaid.Redis.SIsMember("master_"+u.Bot.ID, u.Message.Args[1]).Val() {
				u.Bot.BotMaid.Redis.SRem("master_"+u.Bot.ID, u.Message.Args[1])
				bm.Reply(u, "The master has been unregistered.")
				return true
			}

			u.Bot.BotMaid.Redis.SAdd("master_"+u.Bot.ID, u.Message.Args[1])
			bm.Reply(u, "The user has been registered as master.")
			return true
		},
		Names:      []string{"master"},
		ArgsMinLen: 2,
		ArgsMaxLen: 2,
		Master:     true,
	})
	bm.AddCommand(&Command{
		Do: func(u *Update) bool {
			if u.Bot.BotMaid.Redis.SIsMember("ban_"+u.Bot.ID, u.Message.Args[1]).Val() {
				u.Bot.BotMaid.Redis.SRem("ban_"+u.Bot.ID, u.Message.Args[1])
				bm.Reply(u, "The user has been unbanned.")
				return true
			}

			u.Bot.BotMaid.Redis.SAdd("ban_"+u.Bot.ID, u.Message.Args[1])
			bm.Reply(u, "The user has been banned.")
			return true
		},
		Names:      []string{"ban"},
		ArgsMinLen: 2,
		ArgsMaxLen: 2,
		Master:     true,
	})
	bm.AddCommand(&Command{
		Do: func(u *Update) bool {
			if len(u.Message.Args) == 2 {
				bm.Reply(u, u.Message.Args[1])
				return true
			} else if len(u.Message.Args) == 4 {
				id, err := strconv.ParseInt(u.Message.Args[3], 10, 64)
				if err != nil {
					bm.Reply(u, err.Error())
				}

				(*u.Bot.API).Push(&Update{
					Chat: &Chat{
						ID:   id,
						Type: u.Message.Args[2],
					},
					Message: &Message{
						Text: u.Message.Args[1],
					},
				})
				return true
			}
			return false
		},
		Names:      []string{"send"},
		ArgsMinLen: 2,
		ArgsMaxLen: 4,
		Help:       " <内容> (<发送对象类型> <发送对象 ID>) - 令 Bot 发送消息",
		Master:     true,
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
					if !u.Time.After(bm.RespTime) {
						return
					}

					if u.Message == nil {
						return
					}

					if b.isBanned(u.User) {
						return
					}

					u.Bot = b

					args, err := shlex.Split(u.Message.Text)
					if err == nil {
						u.Message.Args = args
					}
					u.Message.Command = bm.extractCommand(u)

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

					for _, c := range bm.Commands {
						if !b.IsMaster(u.User) && c.Master {
							continue
						}
						if len(c.Names) != 0 && !IsCommand(u, c.Names) {
							continue
						}
						if c.ArgsMinLen != 0 && len(u.Message.Args) < c.ArgsMinLen {
							continue
						}
						if c.ArgsMaxLen != 0 && len(u.Message.Args) > c.ArgsMaxLen {
							continue
						}

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
			section := v

			if conf.Get(section) == nil {
				break
			}

			if conf.Get(section+".Type") == nil {
				return nil, fmt.Errorf("Init botmaid: Missing type of %v", section)
			}

			err := bm.readBotConfig(conf, section)
			if err != nil {
				return nil, fmt.Errorf("Read config: %v", err)
			}
		}
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

// Reply replies a message back.
func (bm *BotMaid) Reply(u *Update, s ...string) (*Update, error) {
	if len(s) < 1 || len(s) > 2 {
		return nil, errors.New("Invalid number of arguments")
	}
	if len(s) == 1 || s[1] == "Text" {
		return (*u.Bot.API).Push(&Update{
			Message: &Message{
				Text: s[0],
			},
			Chat: u.Chat,
		})
	}
	if s[1] == "Image" {
		return (*u.Bot.API).Push(&Update{
			Message: &Message{
				Image: s[0],
			},
			Chat: u.Chat,
		})
	}
	if s[1] == "Audio" {
		return (*u.Bot.API).Push(&Update{
			Message: &Message{
				Audio: s[0],
			},
			Chat: u.Chat,
		})
	}
	return nil, errors.New("Invalid type of message")
}

// In checks if the element is in the slice.
func In(a interface{}, s ...interface{}) bool {
	if len(s) == 1 {
		if _, ok := s[0].([]interface{}); ok {
			for _, v := range s[0].([]interface{}) {
				if v == a {
					return true
				}
			}
			return false
		}
	}

	for _, v := range s {
		if v == a {
			return true
		}
	}
	return false
}

// ListToString convert the list to a string.
func ListToString(list []string, format string, separator string, andWord string) string {
	if len(list) < 1 {
		return ""
	}
	if len(list) == 1 {
		return fmt.Sprintf(format, list[0])
	}
	ret := fmt.Sprintf(format, list[0])
	for i := 1; i < len(list)-1; i++ {
		ret += separator + fmt.Sprintf(format, list[i])
	}
	ret += andWord + fmt.Sprintf(format, list[len(list)-1])
	return ret
}
