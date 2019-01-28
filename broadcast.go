package botmaid

import (
	"fmt"
)

// DBBroadcastChat is a struct saved some information of the chat to
// broadcast.
type DBBroadcastChat struct {
	ID       int64
	BotID    string
	ChatType string
	ChatID   int64
}

// InitBroadcastTable creates a table with the standard structure of a
// broadcast.
func (bm *BotMaid) InitBroadcastTable(tableName string) error {
	stmt, err := bm.DB.Prepare(`CREATE TABLE ` + tableName + ` (
		id SERIAL primary key,
		bot_id text,
		chat_type text,
		chat_id bigint not null
	)`)
	if err != nil {
		return fmt.Errorf("Init broadcast table: %v", err)
	}

	stmt.Exec()

	return nil
}

// Broadcast sends an update to all chats in the table.
func (bm *BotMaid) Broadcast(tableName string, m *Message) {
	rows, err := bm.DB.Query("SELECT * FROM " + tableName)
	if err != nil {
		return
	}
	defer rows.Close()

	dbChats := []DBBroadcastChat{}

	for rows.Next() {
		theChat := DBBroadcastChat{}
		err := rows.Scan(&theChat.ID, &theChat.BotID, &theChat.ChatType, &theChat.ChatID)
		if err != nil {
			return
		}
		dbChats = append(dbChats, theChat)
	}

	for _, v := range dbChats {
		if _, ok := bm.Bots[v.BotID]; !ok {
			continue
		}

		bm.Bots[v.BotID].API.Send(Update{
			Message: m,
			Chat: &Chat{
				Type: v.ChatType,
				ID:   v.ChatID,
			},
		})
	}
}

// SwitchBroadcast switches the broadcast on/off of a chat.
func (bm *BotMaid) SwitchBroadcast(tableName string, chat *Chat, b *Bot) {
	theChat := DBBroadcastChat{}
	err := bm.DB.QueryRow("SELECT * FROM "+tableName+" WHERE bot_id = $1 AND chat_type = $2 AND chat_id = $3", b.ID, chat.Type, chat.ID).Scan(&theChat.ID, &theChat.BotID, &theChat.ChatType, &theChat.ChatID)
	if err != nil {
		stmt, _ := bm.DB.Prepare("INSERT INTO " + tableName + "(bot_id, chat_type, chat_id) VALUES($1, $2, $3)")
		stmt.Exec(b.ID, chat.Type, chat.ID)
	} else {
		stmt, _ := bm.DB.Prepare("DELETE FROM " + tableName + " WHERE bot_id = $1 AND chat_type = $2 AND chat_id = $3")
		stmt.Exec(b.ID, chat.Type, chat.ID)
	}
}
