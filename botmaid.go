// Package botmaid is a package includes more useful public functions for bots.
package botmaid

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/catsworld/api"
	"github.com/catsworld/cqhttp"
	"github.com/catsworld/tgbot"

	"github.com/pelletier/go-toml"
)

// Bot includes some information of a bot.
type Bot struct {
	API api.API

	Self *api.User

	masters    []int64
	testPlaces []int64
}

// IsMaster checks if a user is master of the bot.
func (b *Bot) IsMaster(u *api.User) bool {
	for _, v := range b.masters {
		if u.ID == v {
			return true
		}
	}
	return false
}

// IsTestPlace checks if a place is test place of the bot.
func (b *Bot) IsTestPlace(c *api.Place) bool {
	for _, v := range b.testPlaces {
		if c.ID == v {
			return true
		}
	}
	return false
}

// At returns a string to mention someone in a message.
func (b *Bot) At(u *api.User) string {
	switch b.API.(type) {
	case *cqhttp.API:
		return fmt.Sprintf("[CQ:at,qq=%s]", u.UserName)
	case *tgbot.API:
		return fmt.Sprintf("@%s", u.UserName)
	}

	return ""
}

// BotMaid includes a slice of Bot and some methods to use them.
type BotMaid struct {
	Bots []Bot
}

// Init reads information of bots from the toml and add them into the manager.
func (bm *BotMaid) Init(conf *toml.Tree) error {
	for i := 1; ; i++ {
		section := "Bot_" + strconv.Itoa(i)

		if conf.Get(section) == nil {
			break
		}

		if conf.Get(section+".Type") == nil {
			return fmt.Errorf("Init botmaid: Missing type of " + section)
		}

		botType := conf.Get(section + ".Type").(string)

		b := &Bot{}

		if conf.Get(section+".Masters") != nil {
			if _, ok := conf.Get(section + ".Masters").([]interface{}); !ok {
				return fmt.Errorf("Init botmaid: Expected but not Masters as a slice in " + section)
			}
			for _, master := range conf.Get(section + ".Masters").([]interface{}) {
				if _, ok := master.(int64); !ok {
					return fmt.Errorf("Init botmaid: Expected but not int64 in Masters in " + section)
				}
				b.masters = append(b.masters, master.(int64))
			}
		}

		if conf.Get(section+".TestPlaces") != nil {
			if _, ok := conf.Get(section + ".TestPlaces").([]interface{}); !ok {
				return fmt.Errorf("Init botmaid: Expected but not TestPlaces as a slice in " + section)
			}
			for _, testPlace := range conf.Get(section + ".TestPlaces").([]interface{}) {
				if _, ok := testPlace.(int64); !ok {
					return fmt.Errorf("Init botmaid: Expected but not int64 in TestPlaces in " + section)
				}
				b.testPlaces = append(b.testPlaces, testPlace.(int64))
			}
		}

		if botType == "QQ" {
			q := &cqhttp.API{}

			if conf.Get(section+".AccessToken") != nil {
				if _, ok := conf.Get(section + ".AccessToken").(string); ok {
					q.AccessToken = conf.Get(section + ".AccessToken").(string)
				}
			}

			if conf.Get(section+".Secret") != nil {
				if _, ok := conf.Get(section + ".Secret").(string); ok {
					q.Secret = conf.Get(section + ".Secret").(string)
				}
			}

			if conf.Get(section+".APIEndpoint") != nil {
				if _, ok := conf.Get(section + ".APIEndpoint").(string); ok {
					q.APIEndpoint = conf.Get(section + ".APIEndpoint").(string)
				}
			}

			m, err := q.API("get_login_info", map[string]interface{}{})
			if err != nil {
				return fmt.Errorf("Init botmaid: %v", err)
			}

			u := m.(map[string]interface{})
			b.Self = &api.User{
				ID:       int64(u["user_id"].(float64)),
				UserName: strconv.FormatInt(int64(u["user_id"].(float64)), 10),
				NickName: u["nickname"].(string),
			}

			b.API = q
		} else if botType == "Telegram" {
			t := &tgbot.API{}

			if conf.Get(section+".Token") != nil {
				if _, ok := conf.Get(section + ".Token").(string); ok {
					t.Token = conf.Get(section + ".Token").(string)
				}
			}

			m, err := t.API("getMe", map[string]interface{}{})
			if err != nil {
				return fmt.Errorf("Init botmaid: %v", err)
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

			b.API = t
		} else {
			return fmt.Errorf("Init botmaid: Unknown type of " + section)
		}

		bm.Bots = append(bm.Bots, *b)
	}

	return nil
}

// Run begins to get updates and run commands.
func (bm *BotMaid) Run(conf *toml.Tree, cs []Command, ts []Timer, respTime time.Time) {
	go func() {
		for _, v := range ts {
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
	}()

	for i := range bm.Bots {
		go func(b *Bot) {
			events, errors := b.API.Pull(&api.PullConfig{
				Limit:            100,
				Timeout:          60,
				RetryWaitingTime: time.Second * 3,
			})

			go func() {
				for err := range errors {
					log.Printf("Bot running: %v.\n", err)
				}
			}()

			for e := range events {
				event := e
				go func(e *api.Event) {
					if !e.Time.After(respTime) {
						return
					}

					if conf.Get("Test.Test") != nil {
						if _, ok := conf.Get("Test.Test").(bool); !ok {
							log.Println("Bot running: Expected but not Test as a boolean in Test.")
							return
						}
					}

					if e == nil || e.Message == nil {
						return
					}

					if (conf.Get("Test.Test") != nil && !conf.Get("Test.Test").(bool)) || (b.IsTestPlace(e.Place) && strings.Contains(e.Message.Text, b.At(b.Self))) {
						logText := e.Message.Text

						if e.Sender != nil {
							logText = e.Sender.NickName + "(@" + e.Sender.UserName + "):" + logText
						}

						if e.Place != nil && e.Place.Title != "" {
							logText = "[" + e.Place.Title + "]" + logText
						}

						log.Println(logText)
					}

					for _, c := range cs {
						if c.Do(e, b) {
							break
						}
					}
				}(&event)
			}
		}(&bm.Bots[i])
	}

	select {}
}
