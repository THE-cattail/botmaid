// Package botmaid is a package for managing bots.
package botmaid

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/catsworld/random"

	"github.com/go-redis/redis"
	"github.com/pelletier/go-toml"
)

// BotMaid includes a slice of Bot and some methods to use them.
type BotMaid struct {
	Bots map[string]*Bot

	Conf *toml.Tree

	Redis *redis.Client

	Commands []Command
	Timers   []Timer

	HelpMenus []HelpMenu

	Words map[string][]string

	RespTime time.Time
}

func (bm *BotMaid) status(u *Update, b *Bot) bool {
	b.Reply(u, "√")
	return true
}

func (bm *BotMaid) initCommand() {
	bm.AddCommand(Command{
		Do:       bm.help,
		Priority: 10000,
	})
	bm.AddCommand(Command{
		Do:       bm.help2,
		Priority: -10000,
	})
	bm.AddCommand(Command{
		Do: func(u *Update, b *Bot) bool {
			f, _ := b.BotMaid.Redis.SIsMember("master_"+b.ID, u.User.ID).Result()
			if f {
				b.Reply(u, fmt.Sprintf(random.String(bm.Words["masterExisted"]), u.Message.Args[1]))
				return true
			}

			b.BotMaid.Redis.SAdd("master_"+b.ID, u.User.ID)
			b.Reply(u, fmt.Sprintf(random.String(bm.Words["masterAdded"]), u.Message.Args[1]))
			return true
		},
		Names:      []string{"addmaster"},
		ArgsMinLen: 2,
		ArgsMaxLen: 2,
		Help:       " <用户ID> - 将用户设为 Master",
		Master:     true,
	})
	bm.AddCommand(Command{
		Do: func(u *Update, b *Bot) bool {
			f, _ := b.BotMaid.Redis.SIsMember("master_"+b.ID, u.User.ID).Result()
			if !f {
				b.Reply(u, fmt.Sprintf(random.String(bm.Words["masterNotExisted"]), u.Message.Args[1]))
				return true
			}

			b.BotMaid.Redis.SRem("master_"+b.ID, u.User.ID)
			b.Reply(u, fmt.Sprintf(random.String(bm.Words["masterRemoved"]), u.Message.Args[1]))
			return true
		},
		Names:      []string{"rmmaster"},
		ArgsMinLen: 2,
		ArgsMaxLen: 2,
		Help:       " <用户ID> - 取消用户的 Master 资格",
		Master:     true,
	})
	bm.AddCommand(Command{
		Do: func(u *Update, b *Bot) bool {
			f, _ := b.BotMaid.Redis.SIsMember("testchat_"+b.ID, u.Chat.ID).Result()
			if f {
				b.BotMaid.Redis.SRem("master_"+b.ID, u.Chat.ID)
				b.Reply(u, random.String(bm.Words["testChatRemoved"]))
			} else {
				b.BotMaid.Redis.SAdd("master_"+b.ID, u.Chat.ID)
				b.Reply(u, random.String(bm.Words["testChatAdded"]))
			}
			return true
		},
		Names:      []string{"test"},
		ArgsMinLen: 1,
		ArgsMaxLen: 1,
		Help:       " - 切换本场景的测试开关",
		Master:     true,
	})
	bm.AddCommand(Command{
		Do:         bm.status,
		Names:      []string{"status"},
		ArgsMinLen: 1,
		ArgsMaxLen: 1,
		Help:       " - 查看 Bot 状态",
		Master:     true,
	})

	sort.Stable(CommandSlice(bm.Commands))
}

func (bm *BotMaid) readBotConfig(section string) (Bot, error) {
	botType := bm.Conf.Get(section + ".Type").(string)

	b := Bot{
		ID:      section,
		BotMaid: bm,
	}

	if botType == "QQ" {
		q := &CoolqHTTPAPI{}

		if bm.Conf.Get(section+".AccessToken") != nil {
			if _, ok := bm.Conf.Get(section + ".AccessToken").(string); ok {
				q.AccessToken = bm.Conf.Get(section + ".AccessToken").(string)
			}
		}

		if bm.Conf.Get(section+".Secret") != nil {
			if _, ok := bm.Conf.Get(section + ".Secret").(string); ok {
				q.Secret = bm.Conf.Get(section + ".Secret").(string)
			}
		}

		if bm.Conf.Get(section+".APIEndpoint") != nil {
			if _, ok := bm.Conf.Get(section + ".APIEndpoint").(string); ok {
				q.APIEndpoint = bm.Conf.Get(section + ".APIEndpoint").(string)
			}
		}

		for {
			m, err := q.API("get_login_info", map[string]interface{}{})
			if err != nil {
				log.Printf("Init botmaid: %v, retrying...\n", err)
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

		b.API = q
	} else if botType == "Telegram" {
		t := &TelegramBotAPI{}

		if bm.Conf.Get(section+".Token") != nil {
			if _, ok := bm.Conf.Get(section + ".Token").(string); ok {
				t.Token = bm.Conf.Get(section + ".Token").(string)
			}
		}

		for {
			m, err := t.API("getMe", map[string]interface{}{})
			if err != nil {
				log.Printf("Init botmaid: %v, retrying...\n", err)
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
		b.API = t
	} else {
		return Bot{}, fmt.Errorf("Init botmaid: Unknown type of %v", section)
	}

	if bm.Conf.Get(section+".Master") != nil {
		if _, ok := bm.Conf.Get(section + ".Master").([]interface{}); ok {
			for _, v := range bm.Conf.Get(section + ".Master").([]interface{}) {
				bm.Redis.SAdd("master_"+b.ID, v.(int64))
			}
		}
	}

	return b, nil
}

func (bm *BotMaid) startBot(section string) {
	b, err := bm.readBotConfig(section)
	if err != nil {
		log.Fatalf("[Fatal] Start bot: %v\n", err)
		return
	}

	bm.Bots[section] = &b

	updates, errors := b.API.GetUpdates(GetUpdatesConfig{
		Limit:            100,
		Timeout:          60,
		RetryWaitingTime: time.Second * 3,
	})

	go func() {
		for err := range errors {
			log.Printf("Bot running: %v.\n", err)
		}
	}()

	log.Printf("%s (%s) has been loaded. Begin to get updates.\n", b.Self.NickName, b.Platform())

	for u := range updates {
		go func(u Update) {
			if !u.Time.After(bm.RespTime) {
				return
			}

			if bm.Conf.Get("Test.Test") != nil {
				if _, ok := bm.Conf.Get("Test.Test").(bool); !ok {
					log.Println("Bot running: Expected but not Test as a boolean in Test.")
					return
				}
			}

			if u.Message == nil {
				return
			}

			if !bm.Conf.Get("Test.Test").(bool) || (b.IsTestChat(*u.Chat) && b.BeAt(&u)) {
				u.Message.Args = SplitCommand(u.Message.Text)

				logText := u.Message.Text

				if u.User != nil {
					logText = u.User.NickName + ": " + logText
				}

				if u.Chat != nil && u.Chat.Title != "" {
					logText = "[" + u.Chat.Title + "]" + logText
				}

				log.Println(logText)

				for _, c := range bm.Commands {
					if !b.IsMaster(*u.User) && c.Master {
						continue
					}
					if !b.IsTestChat(*u.Chat) && c.Test {
						continue
					}
					if len(c.Names) != 0 && !b.IsCommand(&u, c.Names) {
						continue
					}
					if c.ArgsMinLen != 0 && len(u.Message.Args) < c.ArgsMinLen {
						continue
					}
					if c.ArgsMaxLen != 0 && len(u.Message.Args) > c.ArgsMaxLen {
						continue
					}

					if c.Do(&u, bm.Bots[section]) {
						break
					}
				}
			}
		}(u)
	}
}

func (bm *BotMaid) loadBots() error {
	for i := 1; ; i++ {
		section := "Bot_" + strconv.Itoa(i)

		if bm.Conf.Get(section) == nil {
			break
		}

		if bm.Conf.Get(section+".Type") == nil {
			return fmt.Errorf("Init botmaid: Missing type of %v", section)
		}

		go bm.startBot(section)
	}

	return nil
}

func (bm *BotMaid) loadTimers() {
	for _, v := range bm.Timers {
		next := v.Start

		if v.Frequency == 0 && time.Now().After(next) {
			continue
		}

		go func(v Timer) {
			for {
				for time.Now().After(next) {
					next = next.Add(v.Frequency)
				}

				if !v.End.IsZero() && next.After(v.End) {
					break
				}

				timer := time.NewTimer(-time.Since(next))
				<-timer.C
				v.Do()

				if v.Frequency == 0 {
					break
				}
			}
		}(v)
	}
}

// Start starts the BotMaid.
func (bm *BotMaid) Start() error {
	rootDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return fmt.Errorf("Init botmaid: Get root directory: %v", err)
	}

	raw, err := ioutil.ReadFile(rootDir + "/config.toml")
	if err != nil {
		return fmt.Errorf("Init botmaid: Read config: %v", err)
	}
	bm.Conf, err = toml.Load(string(raw))
	if err != nil {
		return fmt.Errorf("Init botmaid: Read config: %v", err)
	}

	bm.Redis = redis.NewClient(&redis.Options{
		Addr:     bm.Conf.Get("Redis.Address").(string),
		Password: bm.Conf.Get("Redis.Password").(string),
		DB:       bm.Conf.Get("Redis.Database").(int),
	})

	_, err = bm.Redis.Ping().Result()
	if err != nil {
		return fmt.Errorf("Init botmaid: Connect Redis: %v", err)
	}

	bm.initCommand()

	err = bm.loadBots()
	if err != nil {
		return fmt.Errorf("Init botmaid: %v", err)
	}

	bm.loadTimers()

	select {}
}

// In checks if the element is in the slice.
func In(a interface{}, s ...interface{}) bool {
	for _, v := range s {
		if v == a {
			return true
		}
	}
	return false
}
