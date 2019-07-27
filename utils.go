package main

import (
	"encoding/json"
)

type ankiDeck struct {
	Name string `json:"name"`
}

func getAnkiDeckID(deckName string) string {
	row := db.QueryRow(
		`SELECT decks FROM col`,
	)
	var decksInfo string
	row.Scan(&decksInfo)
	if decksInfo == "" {
		return ""
	}
	var f map[string]ankiDeck
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

func getWordsByAnkiDeckID(deckID string) []string {
	rows, err := db.Query(
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
