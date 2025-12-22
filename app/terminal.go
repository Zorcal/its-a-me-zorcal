package app

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/zorcal/its-a-me-zorcal/pkg/httprouter"
)

func commandHandler() httprouter.Handler {
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

		command := strings.TrimSpace(r.FormValue("command"))

		var output string
		switch command {
		case "":
			// Empty command shows just the prompt
		case "ls":
			output = "README.md  src/  docs/  package.json"
		case "clear":
			// Clear command resets the terminal - set swap to replace command output
			w.Header().Set("HX-Retarget", "#command-output")
			w.Header().Set("HX-Reswap", "innerHTML")
			w.Write([]byte(""))
			return nil
		case "test_error":
			return newHTTPError(http.StatusBadRequest, "some error")
		default:
			output = fmt.Sprintf("shell: %s: command not found...", command)
		}

		data := struct {
			Command string
			Output  string
			Error   bool
		}{
			Command: command,
			Output:  output,
			Error:   false,
		}

		if err := tmpl.Execute(w, data); err != nil {
			return fmt.Errorf("exec template: %w", err)
		}

		return nil
	}
}
