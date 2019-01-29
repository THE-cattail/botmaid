package botmaid

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// CoolqHTTPAPI is a struct stores some basic information of the Coolq HTTP
// API. Please search in Coolq HTTP API document for details.
type CoolqHTTPAPI struct {
	AccessToken string
	Secret      string
	APIEndpoint string
}

var (
	cqhttpRetDesc = map[int]string{
		0:     "Succeeded",
		1:     "Entered asynchronous execution",
		100:   "Missing or invalid parameters",
		102:   "Invalid return data of CoolQ",
		103:   "Operation failed",
		104:   "Provided invalidation certificate from Coolq",
		201:   "Worker thread pool is not properly initialized",
		10100: "Terminated by other request because of conflict",
	}
)

// API returns the body of an HTTP response to the Coolq HTTP API.
func (a *CoolqHTTPAPI) API(end string, m map[string]interface{}) (interface{}, error) {
	url := fmt.Sprintf(a.APIEndpoint, end, a.AccessToken)

	j, err := json.Marshal(m)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(j))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API %s: %v", end, err)
	}
	defer resp.Body.Close()

	raw, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("API %s: %v", end, err)
	}

	ret := map[string]interface{}{}
	err = json.Unmarshal(raw, &ret)
	if err != nil {
		return nil, fmt.Errorf("API %s: %v", end, err)
	}

	if _, ok := ret["status"]; !ok {
		return nil, fmt.Errorf("API %s: Unsuccessful request", end)
	}

	if ret["status"].(string) == "failed" {
		if s, ok := cqhttpRetDesc[int(ret["retcode"].(float64))]; ok {
			return nil, fmt.Errorf("API %s: %v", end, s)
		}
		return nil, fmt.Errorf("API %s: %v", end, ret["retcode"].(float64))
	}

	return ret["data"], nil
}

func (a *CoolqHTTPAPI) mapToUpdates(m []interface{}) ([]Update, error) {
	us := []Update{}
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
					ID:   int64(e["message_id"].(float64)),
					Text: e["raw_message"].(string),
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
					return []Update{}, fmt.Errorf("Get updates: %v", err)
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

		us = append(us, *update)
	}
	return us, nil
}

// GetUpdates gets updates and errors into the channels with a given config.
func (a *CoolqHTTPAPI) GetUpdates(pc GetUpdatesConfig) (UpdateChannel, ErrorChannel) {
	updates := make(chan Update)
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

// Send sends an update and returns it back.
func (a *CoolqHTTPAPI) Send(update Update) (Update, error) {
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

	if update.Message.Audio != "" {
		if strings.HasPrefix(update.Message.Audio, "http://") || strings.HasPrefix(update.Message.Audio, "https://") {
			message += fmt.Sprintf("[CQ:record,file=%s", update.Message.Audio)
		} else {
			file, err := ioutil.ReadFile(update.Message.Audio)
			if err != nil {
				return Update{}, fmt.Errorf("Read audio file: %v", err)
			}
			message += fmt.Sprintf("[CQ:record,file=base64://%s]", base64.StdEncoding.EncodeToString(file))
		}
	} else if update.Message.Image != "" {
		if strings.HasPrefix(update.Message.Image, "http://") || strings.HasPrefix(update.Message.Image, "https://") {
			message += fmt.Sprintf("[CQ:image,file=%s", update.Message.Image)
		} else {
			file, err := ioutil.ReadFile(update.Message.Image)
			if err != nil {
				return Update{}, fmt.Errorf("Read image file: %v", err)
			}
			message += fmt.Sprintf("[CQ:image,file=base64://%s]", base64.StdEncoding.EncodeToString(file))
		}
	} else {
		message += update.Message.Text
	}

	m["message"] = message

	msg, err := a.API("send_msg", m)
	if err != nil {
		return Update{}, fmt.Errorf("Send message: %v", err)
	}

	update.ID = int64(msg.(map[string]interface{})["message_id"].(float64))

	return update, nil
}

// Delete deletes a specific update.
func (a *CoolqHTTPAPI) Delete(update Update) error {
	_, err := a.API("delete_msg", map[string]interface{}{
		"message_id": update.ID,
	})
	if err != nil {
		return fmt.Errorf("Delete message: %v", err)
	}

	return nil
}
