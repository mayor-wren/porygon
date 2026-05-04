package database

import (
	"testing"
)

func TestCreateAndGetCommand(t *testing.T) {
	db := setupTestDB(t)
	defer db.close()

	commandID, err := CreateCommand(db.DB, "hello", "Hello there!", "command", 0, false)
	if err != nil {
		t.Fatalf("failed to create command: %v", err)
	}

	command, err := GetCommand(db.DB, int(commandID))
	if err != nil {
		t.Fatalf("failed to get command: %v", err)
	}

	if command.Name != "hello" {
		t.Errorf("expected name 'hello', got '%s'", command.Name)
	}
	if command.Response != "Hello there!" {
		t.Errorf("expected response 'Hello there!', got '%s'", command.Response)
	}
	if command.Trigger != "command" {
		t.Errorf("expected trigger 'command', got '%s'", command.Trigger)
	}
	if !command.Enabled {
		t.Error("expected command to be enabled by default")
	}
}

func TestGetAllCommands(t *testing.T) {
	db := setupTestDB(t)
	defer db.close()

	CreateCommand(db.DB, "hello", "Hello!", "command", 0, false)
	CreateCommand(db.DB, "goodbye", "Goodbye!", "command", 0, false)
	CreateCommand(db.DB, "lurk", "is lurking", "keyword", 0, false)

	commands, err := GetAllCommands(db.DB)
	if err != nil {
		t.Fatalf("failed to get all commands: %v", err)
	}

	if len(commands) != 3 {
		t.Fatalf("expected 3 commands, got %d", len(commands))
	}
}

func TestUpdateCommand(t *testing.T) {
	db := setupTestDB(t)
	defer db.close()

	commandID, _ := CreateCommand(db.DB, "hello", "Hello!", "command", 0, false)

	err := UpdateCommand(db.DB, int(commandID), "hello", "Updated response!", "keyword", 0, false, false)
	if err != nil {
		t.Fatalf("failed to update command: %v", err)
	}

	command, _ := GetCommand(db.DB, int(commandID))
	if command.Response != "Updated response!" {
		t.Errorf("expected updated response, got '%s'", command.Response)
	}
	if command.Trigger != "keyword" {
		t.Errorf("expected trigger 'keyword', got '%s'", command.Trigger)
	}
	if command.Enabled {
		t.Error("expected command to be disabled")
	}
}

func TestDeleteCommand(t *testing.T) {
	db := setupTestDB(t)
	defer db.close()

	commandID, _ := CreateCommand(db.DB, "hello", "Hello!", "command", 0, false)

	err := DeleteCommand(db.DB, int(commandID))
	if err != nil {
		t.Fatalf("failed to delete command: %v", err)
	}

	_, err = GetCommand(db.DB, int(commandID))
	if err == nil {
		t.Error("expected error when getting deleted command")
	}
}

func TestSetAndGetAliases(t *testing.T) {
	db := setupTestDB(t)
	defer db.close()

	commandID, _ := CreateCommand(db.DB, "song", "Now playing: ...", "command", 0, false)

	err := SetAliases(db.DB, int(commandID), []string{"music", "nowplaying", "np"})
	if err != nil {
		t.Fatalf("failed to set aliases: %v", err)
	}

	aliases, err := GetAliases(db.DB, int(commandID))
	if err != nil {
		t.Fatalf("failed to get aliases: %v", err)
	}

	if len(aliases) != 3 {
		t.Fatalf("expected 3 aliases, got %d", len(aliases))
	}

	// Aliases are ordered alphabetically
	if aliases[0] != "music" || aliases[1] != "nowplaying" || aliases[2] != "np" {
		t.Errorf("unexpected aliases: %v", aliases)
	}
}

func TestSetAliasesReplacesExisting(t *testing.T) {
	db := setupTestDB(t)
	defer db.close()

	commandID, _ := CreateCommand(db.DB, "song", "Now playing", "command", 0, false)

	SetAliases(db.DB, int(commandID), []string{"music", "np"})
	SetAliases(db.DB, int(commandID), []string{"track"})

	aliases, _ := GetAliases(db.DB, int(commandID))
	if len(aliases) != 1 {
		t.Fatalf("expected 1 alias after replacement, got %d", len(aliases))
	}
	if aliases[0] != "track" {
		t.Errorf("expected alias 'track', got '%s'", aliases[0])
	}
}

func TestDeleteCommandCascadesAliases(t *testing.T) {
	db := setupTestDB(t)
	defer db.close()

	commandID, _ := CreateCommand(db.DB, "song", "Now playing", "command", 0, false)
	SetAliases(db.DB, int(commandID), []string{"music", "np"})

	DeleteCommand(db.DB, int(commandID))

	aliases, err := GetAliases(db.DB, int(commandID))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(aliases) != 0 {
		t.Errorf("expected 0 aliases after cascade delete, got %d", len(aliases))
	}
}

func TestCommandAliasesLoadedWithGetCommand(t *testing.T) {
	db := setupTestDB(t)
	defer db.close()

	commandID, _ := CreateCommand(db.DB, "song", "Now playing", "command", 0, false)
	SetAliases(db.DB, int(commandID), []string{"music", "np"})

	command, err := GetCommand(db.DB, int(commandID))
	if err != nil {
		t.Fatalf("failed to get command: %v", err)
	}

	if len(command.Aliases) != 2 {
		t.Fatalf("expected 2 aliases on command, got %d", len(command.Aliases))
	}
}
