package botmaid

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/PuerkitoBio/goquery"
)

// APITelegramBot is a struct stores some basic information of the Telegram Bot API. Please search in official API document for details.
type APITelegramBot struct {
	Token  string
	Offset int64
}

const (
	endPointAPITelegramBot = "https://api.telegram.org/bot%v/%v"
)

// API returns the body of an HTTP response to the Telegram Bot API.
func (a *APITelegramBot) API(end string, m map[string]interface{}) (interface{}, error) {
	url := fmt.Sprintf(endPointAPITelegramBot, a.Token, end)

	j, err := json.Marshal(m)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(j))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API %v: %v", end, err)
	}
	defer resp.Body.Close()

	raw, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("API %v: %v", end, err)
	}

	ret := map[string]interface{}{}
	err = json.Unmarshal(raw, &ret)
	if err != nil {
		return nil, fmt.Errorf("API %v: %v", end, err)
	}

	if _, ok := ret["ok"]; !ok {
		return nil, fmt.Errorf("API %v: Unsuccessful request", end)
	}

	if !ret["ok"].(bool) {
		return nil, fmt.Errorf("API %v: %v", end, ret["description"].(string))
	}

	return ret["result"], nil
}

func (a *APITelegramBot) mapToUpdates(m []interface{}) ([]*Update, error) {
	us := []*Update{}
	for _, v := range m {
		e := v.(map[string]interface{})

		update := &Update{}

		if _, ok := e["message"]; ok {
			m := e["message"].(map[string]interface{})
			c := m["chat"].(map[string]interface{})

			if int64(e["update_id"].(float64)) < a.Offset {
				continue
			}

			if int64(e["update_id"].(float64))+1 > a.Offset {
				a.Offset = int64(e["update_id"].(float64)) + 1
			}

			update = &Update{
				ID: int64(e["update_id"].(float64)),

				Type: "message_text",

				Time: time.Unix(int64(m["date"].(float64)), 0),

				Chat: &Chat{
					ID:   int64(c["id"].(float64)),
					Type: c["type"].(string),
				},

				Message: &Message{
					ID: int64(m["message_id"].(float64)),
				},
			}

			if _, ok := c["title"]; ok {
				update.Chat.Title = c["title"].(string)
			}

			if _, ok := m["text"]; ok {
				update.Message.Content = m["text"].(string)
				if _, ok := m["reply_to_message"]; ok {
					r := m["reply_to_message"].(map[string]interface{})
					if _, ok := r["from"]; ok {
						u := r["from"].(map[string]interface{})
						if _, ok := u["username"]; ok {
							update.Message.Content = fmt.Sprintf("@%v", r["from"].(map[string]interface{})["username"].(string)) + " " + update.Message.Content
						}
					}
				}

				if _, ok := m["entities"]; ok {
					es := m["entities"].([]interface{})
					for i := len(es) - 1; i >= 0; i-- {
						e := es[i].(map[string]interface{})
						if e["type"].(string) != "text_mention" {
							continue
						}

						if e["user"] != nil {
							offset := int(e["offset"].(float64))
							length := int(e["length"].(float64))
							user := e["user"].(map[string]interface{})

							nickName := user["first_name"].(string)
							if _, ok := user["last_name"]; ok {
								nickName += " " + user["last_name"].(string)
							}
							strings.ReplaceAll(nickName, "\\", "\\\\")
							strings.ReplaceAll(nickName, "'", "\\'")
							strings.ReplaceAll(nickName, "\"", "\\\"")

							u16 := utf16.Encode([]rune(update.Message.Content))
							t := append(u16[:offset], utf16.Encode([]rune(fmt.Sprintf("\"<a href=\\\"tg://user?id=%v\\\">%v</a>\"", int64(user["id"].(float64)), nickName)))...)
							u16 = append(t, u16[offset+length:]...)
							update.Message.Content = string(utf16.Decode(u16))
						}
					}
				}
			}

			if _, ok := m["sticker"]; ok {
				s := m["sticker"].(map[string]interface{})
				if _, ok := s["emoji"]; ok {
					update.Message.Content = s["emoji"].(string)
				}
			}

			if _, ok := m["from"]; ok {
				f := m["from"].(map[string]interface{})

				update.User = &User{
					ID:       int64(f["id"].(float64)),
					NickName: f["first_name"].(string),
				}

				if _, ok := f["last_name"]; ok {
					update.User.NickName += " " + f["last_name"].(string)
				}

				if _, ok := f["username"]; ok {
					update.User.UserName = f["username"].(string)
				}
			}
		}

		update.Message.Update = update
		update.Chat.Update = update
		update.User.Update = update
		us = append(us, update)
	}
	return us, nil
}

// Pull pulls updates and errors into the channels with a given config.
func (a *APITelegramBot) Pull(pc *PullConfig) (UpdateChannel, ErrorChannel) {
	updates := make(chan *Update)
	errors := make(chan error)

	go func() {
		for {
			m, err := a.API("getUpdates", map[string]interface{}{
				"limit":   pc.Limit,
				"timeout": pc.Timeout,
				"offset":  a.Offset,
			})
			if err != nil {
				errors <- err
				time.Sleep(pc.RetryWaitingTime)
				continue
			}
			us, err := a.mapToUpdates(m.([]interface{}))
			if err != nil {
				errors <- err
				time.Sleep(pc.RetryWaitingTime)
				continue
			}
			for _, u := range us {
				updates <- u
			}
		}
	}()

	return updates, errors
}

// Push pushes an update and returns it back if existing.
func (a *APITelegramBot) Push(update *Update) (*Update, error) {
	if update.Type == "Delete" {
		_, err := a.API("deleteMessage", map[string]interface{}{
			"chat_id":    update.Chat.ID,
			"message_id": update.ID,
		})
		if err != nil {
			return nil, fmt.Errorf("Delete message: %v", err)
		}

		return nil, nil
	}

	if update.Message.Type == "Image" && strings.HasSuffix(update.Message.Content, ".gif") {
		method := fmt.Sprintf(endPointAPITelegramBot, a.Token, "sendAnimation")

		buf := new(bytes.Buffer)
		w := multipart.NewWriter(buf)

		ct := "multipart/form-data; boundary=" + w.Boundary()

		_ = w.WriteField("chat_id", strconv.FormatInt(update.Chat.ID, 10))

		file, err := ioutil.ReadFile(update.Message.Content)
		if err != nil {
			return nil, fmt.Errorf("Send image: API %v: %v", "sendAnimation", err)
		}

		part, err := w.CreateFormFile("animation", filepath.Base(update.Message.Content))
		if err != nil {
			return nil, fmt.Errorf("Send image: API %v: %v", "sendAnimation", err)
		}

		part.Write(file)
		w.Close()

		header := http.Header{}
		header.Add("Content-Type", ct)

		req, err := http.NewRequest("POST", method, buf)
		if err != nil {
			return nil, fmt.Errorf("Send image: API %v: %v", "sendAnimation", err)
		}
		req.Header = header

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("Send image: API %v: %v", "sendAnimation", err)
		}
		defer resp.Body.Close()

		raw, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("Send image: API %v: %v", "sendAnimation", err)
		}

		m := map[string]interface{}{}
		err = json.Unmarshal(raw, &m)
		if err != nil {
			return nil, fmt.Errorf("Send image: API %v: %v", "sendAnimation", err)
		}

		if _, ok := m["ok"]; !ok {
			return nil, fmt.Errorf("Send image: API %v: Unsuccessful request", "sendAnimation")
		}

		if !m["ok"].(bool) {
			return nil, fmt.Errorf("Send image: API %v: %v", "sendAnimation", m["description"].(string))
		}

		update.ID = int64(m["result"].(map[string]interface{})["message_id"].(float64))

		return update, nil
	}

	if update.Message.Type == "Image" || update.Message.Type == "Sticker" {
		api := "sendPhoto"
		para := "photo"
		if update.Message.Type == "Sticker" {
			api = "sendSticker"
			para = "sticker"
		}

		method := fmt.Sprintf(endPointAPITelegramBot, a.Token, api)

		buf := new(bytes.Buffer)
		w := multipart.NewWriter(buf)

		ct := "multipart/form-data; boundary=" + w.Boundary()

		w.WriteField("chat_id", strconv.FormatInt(update.Chat.ID, 10))

		if strings.HasPrefix(update.Message.Content, "http://") || strings.HasPrefix(update.Message.Content, "https://") {
			w.WriteField(para, update.Message.Content)
		} else {
			file, err := ioutil.ReadFile(update.Message.Content)
			if err != nil {
				return nil, fmt.Errorf("Send image: API %v: %v", api, err)
			}

			part, err := w.CreateFormFile(para, filepath.Base(update.Message.Content))
			if err != nil {
				return nil, fmt.Errorf("Send image: API %v: %v", api, err)
			}
			part.Write(file)
		}
		w.Close()

		header := http.Header{}
		header.Add("Content-Type", ct)

		req, err := http.NewRequest("POST", method, buf)
		if err != nil {
			return nil, fmt.Errorf("Send image: API %v: %v", api, err)
		}
		req.Header = header

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("Send image: API %v: %v", api, err)
		}
		defer resp.Body.Close()

		raw, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("Send image: API %v: %v", api, err)
		}

		m := map[string]interface{}{}
		err = json.Unmarshal(raw, &m)
		if err != nil {
			return nil, fmt.Errorf("Send image: API %v: %v", api, err)
		}

		if _, ok := m["ok"]; !ok {
			return nil, fmt.Errorf("Send image: API %v: Unsuccessful request", api)
		}

		if !m["ok"].(bool) {
			return nil, fmt.Errorf("Send image: API %v: %v", api, m["description"].(string))
		}

		update.ID = int64(m["result"].(map[string]interface{})["message_id"].(float64))

		return update, nil
	}

	if update.Message.Type == "Audio" {
		method := fmt.Sprintf(endPointAPITelegramBot, a.Token, "sendVoice")

		buf := new(bytes.Buffer)
		w := multipart.NewWriter(buf)

		ct := "multipart/form-data; boundary=" + w.Boundary()

		w.WriteField("chat_id", strconv.FormatInt(update.Chat.ID, 10))

		if strings.HasPrefix(update.Message.Content, "http://") || strings.HasPrefix(update.Message.Content, "https://") {
			w.WriteField("voice", update.Message.Content)
		} else {
			file, err := ioutil.ReadFile(update.Message.Content)
			if err != nil {
				return nil, fmt.Errorf("Send audio: API %v: %v", "sendVoice", err)
			}

			part, err := w.CreateFormFile("voice", filepath.Base(update.Message.Content))
			if err != nil {
				return nil, fmt.Errorf("Send audio: API %v: %v", "sendVoice", err)
			}

			part.Write(file)
			w.Close()
		}

		header := http.Header{}
		header.Add("Content-Type", ct)

		req, err := http.NewRequest("POST", method, buf)
		if err != nil {
			return nil, fmt.Errorf("Send image: API %v: %v", "sendVoice", err)
		}
		req.Header = header

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("Send image: API %v: %v", "sendVoice", err)
		}
		defer resp.Body.Close()

		raw, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("Send image: API %v: %v", "sendVoice", err)
		}

		m := map[string]interface{}{}
		err = json.Unmarshal(raw, &m)
		if err != nil {
			return nil, fmt.Errorf("Send image: API %v: %v", "sendVoice", err)
		}

		if _, ok := m["ok"]; !ok {
			return nil, fmt.Errorf("Send image: API %v: Unsuccessful request", "sendVoice")
		}

		if !m["ok"].(bool) {
			return nil, fmt.Errorf("Send image: API %v: %v", "sendVoice", m["description"].(string))
		}

		update.ID = int64(m["result"].(map[string]interface{})["message_id"].(float64))

		return update, nil
	}

	msg, err := a.API("sendMessage", map[string]interface{}{
		"chat_id":    update.Chat.ID,
		"text":       strings.TrimSpace(update.Message.Content),
		"parse_mode": "HTML",
	})
	if err != nil {
		return nil, fmt.Errorf("Send text message: %v", err)
	}

	update.ID = int64(msg.(map[string]interface{})["message_id"].(float64))

	return update, nil
}

// Platform returns a string showing the platform of the bot.
func (a *APITelegramBot) Platform() string {
	return "Telegram"
}

// ParseUserID parses the ID of the User in the At string.
func (a *APITelegramBot) ParseUserID(u *Update, s string) (int64, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(s))
	if err == nil {
		s := doc.Find("a").AttrOr("href", "")
		if strings.HasPrefix(s, "tg://user?id=") {
			s = s[13:]

			id, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				return 0, fmt.Errorf("Invalid At string: %v", err)
			}
			return id, nil
		}
	}

	if strings.HasPrefix(s, "@") {
		s = u.Bot.BotMaid.Redis.HGet("telegramUsers", s[1:]).Val()

		id, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("Invalid At string: %v", err)
		}
		return id, nil
	}

	return 0, errors.New("Invalid At string")
}

func (a *APITelegramBot) ats(u *User) []string {
	return []string{fmt.Sprintf("<a href=\"tg://user?id=%v\">%v</a>", u.ID, u.NickName), fmt.Sprintf("@%v", u.UserName)}
}
