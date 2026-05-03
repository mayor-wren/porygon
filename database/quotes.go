package database

import (
	"database/sql"
	"time"
)

type Quote struct {
	ID        int
	Text      string
	AddedBy   string
	CreatedAt time.Time
}

func GetAllQuotes(db *sql.DB) ([]Quote, error) {
	rows, err := db.Query(`SELECT id, text, added_by, created_at FROM quotes ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var quotes []Quote
	for rows.Next() {
		var quote Quote
		if err := rows.Scan(&quote.ID, &quote.Text, &quote.AddedBy, &quote.CreatedAt); err != nil {
			return nil, err
		}
		quotes = append(quotes, quote)
	}
	return quotes, rows.Err()
}

func GetQuote(db *sql.DB, id int) (*Quote, error) {
	var quote Quote
	err := db.QueryRow(`SELECT id, text, added_by, created_at FROM quotes WHERE id = ?`, id).
		Scan(&quote.ID, &quote.Text, &quote.AddedBy, &quote.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &quote, nil
}

func GetRandomQuote(db *sql.DB) (*Quote, error) {
	var quote Quote
	err := db.QueryRow(`SELECT id, text, added_by, created_at FROM quotes ORDER BY RANDOM() LIMIT 1`).
		Scan(&quote.ID, &quote.Text, &quote.AddedBy, &quote.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &quote, nil
}

func CreateQuote(db *sql.DB, text, addedBy string) (int64, error) {
	result, err := db.Exec(`INSERT INTO quotes (text, added_by) VALUES (?, ?)`, text, addedBy)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func UpdateQuote(db *sql.DB, id int, text string) error {
	_, err := db.Exec(`UPDATE quotes SET text = ? WHERE id = ?`, text, id)
	return err
}

func DeleteQuote(db *sql.DB, id int) error {
	_, err := db.Exec(`DELETE FROM quotes WHERE id = ?`, id)
	return err
}
