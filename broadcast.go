package botmaid

import (
	"fmt"

	"github.com/catsworld/api"
)

// DBBroadcastPlace is a struct saved some information of the place to
// broadcast.
type DBBroadcastPlace struct {
	ID        int64
	BotID     string
	PlaceType string
	PlaceID   int64
}

// InitBroadcastTable creates a table with the standard structure of a
// broadcast.
func (bm *BotMaid) InitBroadcastTable(tableName string) error {
	stmt, err := bm.DB.Prepare(`CREATE TABLE ` + tableName + ` (
		id SERIAL primary key,
		bot_id text,
		place_type text,
		place_id bigint not null
	)`)
	if err != nil {
		return fmt.Errorf("Init broadcast table: %v", err)
	}

	stmt.Exec()

	return nil
}

// Broadcast pushes an event to all places in the table.
func (bm *BotMaid) Broadcast(tableName string, m *api.Message) {
	rows, err := bm.DB.Query("SELECT * FROM " + tableName)
	if err != nil {
		return
	}
	defer rows.Close()

	dbPlaces := []DBBroadcastPlace{}

	for rows.Next() {
		thePlace := DBBroadcastPlace{}
		err := rows.Scan(&thePlace.ID, &thePlace.BotID, &thePlace.PlaceType, &thePlace.PlaceID)
		if err != nil {
			return
		}
		dbPlaces = append(dbPlaces, thePlace)
	}

	for _, v := range dbPlaces {
		if _, ok := bm.Bots[v.BotID]; !ok {
			continue
		}

		bm.Bots[v.BotID].API.Push(api.Event{
			Message: m,
			Place: &api.Place{
				Type: v.PlaceType,
				ID:   v.PlaceID,
			},
		})
	}
}

// SwitchBroadcast switches the broadcast on/off of a place.
func (bm *BotMaid) SwitchBroadcast(tableName string, b *Bot, place *api.Place) {
	thePlace := DBBroadcastPlace{}
	err := bm.DB.QueryRow("SELECT * FROM "+tableName+" WHERE bot_id = $1 AND place_type = $2 AND place_id = $3", b.ID, place.Type, place.ID).Scan(&thePlace.ID, &thePlace.BotID, &thePlace.PlaceType, &thePlace.PlaceID)
	if err != nil {
		stmt, _ := bm.DB.Prepare("INSERT INTO " + tableName + "(bot_id, place_type, place_id) VALUES($1, $2, $3)")
		stmt.Exec(b.ID, place.Type, place.ID)
	} else {
		stmt, _ := bm.DB.Prepare("DELETE FROM " + tableName + " WHERE bot_id = $1 AND place_type = $2 AND place_id = $3")
		stmt.Exec(b.ID, place.Type, place.ID)
	}
}
