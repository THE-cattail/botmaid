package botmaid

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// APICqhttp is a struct stores some basic information of the CQHTTP. Please search in CQHTTP document for details.
type APICqhttp struct {
	AccessToken string
	Secret      string
	APIEndpoint string
}

var (
	retDescCqhttp = map[int]string{
		0:     "Succeeded",
		1:     "Entered asynchronous execution",
		100:   "Missing or invalid parameters",
		102:   "Invalid return data of CQHTTP",
		103:   "Operation failed",
		104:   "Provided invalidation certificate from CQHTTP",
		201:   "Worker thread pool is not properly initialized",
		10100: "Terminated by other request because of conflict",
	}
)

// API returns the body of an HTTP response to the CQHTTP.
func (a *APICqhttp) API(end string, m map[string]interface{}) (interface{}, error) {
	url := fmt.Sprintf(a.APIEndpoint, end, a.AccessToken)

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

	if _, ok := ret["status"]; !ok {
		return nil, fmt.Errorf("API %v: Unsuccessful request", end)
	}

	if ret["status"].(string) == "failed" {
		if s, ok := retDescCqhttp[int(ret["retcode"].(float64))]; ok {
			return nil, fmt.Errorf("API %v: %v", end, s)
		}
		return nil, fmt.Errorf("API %v: %v", end, ret["retcode"].(float64))
	}

	return ret["data"], nil
}

func (a *APICqhttp) mapToUpdates(m []interface{}) ([]*Update, error) {
	us := []*Update{}
	for _, v := range m {
		e := v.(map[string]interface{})

		update := &Update{}

		if e["post_type"].(string) == "message" {
			update = &Update{
				ID: int64(e["message_id"].(float64)),

				Type: "message_text",

				Time: time.Unix(int64(e["time"].(float64)), 0),

				Chat: &Chat{
					Type: e["message_type"].(string),
				},

				User: &User{
					ID: int64(e["user_id"].(float64)),
				},

				Message: &Message{
					ID:      int64(e["message_id"].(float64)),
					Content: e["raw_message"].(string),
				},
			}

			update.User.UserName = strconv.FormatInt(update.User.ID, 10)

			if update.Chat.Type == "private" {
				update.Chat.ID = int64(e["user_id"].(float64))
			} else if update.Chat.Type == "group" {
				update.Chat.ID = int64(e["group_id"].(float64))
			} else if update.Chat.Type == "discuss" {
				update.Chat.ID = int64(e["discuss_id"].(float64))
			}

			if update.Chat.Type == "group" {
				m, err := a.API("get_group_list", map[string]interface{}{})
				if err != nil {
					return []*Update{}, fmt.Errorf("Get updates: %v", err)
				}

				gs := m.([]interface{})
				for _, v := range gs {
					g := v.(map[string]interface{})
					if int64(g["group_id"].(float64)) == update.Chat.ID {
						update.Chat.Title = g["group_name"].(string)
						break
					}
				}

				u := e["sender"].(map[string]interface{})
				update.User.NickName = u["nickname"].(string)
				if _, ok := u["card"]; ok && u["card"].(string) != "" {
					update.User.NickName = u["card"].(string)
				}
			} else {
				u := e["sender"].(map[string]interface{})
				update.User.NickName = u["nickname"].(string)
			}
		} else {
			continue
		}

		us = append(us, update)
	}
	return us, nil
}

// Pull pulls updates and errors into the channels with a given config.
func (a *APICqhttp) Pull(pc *PullConfig) (UpdateChannel, ErrorChannel) {
	updates := make(chan *Update)
	errors := make(chan error)

	go func() {
		for {
			m, err := a.API("get_updates", map[string]interface{}{
				"limit":   pc.Limit,
				"timeout": pc.Timeout,
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
func (a *APICqhttp) Push(update *Update) (*Update, error) {
	if update.Type == "Delete" {
		_, err := a.API("delete_msg", map[string]interface{}{
			"message_id": update.ID,
		})
		if err != nil {
			return nil, fmt.Errorf("Delete message: %v", err)
		}

		return nil, nil
	}

	m := map[string]interface{}{
		"message_type": update.Chat.Type,
	}

	if update.Chat.Type == "private" {
		m["user_id"] = update.Chat.ID
	}
	if update.Chat.Type == "group" {
		m["group_id"] = update.Chat.ID
	}
	if update.Chat.Type == "discuss" {
		m["discuss_id"] = update.Chat.ID
	}

	message := ""

	if update.Message.Type == "Audio" {
		if strings.HasPrefix(update.Message.Content, "http://") || strings.HasPrefix(update.Message.Content, "https://") {
			message += fmt.Sprintf("[CQ:record,file=%v]", update.Message.Content)
		} else {
			file, err := ioutil.ReadFile(update.Message.Content)
			if err != nil {
				return nil, fmt.Errorf("Read audio file: %v", err)
			}
			message += fmt.Sprintf("[CQ:record,file=base64://%v]", base64.StdEncoding.EncodeToString(file))
		}
	} else if update.Message.Type == "Image" || update.Message.Type == "Sticker" {
		if strings.HasPrefix(update.Message.Content, "http://") || strings.HasPrefix(update.Message.Content, "https://") {
			message += fmt.Sprintf("[CQ:image,file=%v]", update.Message.Content)
		} else {
			file, err := ioutil.ReadFile(update.Message.Content)
			if err != nil {
				return nil, fmt.Errorf("Read image file: %v", err)
			}
			message += fmt.Sprintf("[CQ:image,file=base64://%v]", base64.StdEncoding.EncodeToString(file))
		}
	} else {
		message += update.Message.Content
	}

	m["message"] = message

	msg, err := a.API("send_msg", m)
	if err != nil {
		return nil, fmt.Errorf("Send message: %v", err)
	}

	update.ID = int64(msg.(map[string]interface{})["message_id"].(float64))

	return update, nil
}

// Platform returns a string showing the platform of the bot.
func (a *APICqhttp) Platform() string {
	return "QQ"
}

// ParseUserID parses the ID of the User in the At string.
func (a *APICqhttp) ParseUserID(u *Update, s string) (int64, error) {
	if strings.HasPrefix(s, "[CQ:at,qq=") && strings.HasSuffix(s, "]") {
		start := 10
		end := strings.LastIndex(s, "]")
		s = s[start:end]

		id, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("Invalid At string: %v", err)
		}
		return id, nil
	}

	return 0, errors.New("Invalid At string")
}

func (a *APICqhttp) ats(u *User) []string {
	return []string{fmt.Sprintf("[CQ:at,qq=%v]", u.ID)}
}
