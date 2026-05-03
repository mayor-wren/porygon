package database

import (
	"database/sql"
	"testing"
)

func TestCreateAndGetQuote(t *testing.T) {
	db := setupTestDB(t)
	defer db.close()

	quoteID, err := CreateQuote(db.DB, "This is a great quote!", "testuser")
	if err != nil {
		t.Fatalf("failed to create quote: %v", err)
	}

	quote, err := GetQuote(db.DB, int(quoteID))
	if err != nil {
		t.Fatalf("failed to get quote: %v", err)
	}

	if quote.Text != "This is a great quote!" {
		t.Errorf("expected text 'This is a great quote!', got '%s'", quote.Text)
	}
	if quote.AddedBy != "testuser" {
		t.Errorf("expected added_by 'testuser', got '%s'", quote.AddedBy)
	}
}

func TestGetAllQuotes(t *testing.T) {
	db := setupTestDB(t)
	defer db.close()

	CreateQuote(db.DB, "Quote one", "user1")
	CreateQuote(db.DB, "Quote two", "user2")
	CreateQuote(db.DB, "Quote three", "user3")

	quotes, err := GetAllQuotes(db.DB)
	if err != nil {
		t.Fatalf("failed to get all quotes: %v", err)
	}

	if len(quotes) != 3 {
		t.Fatalf("expected 3 quotes, got %d", len(quotes))
	}
}

func TestGetRandomQuote(t *testing.T) {
	db := setupTestDB(t)
	defer db.close()

	CreateQuote(db.DB, "Only quote", "user1")

	quote, err := GetRandomQuote(db.DB)
	if err != nil {
		t.Fatalf("failed to get random quote: %v", err)
	}

	if quote.Text != "Only quote" {
		t.Errorf("expected 'Only quote', got '%s'", quote.Text)
	}
}

func TestGetRandomQuoteEmpty(t *testing.T) {
	db := setupTestDB(t)
	defer db.close()

	_, err := GetRandomQuote(db.DB)
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestUpdateQuote(t *testing.T) {
	db := setupTestDB(t)
	defer db.close()

	quoteID, _ := CreateQuote(db.DB, "Original text", "user1")

	err := UpdateQuote(db.DB, int(quoteID), "Updated text")
	if err != nil {
		t.Fatalf("failed to update quote: %v", err)
	}

	quote, _ := GetQuote(db.DB, int(quoteID))
	if quote.Text != "Updated text" {
		t.Errorf("expected 'Updated text', got '%s'", quote.Text)
	}
}

func TestDeleteQuote(t *testing.T) {
	db := setupTestDB(t)
	defer db.close()

	quoteID, _ := CreateQuote(db.DB, "Delete me", "user1")

	err := DeleteQuote(db.DB, int(quoteID))
	if err != nil {
		t.Fatalf("failed to delete quote: %v", err)
	}

	_, err = GetQuote(db.DB, int(quoteID))
	if err == nil {
		t.Error("expected error when getting deleted quote")
	}
}

func TestQuoteIDsAreNotReused(t *testing.T) {
	db := setupTestDB(t)
	defer db.close()

	id1, _ := CreateQuote(db.DB, "Quote 1", "user1")
	DeleteQuote(db.DB, int(id1))
	id2, _ := CreateQuote(db.DB, "Quote 2", "user1")

	if id2 <= id1 {
		t.Errorf("expected new quote ID (%d) to be greater than deleted ID (%d)", id2, id1)
	}
}
