package app

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/zorcal/its-a-me-zorcal/pkg/httprouter"
	"github.com/zorcal/its-a-me-zorcal/pkg/session"
)

func commandHandler(sessionMgr *session.Manager[terminalSessionEntry]) httprouter.Handler {
	tmpl, err := template.ParseFS(templatesFS, "templates/command_output.html")
	if err != nil {
		return func(w http.ResponseWriter, r *http.Request) error {
			return fmt.Errorf("parse template fs for command handler: %w", err)
		}
	}

	return func(w http.ResponseWriter, r *http.Request) error {
		if err := r.ParseForm(); err != nil {
			return wrapHTTPError(http.StatusBadRequest, "Bad form data", err)
		}

		sessionID := getSessionID(r)
		sess := sessionMgr.GetOrCreateSession(sessionID)

		command := strings.TrimSpace(r.FormValue("command"))

		var output string
		switch command {
		case "":
			// Empty command shows just the prompt
		case "cd":
			return newHTTPError(http.StatusBadRequest, "Not implemented yet.")
		case "ls":
			return newHTTPError(http.StatusBadRequest, "Not implemented yet.")
		case "pwd":
			return newHTTPError(http.StatusBadRequest, "Not implemented yet.")
		case "open":
			return newHTTPError(http.StatusBadRequest, "Not implemented yet.")
		case "cat":
			return newHTTPError(http.StatusBadRequest, "Not implemented yet.")
		case "clear":
			runClearCommand(w, sess)
			return nil
		case "help":
			output = `<div class="help">Available commands:

  <strong>ls</strong>        - List directory contents
                   "d" = directory, "o" = openable file, "c" = catable file
  <strong>cd</strong>        - Change directory
  <strong>pwd</strong>       - Print working directory
  <strong>cat</strong>       - Display file contents
  <strong>open</strong>      - Open files (projects, links)
  <strong>clear</strong>     - Clear terminal history (or use Ctrl+L)
  <strong>help</strong>      - Show this help message

Navigation:
  • Use Ctrl+L to clear the terminal

</div>`
		default:
			output = fmt.Sprintf("shell: %s: command not found...", command)
		}

		entry := newTerminalSessionEntry(command, template.HTML(output), false)
		sess.AddEntry(entry)

		data := struct {
			Command string
			Output  template.HTML
			Error   bool
		}{
			Command: command,
			Output:  template.HTML(output),
			Error:   false,
		}

		if err := tmpl.Execute(w, data); err != nil {
			return fmt.Errorf("exec template: %w", err)
		}

		return nil
	}
}

func runClearCommand(w http.ResponseWriter, sess *session.Session[terminalSessionEntry]) {
	sess.ClearHistory()
	w.Header().Set("HX-Retarget", "#command-output")
	w.Header().Set("HX-Reswap", "innerHTML")
	w.Write([]byte(""))
}
