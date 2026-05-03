package web

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/mayor-wren/porygon/database"
)

func (server *Server) handleCommandsList(w http.ResponseWriter, r *http.Request) {
	commands, err := database.GetAllCommands(server.db)
	if err != nil {
		slog.Error("failed to list commands", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	server.renderTemplate(w, "commands.html", map[string]any{
		"Commands": commands,
	})
}

func (server *Server) handleCommandNew(w http.ResponseWriter, r *http.Request) {
	server.renderTemplate(w, "command_form.html", map[string]any{
		"IsNew":        true,
		"TriggerValue": "command",
	})
}

func (server *Server) handleCommandCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form data", http.StatusBadRequest)
		return
	}
	name := strings.TrimPrefix(strings.TrimSpace(r.FormValue("name")), "!")
	response := strings.TrimSpace(r.FormValue("response"))
	trigger := r.FormValue("trigger")
	aliasString := strings.TrimSpace(r.FormValue("aliases"))

	if name == "" || response == "" {
		http.Error(w, "name and response are required", http.StatusBadRequest)
		return
	}
	if trigger != "command" && trigger != "keyword" {
		trigger = "command"
	}

	formData := map[string]any{
		"IsNew":         true,
		"NameValue":     name,
		"ResponseValue": response,
		"TriggerValue":  trigger,
		"AliasesValue":  aliasString,
	}

	if trigger == "command" {
		if strings.ContainsAny(name, " \t") {
			formData["Error"] = "Command name cannot contain spaces."
			server.renderTemplate(w, "command_form.html", formData)
			return
		}
		aliases := parseAliasString(aliasString)
		for _, alias := range aliases {
			if strings.Contains(alias, " ") {
				formData["Error"] = "Aliases cannot contain spaces."
				server.renderTemplate(w, "command_form.html", formData)
				return
			}
		}
	}

	commandID, err := database.CreateCommand(server.db, name, response, trigger)
	if err != nil {
		if isUniqueConstraintError(err) {
			formData["Error"] = "A command with that name already exists."
			server.renderTemplate(w, "command_form.html", formData)
			return
		}
		slog.Error("failed to create command", "error", err)
		http.Error(w, "failed to create command", http.StatusInternalServerError)
		return
	}

	var aliases []string
	if trigger != "keyword" {
		aliases = parseAliasString(aliasString)
	}
	if err := database.SetAliases(server.db, int(commandID), aliases); err != nil {
		database.DeleteCommand(server.db, int(commandID))
		if isUniqueConstraintError(err) {
			formData["Error"] = "One of those aliases is already in use."
			server.renderTemplate(w, "command_form.html", formData)
			return
		}
		slog.Error("failed to set aliases", "error", err)
		http.Error(w, "failed to set aliases", http.StatusInternalServerError)
		return
	}

	if err := server.bot.ReloadCommands(); err != nil {
		slog.Error("failed to reload commands after create", "error", err)
	}
	http.Redirect(w, r, "/commands", http.StatusFound)
}

func (server *Server) handleCommandEdit(w http.ResponseWriter, r *http.Request) {
	commandID, err := pathID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	command, err := database.GetCommand(server.db, commandID)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "command not found", http.StatusNotFound)
		return
	}
	if err != nil {
		slog.Error("failed to get command", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	server.renderTemplate(w, "command_form.html", map[string]any{
		"IsNew":         false,
		"Command":       command,
		"NameValue":     command.Name,
		"ResponseValue": command.Response,
		"TriggerValue":  command.Trigger,
		"AliasesValue":  strings.Join(command.Aliases, ", "),
		"EnabledValue":  command.Enabled,
	})
}

func (server *Server) handleCommandUpdate(w http.ResponseWriter, r *http.Request) {
	commandID, err := pathID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form data", http.StatusBadRequest)
		return
	}
	name := strings.TrimPrefix(strings.TrimSpace(r.FormValue("name")), "!")
	response := strings.TrimSpace(r.FormValue("response"))
	trigger := r.FormValue("trigger")
	enabled := r.FormValue("enabled") == "on"
	aliasString := strings.TrimSpace(r.FormValue("aliases"))

	if name == "" || response == "" {
		http.Error(w, "name and response are required", http.StatusBadRequest)
		return
	}
	if trigger != "command" && trigger != "keyword" {
		trigger = "command"
	}

	formData := map[string]any{
		"IsNew":         false,
		"Command":       &database.Command{ID: commandID},
		"NameValue":     name,
		"ResponseValue": response,
		"TriggerValue":  trigger,
		"AliasesValue":  aliasString,
		"EnabledValue":  enabled,
	}

	if trigger == "command" {
		if strings.ContainsAny(name, " \t") {
			formData["Error"] = "Command name cannot contain spaces."
			server.renderTemplate(w, "command_form.html", formData)
			return
		}
		aliases := parseAliasString(aliasString)
		for _, alias := range aliases {
			if strings.Contains(alias, " ") {
				formData["Error"] = "Aliases cannot contain spaces."
				server.renderTemplate(w, "command_form.html", formData)
				return
			}
		}
	}

	if err := database.UpdateCommand(server.db, commandID, name, response, trigger, enabled); err != nil {
		if isUniqueConstraintError(err) {
			formData["Error"] = "A command with that name already exists."
			server.renderTemplate(w, "command_form.html", formData)
			return
		}
		slog.Error("failed to update command", "error", err)
		http.Error(w, "failed to update command", http.StatusInternalServerError)
		return
	}

	var aliases []string
	if trigger != "keyword" {
		aliases = parseAliasString(aliasString)
	}
	if err := database.SetAliases(server.db, commandID, aliases); err != nil {
		if isUniqueConstraintError(err) {
			formData["Error"] = "One of those aliases is already in use."
			server.renderTemplate(w, "command_form.html", formData)
			return
		}
		slog.Error("failed to set aliases", "error", err)
		http.Error(w, "failed to set aliases", http.StatusInternalServerError)
		return
	}

	if err := server.bot.ReloadCommands(); err != nil {
		slog.Error("failed to reload commands after update", "error", err)
	}
	http.Redirect(w, r, "/commands", http.StatusFound)
}

func (server *Server) handleCommandDelete(w http.ResponseWriter, r *http.Request) {
	commandID, err := pathID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := database.DeleteCommand(server.db, commandID); err != nil {
		slog.Error("failed to delete command", "error", err)
		http.Error(w, "failed to delete command", http.StatusInternalServerError)
		return
	}

	if err := server.bot.ReloadCommands(); err != nil {
		slog.Error("failed to reload commands after delete", "error", err)
	}
	http.Redirect(w, r, "/commands", http.StatusFound)
}

func parseAliasString(input string) []string {
	parts := strings.Split(input, ",")
	var aliases []string
	for _, part := range parts {
		trimmed := strings.TrimPrefix(strings.TrimSpace(part), "!")
		if trimmed != "" {
			aliases = append(aliases, trimmed)
		}
	}
	return aliases
}

func isUniqueConstraintError(err error) bool {
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}
