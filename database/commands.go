package database

import (
	"database/sql"
	"time"
)

type Command struct {
	ID         int
	Name       string
	Response   string
	Trigger    string
	Cooldown   int
	MatchStart bool
	Enabled    bool
	Aliases    []string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func GetAllCommands(db *sql.DB) ([]Command, error) {
	rows, err := db.Query(`SELECT id, name, response, trigger, cooldown, match_start, enabled, created_at, updated_at FROM commands ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var commands []Command
	for rows.Next() {
		var command Command
		if err := rows.Scan(&command.ID, &command.Name, &command.Response, &command.Trigger, &command.Cooldown, &command.MatchStart, &command.Enabled, &command.CreatedAt, &command.UpdatedAt); err != nil {
			return nil, err
		}
		commands = append(commands, command)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Bulk-load all aliases in one query instead of N+1
	aliasRows, err := db.Query(`SELECT command_id, alias FROM command_aliases ORDER BY alias`)
	if err != nil {
		return nil, err
	}
	defer aliasRows.Close()

	aliasesByCommand := make(map[int][]string)
	for aliasRows.Next() {
		var commandID int
		var alias string
		if err := aliasRows.Scan(&commandID, &alias); err != nil {
			return nil, err
		}
		aliasesByCommand[commandID] = append(aliasesByCommand[commandID], alias)
	}
	if err := aliasRows.Err(); err != nil {
		return nil, err
	}

	for i := range commands {
		commands[i].Aliases = aliasesByCommand[commands[i].ID]
	}

	return commands, nil
}

func GetCommand(db *sql.DB, id int) (*Command, error) {
	var command Command
	err := db.QueryRow(`SELECT id, name, response, trigger, cooldown, match_start, enabled, created_at, updated_at FROM commands WHERE id = ?`, id).
		Scan(&command.ID, &command.Name, &command.Response, &command.Trigger, &command.Cooldown, &command.MatchStart, &command.Enabled, &command.CreatedAt, &command.UpdatedAt)
	if err != nil {
		return nil, err
	}

	aliases, err := GetAliases(db, command.ID)
	if err != nil {
		return nil, err
	}
	command.Aliases = aliases

	return &command, nil
}

func CreateCommand(db *sql.DB, name, response, trigger string, cooldown int, matchStart bool) (int64, error) {
	result, err := db.Exec(`INSERT INTO commands (name, response, trigger, cooldown, match_start) VALUES (?, ?, ?, ?, ?)`, name, response, trigger, cooldown, matchStart)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func UpdateCommand(db *sql.DB, id int, name, response, trigger string, cooldown int, matchStart bool, enabled bool) error {
	_, err := db.Exec(`UPDATE commands SET name = ?, response = ?, trigger = ?, cooldown = ?, match_start = ?, enabled = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		name, response, trigger, cooldown, matchStart, enabled, id)
	return err
}

func DeleteCommand(db *sql.DB, id int) error {
	_, err := db.Exec(`DELETE FROM commands WHERE id = ?`, id)
	return err
}

func GetAliases(db *sql.DB, commandID int) ([]string, error) {
	rows, err := db.Query(`SELECT alias FROM command_aliases WHERE command_id = ? ORDER BY alias`, commandID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var aliases []string
	for rows.Next() {
		var alias string
		if err := rows.Scan(&alias); err != nil {
			return nil, err
		}
		aliases = append(aliases, alias)
	}
	return aliases, rows.Err()
}

func SetAliases(db *sql.DB, commandID int, aliases []string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM command_aliases WHERE command_id = ?`, commandID); err != nil {
		return err
	}
	for _, alias := range aliases {
		if alias == "" {
			continue
		}
		if _, err := tx.Exec(`INSERT INTO command_aliases (command_id, alias) VALUES (?, ?)`, commandID, alias); err != nil {
			return err
		}
	}
	return tx.Commit()
}
