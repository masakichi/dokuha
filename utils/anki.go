package utils

import (
	"database/sql"
	"encoding/json"
)

var AnkiDB *sql.DB

type AnkiDeck struct {
	Name string `json:"name"`
}

func GetAnkiDeckID(deckName string) string {
	row := AnkiDB.QueryRow(
		`SELECT decks FROM col`,
	)
	var decksInfo string
	row.Scan(&decksInfo)
	if decksInfo == "" {
		return ""
	}
	var f map[string]AnkiDeck
	if err := json.Unmarshal([]byte(decksInfo), &f); err != nil {
		panic(err)
	}
	for k, v := range f {
		if v.Name == deckName {
			return k
		}
	}
	return ""
}

func GetWordsByAnkiDeckID(deckID string) []string {
	rows, err := AnkiDB.Query(
		`SELECT DISTINCT(n.sfld)
		 FROM notes n
		 LEFT JOIN cards c on n.id = c.nid
		 WHERE c.did = ?`,
		deckID)
	if err != nil {
		return []string{}
	}
	defer rows.Close()
	result := []string{}
	for rows.Next() {
		var word string
		err := rows.Scan(&word)
		if err == nil {
			result = append(result, word)
		}
	}
	return result
}
