// Package botmaid is a package includes more useful public functions for bots.
package botmaid

import (
	"database/sql"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/catsworld/api"
	"github.com/catsworld/cqhttp"
	"github.com/catsworld/random"
	"github.com/catsworld/tgbot"

	"github.com/pelletier/go-toml"
)

// Bot includes some information of a bot.
type Bot struct {
	ID string

	API api.API

	Self *api.User

	BotMaid *BotMaid
}

type dbMaster struct {
	ID       int64
	BotID    string
	UserName string
}

type dbTestPlace struct {
	ID        int64
	BotID     string
	PlaceType string
	PlaceID   int64
}

// IsMaster checks if a user is master of the bot.
func (b *Bot) IsMaster(u api.User) bool {
	m := dbMaster{}
	err := b.BotMaid.DB.QueryRow("SELECT * FROM masters WHERE bot_id = $1 AND username = $2", b.ID, u.UserName).Scan(&m.ID, &m.BotID, &m.UserName)
	if err != nil {
		return false
	}
	return true
}

// IsTestPlace checks if a place is test place of the bot.
func (b *Bot) IsTestPlace(p api.Place) bool {
	t := dbTestPlace{}
	err := b.BotMaid.DB.QueryRow("SELECT * FROM testplaces WHERE bot_id = $1 AND place_type = $2 AND place_id = $3", b.ID, p.Type, p.ID).Scan(&t.ID, &t.BotID, &t.PlaceType, &t.PlaceID)
	if err != nil {
		return false
	}
	return true
}

// Platform returns a string showing the platform of the bot.
func (b *Bot) Platform() string {
	switch b.API.(type) {
	case *cqhttp.API:
		return "QQ"
	case *tgbot.API:
		return "Telegram"
	}

	return "Unknown Platform"
}

// At returns a string to mention someone in a message.
func (b *Bot) At(u *api.User) []string {
	switch b.API.(type) {
	case *cqhttp.API:
		return []string{fmt.Sprintf("[CQ:at,qq=%s]", u.UserName), fmt.Sprintf("@%s", u.NickName)}
	case *tgbot.API:
		return []string{fmt.Sprintf("@%s", u.UserName)}
	}

	return []string{fmt.Sprintf("@%s", u.UserName)}
}

// BeAt checks if a message of an event is mentioning the bot.
func (b *Bot) BeAt(e *api.Event) bool {
	switch b.API.(type) {
	case *cqhttp.API:
		if (strings.Contains(e.Message.Text, fmt.Sprintf("[CQ:at,qq=%s]", b.Self.UserName)) || strings.Contains(e.Message.Text, fmt.Sprintf("@%s", b.Self.NickName))) && b.extractCommand(e) == "" {
			return true
		}
	case *tgbot.API:
		if strings.Contains(e.Message.Text, fmt.Sprintf("@%s", b.Self.UserName)) && b.extractCommand(e) == "" {
			return true
		}
	}

	return false
}

// UserNameFromAt returns the UserName of the user in the mention query.
func (b *Bot) UserNameFromAt(s string) string {
	switch b.API.(type) {
	case *cqhttp.API:
		if fmt.Sprintf("[CQ:at,qq=%s]", s[10:len(s)-1]) == s {
			return s[10 : len(s)-1]
		}
	case *tgbot.API:
		if fmt.Sprintf("@%s", s[1:]) == s {
			return s[1:]
		}
	}

	return ""
}

// BotMaid includes a slice of Bot and some methods to use them.
type BotMaid struct {
	Bots map[string]*Bot

	Conf *toml.Tree

	DB *sql.DB

	Commands []Command
	Timers   []Timer

	HelpMenus map[string]string

	Words map[string][]string

	RespTime time.Time
}

func (bm *BotMaid) addMaster(e *api.Event, b *Bot) bool {
	args := SplitCommand(e.Message.Text)
	if b.IsCommand(e, "addmaster") && len(args) == 2 {
		if b.UserNameFromAt(args[1]) == "" {
			b.API.Push(api.Event{
				Message: &api.Message{
					Text: fmt.Sprintf(random.String(bm.Words["invalidMaster"]), args[1]),
				},
				Place: e.Place,
			})
			return true
		}

		theMaster := dbMaster{}
		err := bm.DB.QueryRow("SELECT * FROM masters WHERE bot_id = $1 AND username = $2", b.ID, b.UserNameFromAt(args[1])).Scan(&theMaster.ID, &theMaster.BotID, &theMaster.UserName)
		if err == nil {
			b.API.Push(api.Event{
				Message: &api.Message{
					Text: fmt.Sprintf(random.String(bm.Words["masterExisted"]), args[1]),
				},
				Place: e.Place,
			})
			return true
		}

		stmt, _ := bm.DB.Prepare("INSERT INTO masters(bot_id, username) VALUES($1, $2)")
		stmt.Exec(b.ID, b.UserNameFromAt(args[1]))
		b.API.Push(api.Event{
			Message: &api.Message{
				Text: fmt.Sprintf(random.String(bm.Words["masterAdded"]), args[1]),
			},
			Place: e.Place,
		})

		return true
	}

	return false
}

func (bm *BotMaid) removeMaster(e *api.Event, b *Bot) bool {
	args := SplitCommand(e.Message.Text)
	if b.IsCommand(e, "rmmaster") && len(args) == 2 {
		if b.UserNameFromAt(args[1]) == "" {
			b.API.Push(api.Event{
				Message: &api.Message{
					Text: fmt.Sprintf(random.String(bm.Words["invalidMaster"]), args[1]),
				},
				Place: e.Place,
			})
			return true
		}

		theMaster := dbMaster{}
		err := bm.DB.QueryRow("SELECT * FROM masters WHERE bot_id = $1 AND username = $2", b.ID, b.UserNameFromAt(args[1])).Scan(&theMaster.ID, &theMaster.BotID, &theMaster.UserName)
		if err != nil {
			b.API.Push(api.Event{
				Message: &api.Message{
					Text: fmt.Sprintf(random.String(bm.Words["masterNotExisted"]), args[1]),
				},
				Place: e.Place,
			})
			return true
		}

		stmt, _ := bm.DB.Prepare("DELETE FROM masters WHERE bot_id = $1 AND username = $2")
		stmt.Exec(b.ID, b.UserNameFromAt(args[1]))
		b.API.Push(api.Event{
			Message: &api.Message{
				Text: fmt.Sprintf(random.String(bm.Words["masterRemoved"]), args[1]),
			},
			Place: e.Place,
		})

		return true
	}

	return false
}

func (bm *BotMaid) switchTestPlace(e *api.Event, b *Bot) bool {
	args := SplitCommand(e.Message.Text)
	if b.IsCommand(e, "test") && len(args) == 1 {
		theTestPlace := dbTestPlace{}
		err := bm.DB.QueryRow("SELECT * FROM testplaces WHERE bot_id = $1 AND place_type = $2 AND place_id = $3", b.ID, e.Place.Type, e.Place.ID).Scan(&theTestPlace.ID, &theTestPlace.BotID, &theTestPlace.PlaceType, &theTestPlace.PlaceID)
		if err != nil {
			stmt, _ := bm.DB.Prepare("INSERT INTO testplaces(bot_id, place_type, place_id) VALUES($1, $2, $3)")
			stmt.Exec(b.ID, e.Place.Type, e.Place.ID)
			b.API.Push(api.Event{
				Message: &api.Message{
					Text: random.String(bm.Words["testPlaceAdded"]),
				},
				Place: e.Place,
			})
		} else {
			stmt, _ := bm.DB.Prepare("DELETE FROM testplaces WHERE bot_id = $1 AND place_type = $2 AND place_id = $3")
			stmt.Exec(b.ID, e.Place.Type, e.Place.ID)
			b.API.Push(api.Event{
				Message: &api.Message{
					Text: random.String(bm.Words["testPlaceRemoved"]),
				},
				Place: e.Place,
			})
		}

		return true
	}

	return false
}

// Start starts the BotMaid.
func (bm *BotMaid) Start() error {
	var err error

	bm.AddCommand(Command{
		Do:       bm.help,
		Priority: 10000,
	})
	bm.AddCommand(Command{
		Do:       bm.help2,
		Priority: -10000,
	})
	bm.AddCommand(Command{
		Do:       bm.addMaster,
		Priority: 5,
		Names:    []string{"addmaster"},
		Help:     " <@某人> - 将某人设为 Master",
		Master:   true,
	})
	bm.AddCommand(Command{
		Do:       bm.removeMaster,
		Priority: 5,
		Names:    []string{"rmmaster"},
		Help:     " <@某人> - 取消某人的 Master 资格",
		Master:   true,
	})
	bm.AddCommand(Command{
		Do:       bm.switchTestPlace,
		Priority: 5,
		Names:    []string{"test"},
		Help:     " - 切换本场景的测试开关",
		Master:   true,
	})

	sort.Stable(CommandSlice(bm.Commands))

	stmt, err := bm.DB.Prepare(`CREATE TABLE masters (
		id SERIAL primary key,
		bot_id text,
		username text
	)`)
	if err != nil {
		return err
	}

	stmt.Exec()

	stmt, err = bm.DB.Prepare(`CREATE TABLE testplaces (
		id SERIAL primary key,
		bot_id text,
		place_type text,
		place_id bigint not null
	)`)
	if err != nil {
		return err
	}

	stmt.Exec()

	for i := 1; ; i++ {
		section := "Bot_" + strconv.Itoa(i)

		if bm.Conf.Get(section) == nil {
			break
		}

		if bm.Conf.Get(section+".Type") == nil {
			return fmt.Errorf("Init botmaid: Missing type of %v", section)
		}

		go func(section string) {
			botType := bm.Conf.Get(section + ".Type").(string)

			b := Bot{
				ID:      section,
				BotMaid: bm,
			}

			if botType == "QQ" {
				q := &cqhttp.API{}

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
					b.Self = &api.User{
						ID:       int64(u["user_id"].(float64)),
						UserName: strconv.FormatInt(int64(u["user_id"].(float64)), 10),
						NickName: u["nickname"].(string),
					}

					break
				}

				b.API = q
			} else if botType == "Telegram" {
				t := &tgbot.API{}

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
					b.Self = &api.User{
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
				log.Printf("Init botmaid: Unknown type of %s\n", section)
				return
			}

			bm.Bots[section] = &b

			events, errors := b.API.Pull(api.PullConfig{
				Limit:            100,
				Timeout:          60,
				RetryWaitingTime: time.Second * 3,
			})

			go func() {
				for err := range errors {
					log.Printf("Bot running: %v.\n", err)
				}
			}()

			log.Printf("%s (%s) has been loaded. Begin to pull events.\n", b.Self.NickName, b.Platform())

			for e := range events {
				go func(e api.Event) {
					if !e.Time.After(bm.RespTime) {
						return
					}

					if bm.Conf.Get("Test.Test") != nil {
						if _, ok := bm.Conf.Get("Test.Test").(bool); !ok {
							log.Println("Bot running: Expected but not Test as a boolean in Test.")
							return
						}
					}

					if e.Message == nil {
						return
					}

					if !bm.Conf.Get("Test.Test").(bool) || (b.IsTestPlace(*e.Place) && b.BeAt(&e)) {
						logText := e.Message.Text

						if e.Sender != nil {
							logText = e.Sender.NickName + "(@" + e.Sender.UserName + "):" + logText
						}

						if e.Place != nil && e.Place.Title != "" {
							logText = "[" + e.Place.Title + "]" + logText
						}

						log.Println(logText)

						for _, c := range bm.Commands {
							if !b.IsMaster(*e.Sender) && c.Master {
								continue
							}
							if !b.IsTestPlace(*e.Place) && c.Test {
								continue
							}
							if c.Do(&e, bm.Bots[section]) {
								break
							}
						}
					}
				}(e)
			}
		}(section)
	}

	for _, v := range bm.Timers {
		if v.Frequency == "once" && time.Now().After(v.Time) {
			continue
		}

		go func(v Timer) {
			for {
				if v.Frequency == "daily" {
					for time.Now().After(v.Time) {
						v.Time = v.Time.AddDate(0, 0, 1)
					}
				} else if v.Frequency == "weekly" {
					for time.Now().After(v.Time) {
						v.Time = v.Time.AddDate(0, 0, 7)
					}
				} else if v.Frequency == "monthly" {
					for time.Now().After(v.Time) {
						v.Time = v.Time.AddDate(0, 1, 0)
					}
				} else if v.Frequency == "yearly" {
					for time.Now().After(v.Time) {
						v.Time = v.Time.AddDate(1, 0, 0)
					}
				}

				timer := time.NewTimer(-time.Since(v.Time))
				<-timer.C
				v.Do()

				if v.Frequency == "once" {
					break
				}
			}
		}(v)
	}

	select {}
}
