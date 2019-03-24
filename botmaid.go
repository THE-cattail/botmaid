// Package botmaid is a package includes more useful public functions for bots.
package botmaid

import (
	"database/sql"
	"fmt"
	"log"
	"sort"
	"strconv"
	"time"

	"github.com/catsworld/random"

	"github.com/pelletier/go-toml"
)

// BotMaid includes a slice of Bot and some methods to use them.
type BotMaid struct {
	Bots map[string]*Bot

	Conf *toml.Tree

	DB *sql.DB

	Commands []Command
	Timers   []Timer

	HelpMenus []HelpMenu

	Words map[string][]string

	RespTime time.Time
}

func (bm *BotMaid) addMaster(u *Update, b *Bot) bool {
	args := SplitCommand(u.Message.Text)
	if b.IsCommand(u, "addmaster") && len(args) == 2 {
		theMaster := dbMaster{}
		err := bm.DB.QueryRow("SELECT * FROM masters WHERE bot_id = $1 AND user_id = $2", b.ID, args[1]).Scan(&theMaster.ID, &theMaster.BotID, &theMaster.UserID)
		if err == nil {
			b.Reply(u, fmt.Sprintf(random.String(bm.Words["masterExisted"]), args[1]))
			return true
		}

		stmt, _ := bm.DB.Prepare("INSERT INTO masters(bot_id, user_id) VALUES($1, $2)")
		stmt.Exec(b.ID, args[1])
		b.Reply(u, fmt.Sprintf(random.String(bm.Words["masterAdded"]), args[1]))

		return true
	}

	return false
}

func (bm *BotMaid) removeMaster(u *Update, b *Bot) bool {
	args := SplitCommand(u.Message.Text)
	if b.IsCommand(u, "rmmaster") && len(args) == 2 {
		theMaster := dbMaster{}
		err := bm.DB.QueryRow("SELECT * FROM masters WHERE bot_id = $1 AND user_id = $2", b.ID, args[1]).Scan(&theMaster.ID, &theMaster.BotID, &theMaster.UserID)
		if err != nil {
			b.Reply(u, fmt.Sprintf(random.String(bm.Words["masterNotExisted"]), args[1]))
			return true
		}

		stmt, _ := bm.DB.Prepare("DELETE FROM masters WHERE bot_id = $1 AND user_id = $2")
		stmt.Exec(b.ID, args[1])
		b.Reply(u, fmt.Sprintf(random.String(bm.Words["masterRemoved"]), args[1]))

		return true
	}

	return false
}

func (bm *BotMaid) switchTestChat(u *Update, b *Bot) bool {
	args := SplitCommand(u.Message.Text)
	if b.IsCommand(u, "test") && len(args) == 1 {
		theTestChat := dbTestChat{}
		err := bm.DB.QueryRow("SELECT * FROM testchats WHERE bot_id = $1 AND chat_type = $2 AND chat_id = $3", b.ID, u.Chat.Type, u.Chat.ID).Scan(&theTestChat.ID, &theTestChat.BotID, &theTestChat.ChatType, &theTestChat.ChatID)
		if err != nil {
			stmt, _ := bm.DB.Prepare("INSERT INTO testchats(bot_id, chat_type, chat_id) VALUES($1, $2, $3)")
			stmt.Exec(b.ID, u.Chat.Type, u.Chat.ID)
			b.Reply(u, random.String(bm.Words["testChatAdded"]))
		} else {
			stmt, _ := bm.DB.Prepare("DELETE FROM testchats WHERE bot_id = $1 AND chat_type = $2 AND chat_id = $3")
			stmt.Exec(b.ID, u.Chat.Type, u.Chat.ID)
			b.Reply(u, random.String(bm.Words["testChatRemoved"]))
		}

		return true
	}

	return false
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
		Do:     bm.addMaster,
		Names:  []string{"addmaster"},
		Help:   " <@某人> - 将某人设为 Master",
		Master: true,
	})
	bm.AddCommand(Command{
		Do:     bm.removeMaster,
		Names:  []string{"rmmaster"},
		Help:   " <@某人> - 取消某人的 Master 资格",
		Master: true,
	})
	bm.AddCommand(Command{
		Do:     bm.switchTestChat,
		Names:  []string{"test"},
		Help:   " - 切换本场景的测试开关",
		Master: true,
	})
	bm.AddCommand(Command{
		Do:     bm.status,
		Names:  []string{"status"},
		Help:   " - 查看 Bot 状态",
		Master: true,
	})

	sort.Stable(CommandSlice(bm.Commands))
}

func (bm *BotMaid) initDatabase() error {
	stmt, err := bm.DB.Prepare(`CREATE TABLE masters (
		id SERIAL primary key,
		bot_id text,
		user_id bigint not null
	)`)
	if err != nil {
		return fmt.Errorf("Init botmaid database: %v", err)
	}
	stmt.Exec()

	stmt, err = bm.DB.Prepare(`CREATE TABLE testchats (
		id SERIAL primary key,
		bot_id text,
		chat_type text,
		chat_id bigint not null
	)`)
	if err != nil {
		return fmt.Errorf("Init botmaid database: %v", err)
	}
	stmt.Exec()

	return nil
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
				theMaster := dbMaster{}
				err := bm.DB.QueryRow("SELECT * FROM masters WHERE bot_id = $1 AND user_id = $2", b.ID, v.(int64)).Scan(&theMaster.ID, &theMaster.BotID, &theMaster.UserID)
				if err != nil {
					stmt, _ := bm.DB.Prepare("INSERT INTO masters(bot_id, user_id) VALUES($1, $2)")
					stmt.Exec(b.ID, v.(int64))
				}
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
					if c.Check == nil || c.Check(&u, bm.Bots[section]) {
						if c.Do(&u, bm.Bots[section]) {
							break
						}
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

				timer := time.NewTimer(-time.Since(next))
				<-timer.C
				v.Do()

				if v.Frequency == 0 || (v.End != time.Time{} && next.After(v.End)) {
					break
				}
			}
		}(v)
	}
}

// Start starts the BotMaid.
func (bm *BotMaid) Start() error {
	bm.initCommand()

	err := bm.initDatabase()
	if err != nil {
		return fmt.Errorf("Init botmaid: %v", err)
	}

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
